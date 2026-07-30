// pubsub_listener is a CLI utility that subscribes to a Google Cloud Pub/Sub
// subscription and logs the payload and metadata of every received message.
//
// Usage:
//
//	go run pubsub_listener.go \
//	    --project   my-gcp-project \
//	    --subscription my-subscription \
//	    [--host localhost] \
//	    [--port 9200]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ── CLI flags ────────────────────────────────────────────────────────────────

type config struct {
	project        string
	subscription   string
	host           string
	port           int
	noAck          bool
	maxOutstanding int
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.project, "project", "", "Google Cloud project ID (required)")
	flag.StringVar(&cfg.subscription, "subscription", "", "Pub/Sub subscription ID (required)")
	flag.StringVar(&cfg.host, "host", "", "Emulator host (e.g. localhost). Leave empty for real GCP.")
	flag.IntVar(&cfg.port, "port", 9200, "Emulator port (only used when --host is set)")
	flag.BoolVar(&cfg.noAck, "no-ack", false, "Nack every message instead of acking it")
	flag.IntVar(&cfg.maxOutstanding, "max-messages", 1000, "Maximum number of outstanding messages")
	flag.Parse()

	if cfg.project == "" || cfg.subscription == "" {
		fmt.Fprintln(os.Stderr, "Error: --project and --subscription are required.")
		flag.Usage()
		os.Exit(1)
	}
	return cfg
}

// ── Payload helpers ───────────────────────────────────────────────────────────

// prettyPayload tries to pretty-print the data as JSON; falls back to raw string.
func prettyPayload(data []byte) string {
	var v any
	if err := json.Unmarshal(data, &v); err == nil {
		b, err := json.MarshalIndent(v, "    ", "  ")
		if err == nil {
			return "    " + string(b)
		}
	}
	// Not JSON — return as plain text, indented
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		lines[i] = "    " + l
	}
	return strings.Join(lines, "\n")
}

// ── Message handler ───────────────────────────────────────────────────────────

func handleMessage(msg *pubsub.Message, noAck bool) {
	receivedAt := time.Now().UTC().Format(time.RFC3339)

	// Attributes
	var attrLines []string
	if len(msg.Attributes) == 0 {
		attrLines = []string{"    (none)"}
	} else {
		for k, v := range msg.Attributes {
			attrLines = append(attrLines, fmt.Sprintf("    %s: %s", k, v))
		}
	}

	publishedAt := "unavailable"
	if !msg.PublishTime.IsZero() {
		publishedAt = msg.PublishTime.UTC().Format(time.RFC3339)
	}

	deliveryAttempt := "N/A"
	if msg.DeliveryAttempt != nil {
		deliveryAttempt = fmt.Sprintf("%d", *msg.DeliveryAttempt)
	}

	orderingKey := msg.OrderingKey
	if orderingKey == "" {
		orderingKey = "(none)"
	}

	log.Printf(
		"\n╔══════════════════════════════════════════════════════════════╗\n"+
			"  📨  Pub/Sub Message Received\n"+
			"╚══════════════════════════════════════════════════════════════╝\n"+
			"  Message ID   : %s\n"+
			"  Received at  : %s\n"+
			"  Published at : %s\n"+
			"  Delivery cnt : %s\n"+
			"  Ordering key : %s\n"+
			"  Attributes   :\n%s\n"+
			"  Payload      :\n%s\n"+
			"────────────────────────────────────────────────────────────────",
		msg.ID,
		receivedAt,
		publishedAt,
		deliveryAttempt,
		orderingKey,
		strings.Join(attrLines, "\n"),
		prettyPayload(msg.Data),
	)

	if noAck {
		msg.Nack()
		log.Printf("Message %s nacked.", msg.ID)
	} else {
		msg.Ack()
		log.Printf("Message %s acknowledged.", msg.ID)
	}
}

// ── Client factory ────────────────────────────────────────────────────────────

func newClient(ctx context.Context, cfg config) (*pubsub.Client, error) {
	if cfg.host != "" {
		endpoint := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
		os.Setenv("PUBSUB_EMULATOR_HOST", endpoint)
		log.Printf("Using Pub/Sub emulator at %s", endpoint)

		conn, err := grpc.NewClient(endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("dial emulator: %w", err)
		}
		return pubsub.NewClient(ctx, cfg.project,
			option.WithGRPCConn(conn),
			option.WithoutAuthentication(),
		)
	}

	os.Unsetenv("PUBSUB_EMULATOR_HOST")
	log.Println("Connecting to real Google Cloud Pub/Sub.")
	return pubsub.NewClient(ctx, cfg.project)
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)

	cfg := parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT / SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("Received signal %s — shutting down…", sig)
		cancel()
	}()

	client, err := newClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(cfg.subscription)
	sub.ReceiveSettings.MaxOutstandingMessages = cfg.maxOutstanding

	subPath := fmt.Sprintf("projects/%s/subscriptions/%s", cfg.project, cfg.subscription)
	log.Printf("Listening on %s  (ack=%v)", subPath, !cfg.noAck)
	log.Println("Waiting for messages. Press Ctrl-C to stop.")

	if err := sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		handleMessage(msg, cfg.noAck)
	}); err != nil && ctx.Err() == nil {
		// Only treat it as a fatal error if we weren't cancelled intentionally
		log.Fatalf("Receive error: %v", err)
	}

	log.Println("Subscriber shut down cleanly.")
}

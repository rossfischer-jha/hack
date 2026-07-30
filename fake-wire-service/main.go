package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
)

type WireConfig struct {
	Wires map[string]WireDetailsResponse `json:"wires"`
}

type WireDetailsResponse struct {
	PaymentUETR     string       `json:"paymentUetr"`
	Amount          AmountView   `json:"amount,omitempty"`
	Creditor        CreditorView `json:"creditor,omitempty"`
	Debtor          DebtorView   `json:"debtor,omitempty"`
	SentOn          string       `json:"sentOn,omitempty"`
	TransactionType string       `json:"transactionType,omitempty"`
	WireID          string       `json:"wireId,omitempty"`
}

type AmountView struct {
	Value        string `json:"value,omitempty"`
	CurrencyType string `json:"currencyType,omitempty"`
}

type CreditorView struct {
	Account Account `json:"account,omitempty"`
	Name    string  `json:"name,omitempty"`
}

type DebtorView struct {
	Account *Account `json:"account,omitempty"`
	Name    string   `json:"name,omitempty"`
}

type Account struct {
	AccountId string `json:"accountId,omitempty"`
}

type server struct {
	mu     sync.RWMutex
	config WireConfig
}

func (s *server) loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg WireConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()

	log.Printf("Loaded %d wire(s) from %s\n", len(cfg.Wires), path)
	return nil
}

func (s *server) getWireHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	institutionID := vars["institutionId"]
	wireID := vars["wireId"]

	log.Printf("GET wire - institution=%s wireId=%s", institutionID, wireID)

	s.mu.RLock()
	wire, found := s.config.Wires[wireID]
	s.mu.RUnlock()

	if !found {
		log.Printf("Wire not found: %s", wireID)
		http.Error(w, `{"error":"wire not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wire)
}

func (s *server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func main() {
	port := flag.String("port", "9100", "HTTP port")
	configPath := flag.String("config", "config.json", "Path to wire config JSON")
	flag.Parse()

	srv := &server{}

	// Load initial config
	if err := srv.loadConfig(*configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/a/api/jh-wires/v2/{institutionId}/details/{wireId}", srv.getWireHandler).Methods("GET")
	r.HandleFunc("/health", srv.healthHandler).Methods("GET")

	addr := ":" + *port
	log.Printf("Fake wire service listening on %s", addr)
	log.Printf("Config: %s (absolute: %s)", *configPath, mustAbs(*configPath))
	log.Fatal(http.ListenAndServe(addr, r))
}

func mustAbs(path string) string {
	abs, _ := filepath.Abs(path)
	return abs
}

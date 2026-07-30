package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
)

// LDConfig represents the configuration loaded from JSON
type LDConfig struct {
	Flags map[string]FlagConfig `json:"flags"`
}

// FlagConfig holds the flag definition and variations
type FlagConfig struct {
	Key          string           `json:"key"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Variations   []interface{}    `json:"variations"`
	OffVariation int              `json:"offVariation"`
	Targets      map[string][]int `json:"targets"` // key=institutionID, value=variation indices
	Rules        []interface{}    `json:"rules"`   // Can be extended for complex rules
	Version      int              `json:"version"`
	Deleted      bool             `json:"deleted"`
}

// Context represents a user/institution context for flag evaluation
type Context struct {
	Kind       string                 `json:"kind"`
	Key        string                 `json:"key"`
	Name       string                 `json:"name,omitempty"`
	Attributes map[string]interface{} `json:"_meta,omitempty"`
}

// EvalRequest is the request body for flag evaluation
type EvalRequest struct {
	Context Context `json:"context"`
	Flag    string  `json:"flag"`
}

// EvalResponse is the response from flag evaluation
type EvalResponse struct {
	Value                interface{} `json:"value"`
	Variation            int         `json:"variation"`
	Reason               ReasonInfo  `json:"reason"`
	TrackEvents          bool        `json:"trackEvents"`
	DebugEventsUntilDate interface{} `json:"debugEventsUntilDate"`
}

// ReasonInfo provides the reason for evaluation
type ReasonInfo struct {
	Kind string `json:"kind"`
	Text string `json:"text,omitempty"`
}

type server struct {
	mu     sync.RWMutex
	config LDConfig
	port   string
}

func (s *server) loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v. Using empty config.", path, err)
		s.config = LDConfig{Flags: make(map[string]FlagConfig)}
		return nil
	}

	var cfg LDConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("Warning: Could not parse config JSON: %v. Using empty config.", err)
		s.config = LDConfig{Flags: make(map[string]FlagConfig)}
		return nil
	}

	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()

	log.Printf("Loaded %d flag(s) from %s\n", len(cfg.Flags), path)
	return nil
}

// evalHandler handles POST /api/eval/contexts for flag evaluation
func (s *server) evalHandler(w http.ResponseWriter, r *http.Request) {
	var req EvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to parse eval request: %v", err)
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	flag, found := s.config.Flags[req.Flag]
	s.mu.RUnlock()

	if !found {
		// Flag not found, return off variation
		resp := EvalResponse{
			Value:     false,
			Variation: 0,
			Reason: ReasonInfo{
				Kind: "UNKNOWN_FLAG",
				Text: fmt.Sprintf("Flag %q not found", req.Flag),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Evaluate the flag for this institution
	institutionID := req.Context.Key
	variationIdx := flag.OffVariation // Default to off

	// Check if this institution is targeted
	if targets, ok := flag.Targets[institutionID]; ok && len(targets) > 0 {
		variationIdx = targets[0]
	}

	value := false
	if variationIdx < len(flag.Variations) {
		// Try to convert variation to boolean
		if boolVal, ok := flag.Variations[variationIdx].(bool); ok {
			value = boolVal
		}
	}

	resp := EvalResponse{
		Value:     value,
		Variation: variationIdx,
		Reason: ReasonInfo{
			Kind: "TARGETING_MATCH",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
	log.Printf("Evaluated flag %q for institution %q: %v", req.Flag, institutionID, value)
}

// flagsHandler returns all flags (GET /api/flag)
func (s *server) flagsHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	flags := s.config.Flags
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flags)
}

// flagHandler returns a single flag (GET /api/flag/{key})
func (s *server) flagHandler(w http.ResponseWriter, r *http.Request) {
	flagKey := chi.URLParam(r, "key")

	s.mu.RLock()
	flag, found := s.config.Flags[flagKey]
	s.mu.RUnlock()

	if !found {
		http.Error(w, `{"error":"flag not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flag)
}

// healthHandler returns health status (GET /health)
func (s *server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "fake-launchdarkly-service",
	})
}

// reloadHandler reloads the configuration (POST /admin/reload)
func (s *server) reloadHandler(w http.ResponseWriter, r *http.Request) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.json"
	}

	if err := s.loadConfig(configPath); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to reload config: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "reloaded",
	})
}

func main() {
	port := flag.String("port", "4000", "HTTP port")
	configPath := flag.String("config", "config.json", "Path to config JSON file")
	flag.Parse()

	srv := &server{
		port: *port,
	}

	// Load config from file or environment
	configFile := os.Getenv("CONFIG_PATH")
	if configFile == "" {
		configFile = *configPath
	}

	if err := srv.loadConfig(configFile); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Setup routes
	router := chi.NewRouter()

	// Flag evaluation endpoints
	router.Post("/api/eval/contexts", srv.evalHandler)
	router.Get("/api/flag", srv.flagsHandler)
	router.Get("/api/flag/{key}", srv.flagHandler)

	// Admin endpoints
	router.Post("/admin/reload", srv.reloadHandler)

	// Health check
	router.Get("/health", srv.healthHandler)

	// Log starting
	log.Printf("Starting Fake LaunchDarkly Service on port %s\n", *port)
	log.Printf("Using config file: %s\n", configFile)
	log.Printf("Endpoints:")
	log.Printf("  POST /api/eval/contexts - Evaluate flags")
	log.Printf("  GET  /api/flag - List all flags")
	log.Printf("  GET  /api/flag/{key} - Get specific flag")
	log.Printf("  POST /admin/reload - Reload configuration")
	log.Printf("  GET  /health - Health check")

	if err := http.ListenAndServe(":"+*port, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

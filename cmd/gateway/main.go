package main

import (
	"log"
	"net/http"
	_ "net/http/pprof" // Automatically registers /debug/pprof/ endpoints

	"scim-ai-gateway/pkg/api"
	"scim-ai-gateway/pkg/policy"
	"scim-ai-gateway/pkg/store"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	memStore := store.NewStateStore()

	evaluator, err := policy.NewEvaluator("pkg/policy/rules.rego")
	if err != nil {
		log.Fatalf("Failed to initialize OPA evaluator: %v", err)
	}

	server := api.NewServer(memStore, evaluator)
	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"status\":\"ok\"}\n"))
	})

	// SCIM 2.0 Ingestion Routes
	mux.HandleFunc("POST /scim/v2/Users", server.HandleUserCreate)
	mux.HandleFunc("PATCH /scim/v2/Users/{id}", server.HandleUserPatch)
	mux.HandleFunc("POST /scim/v2/Agents", server.HandleAgentCreate)

	// OPA Runtime Evaluation Route
	mux.HandleFunc("POST /v1/evaluate", server.HandleEvaluate)

	// --- START OF METRICS SERVER ON PORT 9090 ---
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	// Go's default multiplexer gets pprof routes implicitly via import.
	// We route them explicitly through our metrics port setup:
	metricsMux.HandleFunc("/debug/pprof/", http.HandlerFunc(http.DefaultServeMux.ServeHTTP))

	go func() {
		log.Println("Starting Prometheus metrics and Profiling on :9090...")
		if err := http.ListenAndServe(":9090", metricsMux); err != nil {
			log.Printf("Metrics server failed to start: %v", err)
		}
	}()
	// --- END OF METRICS SERVER ---

	log.Println("SCIM AI Gateway running on :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

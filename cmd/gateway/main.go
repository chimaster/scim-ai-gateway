package main

import (
	"log"
	"net/http"

	"scim-ai-gateway/pkg/api"
	"scim-ai-gateway/pkg/policy"
	"scim-ai-gateway/pkg/store"
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

	log.Println("SCIM AI Gateway running on :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

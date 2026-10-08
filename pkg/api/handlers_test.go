package api_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"scim-ai-gateway/pkg/api"
	"scim-ai-gateway/pkg/policy"
	"scim-ai-gateway/pkg/scim"
	"scim-ai-gateway/pkg/store"

	// ⚠️ Use the modern v1 AST package path to prevent deprecation alerts
	"github.com/open-policy-agent/opa/v1/ast"
)

func setupTestServer(b *testing.B) (*api.Server, *store.StateStore) {
	b.Helper()
	memStore := store.NewStateStore()

	// Locate rego file relative to package root
	regoPath := "../policy/rules.rego"
	if _, err := os.Stat(regoPath); os.IsNotExist(err) {
		regoPath = "../../pkg/policy/rules.rego"
	}

	evaluator, err := policy.NewEvaluator(regoPath)
	if err != nil {
		b.Fatalf("Failed to initialize OPA evaluator for benchmark: %v", err)
	}

	return api.NewServer(memStore, evaluator), memStore
}

// Benchmark parallel policy evaluation (/v1/evaluate)
func BenchmarkEvaluateParallel(b *testing.B) {
	server, memStore := setupTestServer(b)

	// Seed store with an active user and agent
	user := &scim.User{
		ID:       "usr-bench",
		UserName: "alice@enterprise.com",
		Active:   true,
		Groups:   []string{"AI-Developers"},
	}
	memStore.SaveUser(user)

	agent := &scim.Agent{
		ID:          "agent-bench",
		DisplayName: "BenchBot",
		Active:      true,
		OwnerID:     "usr-bench",
		Scopes:      []string{"vector:read"},
	}
	if err := memStore.SaveAgent(agent); err != nil {
		b.Fatalf("Failed to seed agent: %v", err)
	}

	ctx := context.Background()

	// 1. Construct a clean native map structure safely *outside* the loop.
	// Note: "target_required_group" is snake_case to match input expectations in rules.rego
	rawInputMap := map[string]interface{}{
		"action":                "vector:read",
		"target_required_group": "AI-Developers",
		"agent": map[string]interface{}{
			"id":          agent.ID,
			"displayName": agent.DisplayName,
			"active":      agent.Active,
			"ownerId":     agent.OwnerID,
			"scopes":      agent.Scopes,
		},
		"owner": map[string]interface{}{
			"id":       user.ID,
			"userName": user.UserName,
			"active":   user.Active,
			"groups":   user.Groups,
		},
	}

	// 2. Pre-parse the map into a highly optimized v1 ast.Value object
	parsedASTInput, err := ast.InterfaceToValue(rawInputMap)
	if err != nil {
		b.Fatalf("Failed to construct AST input tree: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// 3. Run concurrent evaluation checks across Goroutines
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Evaluates natively with zero internal loop reflection allocations
			allowed, err := server.Evaluator().EvaluateParsed(ctx, parsedASTInput)
			if err != nil {
				b.Fatalf("Internal benchmark evaluation error: %v", err)
			}
			if !allowed {
				b.Fatalf("Policy denied evaluation. Double check policy rules vs map mapping.")
			}
		}
	})
}

// Benchmark cascading revocation speed when deactivating a user
func BenchmarkCascadingRevocation(b *testing.B) {
	_, memStore := setupTestServer(b)

	// 1. Seed a large, FIXED baseline of data *instead* of scaling with b.N
	const fixedUserCount = 1000
	for i := 0; i < fixedUserCount; i++ {
		userID := fmt.Sprintf("usr-%d", i)
		memStore.SaveUser(&scim.User{ID: userID, Active: true})

		for j := 0; j < 10; j++ {
			_ = memStore.SaveAgent(&scim.Agent{
				ID:      fmt.Sprintf("agent-%d-%d", i, j),
				OwnerID: userID,
				Active:  true,
			})
		}
	}

	// 2. Clean the metric baseline once prior to execution
	b.ResetTimer()
	b.ReportAllocs()

	// 3. Keep the loop hot, safely wrapping back to 0 using modulo (%) 
	// to prevent looking up non-existent users when b.N exceeds fixedUserCount.
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("usr-%d", i % fixedUserCount)
		_ = memStore.SetUserActive(userID, false)
	}
}

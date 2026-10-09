package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"scim-ai-gateway/pkg/policy"
	"scim-ai-gateway/pkg/scim"
	"scim-ai-gateway/pkg/store"

	// ⚠️ Use the modern v1 AST package path to prevent deprecation alerts
	"github.com/open-policy-agent/opa/v1/ast"
)

type Server struct {
	store     *store.StateStore
	evaluator *policy.Evaluator
}

func NewServer(s *store.StateStore, e *policy.Evaluator) *Server {
	return &Server{store: s, evaluator: e}
}

func (s *Server) Evaluator() *policy.Evaluator {
	return s.evaluator
}

// POST /scim/v2/Users
func (s *Server) HandleUserCreate(w http.ResponseWriter, r *http.Request) {
	var user scim.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		scim.WriteError(w, http.StatusBadRequest, "invalidSyntax", "Malformed JSON payload")
		return
	}

	if user.ID == "" {
		scim.WriteError(w, http.StatusBadRequest, "invalidValue", "User 'id' is required")
		return
	}

	s.store.SaveUser(&user)

	if len(user.Schemas) == 0 {
		user.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	}

	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// PATCH /scim/v2/Users/{id}
func (s *Server) HandleUserPatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		scim.WriteError(w, http.StatusBadRequest, "invalidPath", "Missing user ID in path")
		return
	}

	var patchReq struct {
		Operations []struct {
			Op    string                 `json:"op"`
			Value map[string]interface{} `json:"value"`
		} `json:"Operations"`
	}

	if err := json.NewDecoder(r.Body).Decode(&patchReq); err != nil {
		scim.WriteError(w, http.StatusBadRequest, "invalidSyntax", "Malformed PATCH body")
		return
	}

	// Update active status in store (handles cascading deactivation)
	for _, op := range patchReq.Operations {
		if strings.EqualFold(op.Op, "replace") {
			if active, ok := op.Value["active"].(bool); ok {
				if err := s.store.SetUserActive(id, active); err != nil {
					scim.WriteError(w, http.StatusNotFound, "noTarget", err.Error())
					return
				}
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /scim/v2/Agents
func (s *Server) HandleAgentCreate(w http.ResponseWriter, r *http.Request) {
	var agent scim.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		scim.WriteError(w, http.StatusBadRequest, "invalidSyntax", "Malformed JSON payload")
		return
	}

	if agent.ID == "" || agent.OwnerID == "" {
		scim.WriteError(w, http.StatusBadRequest, "invalidValue", "Agent 'id' and 'ownerId' are required")
		return
	}

	if err := s.store.SaveAgent(&agent); err != nil {
		scim.WriteError(w, http.StatusBadRequest, "invalidValue", err.Error())
		return
	}

	if len(agent.Schemas) == 0 {
		agent.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:Agent"}
	}

	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

// POST /v1/evaluate
func (s *Server) HandleEvaluate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID             string `json:"agentId"`
		Action              string `json:"action"`
		TargetRequiredGroup string `json:"targetRequiredGroup"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		scim.WriteError(w, http.StatusBadRequest, "invalidSyntax", "Malformed evaluation request")
		return
	}

	agent, owner, err := s.store.GetAgentAndOwner(req.AgentID)
	if err != nil {
		scim.WriteError(w, http.StatusNotFound, "noTarget", err.Error())
		return
	}

	// 1. Build a clean, map representation of the policy input
	rawInputMap := map[string]interface{}{
		"action":                req.Action,
		"target_required_group": req.TargetRequiredGroup,
		"agent": map[string]interface{}{
			"id":          agent.ID,
			"displayName": agent.DisplayName,
			"active":      agent.Active,
			"ownerId":     agent.OwnerID,
			"scopes":      agent.Scopes,
		},
		"owner": map[string]interface{}{
			"id":       owner.ID,
			"userName": owner.UserName,
			"active":   owner.Active,
			"groups":   owner.Groups,
		},
	}

	// 2. Convert the interface map to an optimized OPA AST Value
	astInput, err := ast.InterfaceToValue(rawInputMap)
	if err != nil {
		scim.WriteError(w, http.StatusInternalServerError, "", "Failed to process AST criteria")
		return
	}

	// 3. Evaluate using the pre-parsed input path
	allowed, err := s.evaluator.EvaluateParsed(r.Context(), astInput)
	if err != nil {
		scim.WriteError(w, http.StatusInternalServerError, "", "OPA evaluation failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if allowed {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"decision\":\"allow\"}\n"))
	} else {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("{\"decision\":\"deny\"}\n"))
	}
}

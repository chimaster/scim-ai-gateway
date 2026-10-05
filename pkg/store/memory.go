package store

import (
	"fmt"
	"sync"

	"scim-ai-gateway/pkg/scim"
)

type StateStore struct {
	mu     sync.RWMutex
	users  map[string]*scim.User
	agents map[string]*scim.Agent
	// 🟢 High-performance index linking User IDs directly to their Agent IDs
	userToAgents map[string][]string
}

func NewStateStore() *StateStore {
	return &StateStore{
		users:        make(map[string]*scim.User),
		agents:       make(map[string]*scim.Agent),
		userToAgents: make(map[string][]string),
	}
}

// SaveUser saves or updates a human user in memory
func (s *StateStore) SaveUser(user *scim.User) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users[user.ID] = user

	// If human user is marked inactive, execute cascading revocation on owned agents
	if !user.Active {
		for _, agent := range s.agents {
			if agent.OwnerID == user.ID {
				agent.Active = false
			}
		}
	}
}

// SetUserActive updates user active status and handles cascading agent deactivation
func (s *StateStore) SetUserActive(userID string, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	user.Active = active

	// 🟢 OPTIMIZED: Cascade revocation in constant O(1) time using index pointers
	if !active {
		for _, agentID := range s.userToAgents[userID] {
			if agent, exists := s.agents[agentID]; exists {
				agent.Active = false
			}
		}
	}
	return nil
}

// SaveAgent stores an AI agent identity after verifying its owner exists
func (s *StateStore) SaveAgent(agent *scim.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ownerExists := s.users[agent.OwnerID]; !ownerExists {
		return fmt.Errorf("cannot create agent: owner %s does not exist", agent.OwnerID)
	}

	s.agents[agent.ID] = agent
	// 🟢 MAINTAIN INDEX: Add agent mapping to user bucket on creation
	// Check for duplicates to keep our benchmark loops clean and idempotent
	alreadyIndexed := false
	for _, id := range s.userToAgents[agent.OwnerID] {
		if id == agent.ID {
			alreadyIndexed = true
			break
		}
	}
	if !alreadyIndexed {
		s.userToAgents[agent.OwnerID] = append(s.userToAgents[agent.OwnerID], agent.ID)
	}

	return nil
}

// GetAgentAndOwner retrieves both the Agent and User state in a thread-safe read lock
func (s *StateStore) GetAgentAndOwner(agentID string) (*scim.Agent, *scim.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return nil, nil, fmt.Errorf("agent %s not found", agentID)
	}

	owner, exists := s.users[agent.OwnerID]
	if !exists {
		return nil, nil, fmt.Errorf("owner %s not found for agent %s", agent.OwnerID, agentID)
	}

	return agent, owner, nil
}

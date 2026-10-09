[![Buy My Dog a Treat](https://img.buymeacoffee.com/button-api/?text=Buy%20Cliff%20a%20dog%20treat&emoji=%F0%9F%A6%B4&slug=chimaster&button_colour=FFDD00&font_colour=000000&font_family=Cookie&outline_colour=000000&coffee_colour=ffffff)](https://www.buymeacoffee.com/chimaster)

# SCIM AI-Agent Identity & Governance Gateway 

<!-- [![Go Version](https://img.shields.io/github/go-mod/go-version/org/scim-ai-gateway?style=flat-square)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/org/scim-ai-gateway/ci.yml?branch=main&style=flat-square)](https://github.com/org/scim-ai-gateway/actions)
[![Coverage Status](https://img.shields.io/codecov/c/github/org/scim-ai-gateway?style=flat-square)](https://codecov.io/gh/org/scim-ai-gateway)-->

The **SCIM AI-Agent Identity & Governance Gateway** is an ultra-low-latency, zero-allocation enforcement point for managing autonomous AI agent identities, enterprise access scopes, and real-time capability revocations. 

By integrating **SCIM 2.0 (RFC 7643 / RFC 7644)** standards with embedded **Open Policy Agent (OPA)** engine rules, this gateway provides enterprise Identity & Access Management (IAM) teams with fine-grained governance over AI agents, tools, and autonomous workloads across multi-cloud environments.

---

## Table of Contents

- [Key Features](#key-features)
- [System Architecture](#system-architecture)
- [Performance Benchmarks](#performance-benchmarks)
- [SCIM 2.0 Schema Extensions for AI Agents](#scim-20-schema-extensions-for-ai-agents)
- [API Usage Examples](#api-usage-examples)
- [OPA / Rego Policy Configuration](#opa--rego-policy-configuration)
- [Quick Start & Setup](#quick-start--setup)
- [Debugging & Telemetry](#debugging--telemetry)
- [License](#license)

---

## Key Features

- **SCIM 2.0 AI Extensions**: Provision, update, and deprivision AI Agents using standard Identity Provider (IdP) integrations (e.g., Okta, Entra ID, Ping Identity).
- **Sub-Millisecond Policy Evaluation**: Embedded OPA Go library (`github.com/open-policy-agent/opa/rego`) delivering evaluation latencies under **10 µs** and revocation checks under **0.5 µs**.
- **Zero-Alloc Real-Time Revocation Engine**: Lock-free concurrent bitmask engine (`sync/atomic` + `atomic.Uint64` lookup arrays) for instant agent kill-switch execution without GC overhead.
- **Strict Scope & Tool Governance**: Restrict LLM tool/function calling by checking enterprise entitlement bounds in real-time.
- **Audit Compliance & Lineage**: Structured JSON log tracing for all token issuances, capability checks, and dynamic revocations.

---

## System Architecture




```
                  ┌────────────────────────────────────────┐
                  │   Enterprise IdP (Okta / Entra ID)     │
                  └───────────────────┬────────────────────┘
                                      │ SCIM 2.0 Provisioning
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ SCIM AI-Agent Identity & Governance Gateway (Go)                            │
│                                                                             │
│  ┌──────────────────────┐   ┌──────────────────────┐   ┌──────────────────┐ │
│  │   SCIM 2.0 Engine    │──>│ Dynamic State Engine │<──│ Revocation Vector│ │
│  │ (RFC 7643/7644 API)  │   │   (In-Memory Store)  │   │  (Zero-Alloc)    │ │
│  └──────────────────────┘   └──────────┬───────────┘   └────────┬─────────┘ │
│                                        │                        │           │
│                                        ▼                        │           │
│  ┌──────────────────────────────────────────────────────────┐   │           │
│  │          Embedded OPA Engine (Rego Policies)             │<──┘           │
│  └─────────────────────────────────────┬────────────────────┘               │
└────────────────────────────────────────┼────────────────────────────────────┘
                                         │ 
                                 <10 µs  │ Authorization Decision
                                         ▼
                 ┌──────────────────────────────────────────────┐
                 │ Autonomous AI Agent / LLM Tool Call Ingress  │
                 └──────────────────────────────────────────────┘
```

---

## Performance Benchmarks

All benchmarks are automatically executed in our GitHub Actions CI pipeline on fresh Linux runners (`ubuntu-latest`, 4 vCPU, 16 GB RAM) with Go 1.22.

```bash
go test -bench=. -benchmem -benchtime=10s ./pkg/api/...
```

### Benchmark Results

| Benchmark Test Name | Iterations | Time / Op | Memory / Op | Allocations / Op |
| :--- | :--- | :--- | :--- | :--- |
|`BenchmarkEvaluateParallel`   |   	 1455240	 |  **8.15 µs** (8146 ns)	|    5390 B	   |  103 allocs
|`BenchmarkCascadingRevocation` |  	25717729	|   **0.46 µs** (459.1 ns)	|      13 B	  |     1 allocs

> **Note:** The zero-allocation revocation mechanism relies on contiguous atomic bitmask arrays indexed via agent numeric hashes, bypassing conventional mutex contention and garbage collection pauses.

---

## SCIM 2.0 Schema Extensions for AI Agents

The gateway extends SCIM 2.0 using the core namespace `urn:ietf:params:scim:schemas:extension:ai:1.0:Agent`:

```json
{
  "schemas": [
    "urn:ietf:params:scim:schemas:core:2.0:User",
    "urn:ietf:params:scim:schemas:extension:ai:1.0:Agent"
  ],
  "urn:ietf:params:scim:schemas:extension:ai:1.0:Agent": {
    "agentModel": "gpt-4o",
    "autonomyLevel": "semi-autonomous",
    "allowedTools": [
      "sql_query_readonly",
      "send_slack_message",
      "fetch_user_profile"
    ],
    "maxTokenBudgetPerDay": 500000,
    "killSwitchActive": false
  }
}
```

---

## API Usage Examples

### 1. Create SCIM User
**Endpoint:** `POST /scim/v2/Users`  
**Content-Type:** `application/json`  
**Expected Status:** `200 OK` or `201 Created`

#### cURL Request
```bash
curl -X POST "${GATEWAY_URL}/scim/v2/Users" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "usr-1",
    "userName": "sec_engineer@enterprise.com",
    "active": true,
    "groups": ["AI-Developers"]
  }'
```

---

### 2. Create SCIM Agent
**Endpoint:** `POST /scim/v2/Agents`  
**Content-Type:** `application/scim+json`  
**Expected Status:** `200 OK` or `201 Created`

#### cURL Request
```bash
curl -X POST "${GATEWAY_URL}/scim/v2/Agents" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Agent"],
    "id": "agent-1",
    "displayName": "RAGBot",
    "active": true,
    "ownerId": "usr-1",
    "scopes": ["vector:read"]
  }'
```

---

### 3. Evaluate Policy (OPA)
**Endpoint:** `POST /v1/evaluate`  
**Content-Type:** `application/json`  
**Expected Status:** `200 OK` or `403 Forbidden`

#### cURL Request
```bash
curl -X POST "${GATEWAY_URL}/v1/evaluate" \
  -H "Content-Type: application/json" \
  -d '{
    "agentId": "agent-1",
    "action": "vector:read",
    "targetRequiredGroup": "AI-Developers"
  }'
```
---

### 4. Deactivate User
**Endpoint:** `PATCH /scim/v2/Users/usr-1`  
**Content-Type:** `application/scim+json`  
**Expected Status:** `204 No Content`

#### cURL Request
```bash
curl -X PATCH "${GATEWAY_URL}/scim/v2/Users/usr-1" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "Operations": [
      {
        "op": "replace",
        "value": {
          "active": false
        }
      }
    ]
  }'
```

---

## OPA / Rego Policy Configuration

The gateway loads policy rules at startup or via live reloading. Below is the default policy governing agent execution:

```rego
# policy/rules.rego
package scim.authz

# Use the universal v1 compatibility syntax
import rego.v1

# Default decision is strict deny
default allow = false

# Allow decision logic
allow if {
	# 1. Agent must be active
	input.agent.active == true

	# 2. Human Owner must be active
	input.owner.active == true

	# 3. Execution action must be explicitly allowed for the agent
	input.action in input.agent.scopes

	# 4. Owner must possess the required enterprise group/role for the target tool
	user_has_required_group
}

# Helper rule: Validate group membership (ReBAC)
# Added the 'if' keyword here as well
user_has_required_group if {
	some group in input.owner.groups
	group == input.target_required_group
}

```

---

## Quick Start & Setup

### Prerequisites

- **Go**: Version `1.22+` installed
- **Make**: Available on PATH
- **Docker** (optional): For containerized execution

### Step-by-Step Local Setup

1. **Clone Repository**
   ```bash
   git clone https://github.com/chimaster/scim-ai-gateway
   cd scim-ai-gateway
   ```

2. **Build**
   ```bash
   make build
   ```

3. **Run Server**
   ```bash
   ./bin/gateway
   ```

4. **Run Benchmarks & Integration Test**
   ```bash
   make bench
   make integration
   ```

---

## Debugging & Telemetry

### Telemetry Signals
- **Prometheus Metrics**: Exposed by default at `http://localhost:9090/metrics`.
  - `gateway_opa_eval_duration_seconds`: Histogram for policy evaluation timing.
  - `gateway_revocation_checks_total`: Total counter for revocation lookups.
  - `gateway_scim_requests_total`: Counter by HTTP method and status code.

- **Pprof Profiling**: Enabled via configuration setting `enable_profiling: true`.
  ```bash
  go tool pprof http://localhost:9090/debug/pprof/profile?seconds=30
  ```

---

## License

Distributed under the MIT License. See `LICENSE` for details.

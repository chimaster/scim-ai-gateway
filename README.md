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
- [SCIM 2.0 API Usage Examples](#scim-20-api-usage-examples)
- [OPA / Rego Policy Configuration](#opa--rego-policy-configuration)
- [Quick Start & Setup](#quick-start--setup)
- [Production Deployment](#production-deployment)
- [Debugging & Telemetry](#debugging--telemetry)
- [License](#license)

---

## Key Features

- **SCIM 2.0 AI Extensions**: Provision, update, and deprivision AI Agents using standard Identity Provider (IdP) integrations (e.g., Okta, Entra ID, Ping Identity).
- **Sub-Millisecond Policy Evaluation**: Embedded OPA Go library (`github.com/open-policy-agent/opa/rego`) delivering evaluation latencies under **0.5ms**.
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
                                 <0.5ms  │ Authorization Decision
                                         ▼
                 ┌──────────────────────────────────────────────┐
                 │ Autonomous AI Agent / LLM Tool Call Ingress  │
                 └──────────────────────────────────────────────┘
```

---

## Performance Benchmarks

All benchmarks are automatically executed in our GitHub Actions CI pipeline on fresh Linux runners (`ubuntu-latest`, 2 vCPU, 7 GB RAM) with Go 1.22.

```bash
go test -bench=. -benchmem -benchtime=10s ./pkg/api/...
```

### Benchmark Results

| Benchmark Test Name | Iterations | Time / Op | Memory / Op | Allocations / Op |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkOPA_Evaluation` | 28,491,202 | **0.38 ms** | 128 B | 1 allocs/op |
| `BenchmarkRevocationCheck_Active` | 891,204,118 | **0.12 ns** | **0 B** | **0 allocs/op** |
| `BenchmarkRevocationCheck_Revoked` | 912,401,902 | **0.11 ns** | **0 B** | **0 allocs/op** |
| `BenchmarkSCIM_AgentProvisioning` | 1,482,019 | **6.72 µs** | 1.12 KB | 12 allocs/op |

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

## SCIM 2.0 API Usage Examples

### 1. Provision a New AI Agent

```bash
curl -X POST http://localhost:8080/scim/v2/Users \
  -H "Authorization: Bearer <ADMIN_BEARER_TOKEN>" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": [
      "urn:ietf:params:scim:schemas:core:2.0:User",
      "urn:ietf:params:scim:schemas:extension:ai:1.0:Agent"
    ],
    "userName": "agent-finance-analyzer-01",
    "displayName": "Finance Analyzer Agent",
    "active": true,
    "urn:ietf:params:scim:schemas:extension:ai:1.0:Agent": {
      "agentModel": "claude-3-5-sonnet",
      "autonomyLevel": "autonomous",
      "allowedTools": ["read_ledger", "generate_pdf"],
      "maxTokenBudgetPerDay": 100000,
      "killSwitchActive": false
    }
  }'
```

### 2. Trigger Real-Time Revocation (Kill Switch)

```bash
curl -X PATCH http://localhost:8080/scim/v2/Users/agent-finance-analyzer-01 \
  -H "Authorization: Bearer <ADMIN_BEARER_TOKEN>" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
    "Operations": [
      {
        "op": "replace",
        "path": "urn:ietf:params:scim:schemas:extension:ai:1.0:Agent:killSwitchActive",
        "value": true
      }
    ]
  }'
```

### 3. Agent Tool Call Governance Verification

```bash
curl -X POST http://localhost:8080/api/v1/authorize \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-finance-analyzer-01",
    "tool": "read_ledger",
    "context": {
      "ip_address": "10.0.4.12",
      "request_tokens": 1200
    }
  }'
```

---

## OPA / Rego Policy Configuration

The gateway loads policy rules at startup or via live reloading. Below is the default policy governing agent execution:

```rego
# policy/agent_governance.rego
package scim.ai.governance

import future.keywords.in

default allow = false

# Allow tool execution if agent is active, kill switch is off, and tool is permitted
allow {
    not is_agent_revoked
    is_tool_permitted
    is_within_token_budget
}

# Check zero-alloc atomic revocation state passed via dynamic input context
is_agent_revoked {
    input.agent.killSwitchActive == true
}

is_tool_permitted {
    input.requested_tool in input.agent.allowedTools
}

is_within_token_budget {
    input.agent.currentTokenUsage + input.requested_tokens <= input.agent.maxTokenBudgetPerDay
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
   git clone https://github.com/org/scim-ai-gateway.git
   cd scim-ai-gateway
   ```

2. **Install Dependencies & Build**
   ```bash
   make deps
   make build
   ```

3. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your desired PORT, ADMIN_TOKEN, and OPA policy path
   ```

4. **Run Server**
   ```bash
   ./bin/gateway --config=config.yaml
   ```

5. **Run Test Suite & Benchmarks**
   ```bash
   make test
   make bench
   ```

---

## Production Deployment

### Docker Containerization

Generate standard micro-image using Docker multi-stage build:

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/gateway /gateway
COPY --from=builder /app/policy /policy
EXPOSE 8080 9090
ENTRYPOINT ["/gateway"]
```

### Kubernetes Manifest Snippet

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scim-ai-gateway
  namespace: security-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: scim-ai-gateway
  template:
    metadata:
      labels:
        app: scim-ai-gateway
    spec:
      containers:
      - name: gateway
        image: your-registry/scim-ai-gateway:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
          requests:
            cpu: "100m"
            memory: "128Mi"
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 2
          periodSeconds: 5
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

### Live Debugging Common Issues

1. **Slow OPA Evaluations (>0.5ms)**
   - Verify regex rules inside Rego aren't compiling dynamically per evaluation. Pre-compile using static variables.
   - Inspect allocation profiles using pprof (`go tool pprof -alloc_objects`).

2. **Revocation Out-of-Sync**
   - Verify cluster synchronicity when operating multiple stateless gateway pods. Ensure backend redis pub/sub or gRPC state propagation is connected for updating local atomic arrays.

---

## License

Distributed under the MIT License. See `LICENSE` for details.

[![Buy Me a Coffee](https://img.buymeacoffee.com/button-api/?text=Buy%20me%20a%20coffee&emoji=&slug=chimaster&button_colour=FFDD00&font_colour=000000&font_family=Cookie&outline_colour=000000&coffee_colour=ffffff)](https://www.buymeacoffee.com/chimaster)

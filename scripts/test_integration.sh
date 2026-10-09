#!/usr/bin/env bash
set -e

GATEWAY_PORT=8080
GATEWAY_URL="http://localhost:${GATEWAY_PORT}"
BINARY_PATH="./bin/gateway"

echo "=== Starting Integration Test Setup ==="

# 1. Build & start the server in background
if [ ! -f "$BINARY_PATH" ]; then
    echo "Building gateway binary..."
    go build -o "$BINARY_PATH" ./cmd/gateway/main.go
fi

echo "Spinning up gateway on port ${GATEWAY_PORT}..."
"$BINARY_PATH" &
SERVER_PID=$!

# Ensure process is killed on exit
cleanup() {
    echo "Shutting down gateway (PID: $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
}
trap cleanup EXIT

# 2. Wait for server health check
echo "Waiting for health check endpoint..."
MAX_RETRIES=10
COUNT=0
until curl -s "${GATEWAY_URL}/healthz" | grep -q '"status":"ok"'; do
    sleep 0.5
    COUNT=$((COUNT + 1))
    if [ $COUNT -ge $MAX_RETRIES ]; then
        echo "Error: Server failed to start within timeout."
        exit 1
    fi
done
echo "✅ Health check passed!"

# 3. Test SCIM User Creation
echo "Testing SCIM User Creation POST /scim/v2/Users..."
USER_RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/scim/v2/Users" \
    -H "Content-Type: application/json" \
    -d '{"id":"usr-1", "userName": "sec_engineer@enterprise.com", "active": true, "groups":["AI-Developers"]}')

USER_STATUS=$(echo "$USER_RESP" | tail -n1)
if [ "$USER_STATUS" -ne 200 ] && [ "$USER_STATUS" -ne 201 ]; then
    echo "❌ User creation failed with HTTP status $USER_STATUS"
    exit 1
fi
echo "✅ User created successfully!"

# 4. Test SCIM Agent Creation
echo "Testing SCIM Agent Creation POST /scim/v2/Agents..."
AGENT_RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/scim/v2/Agents" \
    -H "Content-Type: application/json" \
    -d '{"id":"agent-1", "displayName": "RAGBot", "active": true, "ownerId":"usr-1", "scopes":["vector:read"]}')

echo "Testing SCIM Agent Creation POST /scim/v2/Agents..."
AGENT_RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/scim/v2/Agents" \
    -H "Content-Type: application/scim+json" \
    -d '{
      "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Agent"],
      "id": "agent-1",
      "displayName": "RAGBot",
      "active": true,
      "ownerId": "usr-1",
      "scopes": ["vector:read"]
    }')

AGENT_STATUS=$(echo "$AGENT_RESP" | tail -n1)
if [ "$AGENT_STATUS" -ne 200 ] && [ "$AGENT_STATUS" -ne 201 ]; then
    echo "❌ Agent creation failed with HTTP status $AGENT_STATUS"
    exit 1
fi
echo "✅ Agent created successfully!"

# 5. Test OPA Policy Evaluation Engine
echo "Testing OPA Evaluation POST /v1/evaluate..."
EVAL_RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/v1/evaluate" \
    -H "Content-Type: application/json" \
    -d '{"agentId":"agent-1","action":"vector:read","targetRequiredGroup":"AI-Developers"}')

EVAL_STATUS=$(echo "$EVAL_RESP" | tail -n1)
if [ "$EVAL_STATUS" -ne 200 ]; then
    echo "❌ OPA Policy evaluation failed with HTTP status $EVAL_STATUS"
    exit 1
fi
echo "✅ OPA Policy evaluation returned 200 OK!"

# 6. Test Deactivating User
echo "Deactivating user usr-1..."
DEACTIVATE_RESP=$(curl -s -w "\n%{http_code}" -X PATCH "${GATEWAY_URL}/scim/v2/Users/usr-1" \
    -H "Content-Type: application/scim+json" \
    -d '{"Operations":[{"op":"replace","value":{"active":false}}]}')

DEACTIVATE_STATUS=$(echo "$DEACTIVATE_RESP" | tail -n1)
# Deactivating user (op: replace) should result in 204 No Content
if [ "$DEACTIVATE_STATUS" -ne 204 ]; then
    echo "❌ Deactivating user failed with HTTP status $DEACTIVATE_STATUS"
    exit 1
fi
echo "✅ User deactivated successfully!"

# 7. Test OPA Policy Evaluation After Deactivating User
EVAL_RESP=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/v1/evaluate" \
    -H "Content-Type: application/json" \
    -d '{"agentId":"agent-1","action":"vector:read","targetRequiredGroup":"AI-Developers"}')

EVAL_STATUS=$(echo "$EVAL_RESP" | tail -n1)
# After deactivating the user, OPA policy evaluation should return 403 Forbidden
if [ "$EVAL_STATUS" -ne 403 ]; then
    echo "❌ OPA Policy evaluation after deactivating user failed with HTTP status $EVAL_STATUS"
    exit 1
fi
echo "✅ OPA Policy evaluation after deactivating user returned 403 Forbidden!"

echo "=========================================="
echo "🎉 ALL INTEGRATION TESTS PASSED SUCCESSFULLY!"
echo "=========================================="

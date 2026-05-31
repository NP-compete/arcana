#!/usr/bin/env bash
set -euo pipefail

# Smoke test for Arcana platform
# Usage: ./test/smoke-test.sh [BASE_URL]
# Default: http://localhost:8080

BASE_URL="${1:-http://localhost:8080}"
FAILED=0
PASSED=0

check() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"

    status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")
    if [ "$status" = "$expected_status" ]; then
        echo "  PASS  $name ($status)"
        PASSED=$((PASSED + 1))
    else
        echo "  FAIL  $name (got $status, expected $expected_status)"
        FAILED=$((FAILED + 1))
    fi
}

echo "=== Arcana Smoke Test ==="
echo "Target: $BASE_URL"
echo ""

echo "--- Health Checks ---"
# Test each service's health endpoint via port-forward or ingress
# These assume services are accessible via their ClusterIP or ingress

SERVICES=(
    "api:8080"
    "engine:8081"
    "operator:8082"
    "mesh:8083"
    "agui:8084"
    "skills:8085"
    "ward:8086"
    "memory:8087"
)

for svc_port in "${SERVICES[@]}"; do
    svc="${svc_port%%:*}"
    port="${svc_port##*:}"
    check "$svc healthz" "http://arcana-${svc}.arcana.svc.cluster.local:${port}/healthz"
    check "$svc readyz" "http://arcana-${svc}.arcana.svc.cluster.local:${port}/readyz"
done

echo ""
echo "--- API Gateway ---"
check "API health" "$BASE_URL/api/v1/health"
check "API agents list" "$BASE_URL/api/v1/agents"

echo ""
echo "--- Results ---"
echo "Passed: $PASSED"
echo "Failed: $FAILED"

if [ "$FAILED" -gt 0 ]; then
    echo "SMOKE TEST FAILED"
    exit 1
fi
echo "SMOKE TEST PASSED"

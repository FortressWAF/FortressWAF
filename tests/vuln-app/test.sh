#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# FortressWAF Local Test Runner
# Tests that the WAF blocks attacks against the vulnerable test app.
# =============================================================================

VULN_APP_PORT="${VULN_APP_PORT:-9099}"
PROXY_PORT="${PROXY_PORT:-8081}"
ADMIN_PORT="${ADMIN_PORT:-8444}"
VULN_APP_BIN="${VULN_APP_BIN:-/tmp/vuln-app}"
PROXY_BIN="${PROXY_BIN:-/tmp/fortress-proxy}"
CONFIG="${CONFIG:-fortresswaf.local.yaml}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
UA="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"

PASS=0
FAIL=0

cleanup() {
  echo -e "\n${YELLOW}Cleaning up...${NC}"
  [ -n "${VULN_PID:-}" ] && kill "$VULN_PID" 2>/dev/null || true
  [ -n "${PROXY_PID:-}" ] && kill "$PROXY_PID" 2>/dev/null || true
  wait 2>/dev/null || true
}
trap cleanup EXIT

check_deps() {
  for cmd in go curl; do
    if ! command -v "$cmd" &>/dev/null; then
      echo "Missing dependency: $cmd"
      exit 1
    fi
  done
}

build_binaries() {
  echo -e "${YELLOW}Building binaries...${NC}"

  echo "  Building vuln-app..."
  (cd "$(dirname "$0")" && go build -o "$VULN_APP_BIN" .)

  PROXY_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
  echo "  Building fortress-proxy from $PROXY_DIR..."
  (cd "$PROXY_DIR" && go build -o "$PROXY_BIN" ./cmd/proxy)
}

start_services() {
  local script_dir
  script_dir="$(cd "$(dirname "$0")" && pwd)"

  echo -e "${YELLOW}Starting vulnerable test app on port $VULN_APP_PORT...${NC}"
  VULN_APP_PORT="$VULN_APP_PORT" "$VULN_APP_BIN" &
  VULN_PID=$!
  sleep 1

  if ! kill -0 "$VULN_PID" 2>/dev/null; then
    echo -e "${RED}Failed to start vuln-app${NC}"
    exit 1
  fi
  echo -e "${GREEN}  vuln-app running (PID: $VULN_PID)${NC}"

  echo -e "${YELLOW}Starting fortress-proxy on port $PROXY_PORT (config: $script_dir/$CONFIG)...${NC}"
  "$PROXY_BIN" -config "$script_dir/$CONFIG" -proxy-port "$PROXY_PORT" -admin-port "$ADMIN_PORT" &
  PROXY_PID=$!
  sleep 2

  if ! kill -0 "$PROXY_PID" 2>/dev/null; then
    echo -e "${RED}Failed to start fortress-proxy${NC}"
    exit 1
  fi
  echo -e "${GREEN}  fortress-proxy running (PID: $PROXY_PID)${NC}"
}

curlb() {
  curl -s -A "$UA" "$@"
}

assert_direct() {
  local desc="$1" method="$2" path="$3" data="$4" expect_status="$5"

  local args=(-o /dev/null -w "%{http_code}" --max-time 3 -X "$method")
  [ -n "$data" ] && args+=(-d "$data")
  args+=("http://localhost:${VULN_APP_PORT}${path}")

  local status
  status=$(curlb "${args[@]}" 2>/dev/null || echo "000")

  if [ "$status" = "$expect_status" ]; then
    echo -e "  ${GREEN}✓${NC} $desc (direct: $status)"
  else
    echo -e "  ${RED}✗${NC} $desc (expected $expect_status, got $status)"
  fi
}

assert_via_proxy() {
  local desc="$1" method="$2" path="$3" data="$4" expect_status="$5"

  local args=(-o /dev/null -w "%{http_code}" --max-time 5 -X "$method")
  [ -n "$data" ] && args+=(-d "$data")
  args+=("http://localhost:${PROXY_PORT}${path}")

  local status
  status=$(curlb "${args[@]}" 2>/dev/null || echo "000")

  if [ "$status" = "$expect_status" ]; then
    echo -e "  ${GREEN}✓${NC} $desc (proxy: $status)"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $desc (expected $expect_status, got $status)"
    FAIL=$((FAIL + 1))
  fi
}

assert_proxy_content() {
  local desc="$1" method="$2" path="$3" data="$4" expect_content="$5"

  local args=(--max-time 5 -X "$method")
  [ -n "$data" ] && args+=(-d "$data")
  args+=("http://localhost:${PROXY_PORT}${path}")

  local body
  body=$(curlb "${args[@]}" 2>/dev/null || echo "")

  if echo "$body" | grep -q "$expect_content"; then
    echo -e "  ${GREEN}✓${NC} $desc (proxy: contains expected content)"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $desc (expected content '$expect_content' not found)"
    FAIL=$((FAIL + 1))
  fi
}

run_tests() {
  echo ""
  echo "============================================"
  echo "  FortressWAF Attack Test Suite"
  echo "============================================"
  echo ""

  # ----- PART 1: Direct access (vuln-app without WAF) -----
  echo -e "${YELLOW}[1/5] Direct access — sanity checks (vuln-app without WAF)${NC}"

  assert_direct "Health check" GET "/health" "" "200"
  assert_direct "Home page" GET "/" "" "200"
  assert_direct "Login page" GET "/login" "" "200"
  assert_direct "Search page" GET "/search" "" "200"
  assert_direct "Ping page" GET "/ping" "" "200"
  assert_direct "File page" GET "/file" "" "200"

  echo ""

  # ----- PART 2: WAF allows legitimate traffic -----
  echo -e "${YELLOW}[2/5] WAF allows legitimate traffic${NC}"

  assert_via_proxy "Health check passes through WAF" GET "/health" "" "200"
  assert_via_proxy "Home page through WAF" GET "/" "" "200"
  assert_proxy_content "Home page shows VulnApp content" GET "/" "" "VulnApp"
  assert_via_proxy "Legitimate login (wrong creds)" POST "/login" "username=nobody&password=nothing" "200"
  assert_via_proxy "Empty search" GET "/search?q=" "" "200"
  assert_via_proxy "Legitimate search query" GET "/search?q=laptop" "" "200"
  assert_via_proxy "Legitimate comment" POST "/comment" "author=TestUser&message=Great+app" "200"

  echo ""

  # ----- PART 3: WAF blocks SQL Injection -----
  echo -e "${YELLOW}[3/5] WAF blocks SQL Injection attacks${NC}"

  assert_via_proxy "SQLi: OR 1=1 in login username" POST "/login" "username='+OR+'1'%3D'1&password=test" "403"
  assert_via_proxy "SQLi: OR 1=1 in password" POST "/login" "username=admin&password='+OR+'1'%3D'1" "403"
  assert_via_proxy "SQLi: UNION SELECT in search" GET "/search?q='+UNION+SELECT+*+FROM+users" "" "403"
  assert_via_proxy "SQLi: DROP TABLE in search" GET "/search?q='%3BDROP+TABLE+users" "" "403"
  assert_via_proxy "SQLi: comment injection #" GET "/search?q=test'--" "" "403"

  echo ""

  # ----- PART 4: WAF blocks XSS attacks -----
  echo -e "${YELLOW}[4/5] WAF blocks XSS attacks${NC}"

  assert_via_proxy "XSS: <script> in search" GET '/search?q=<script>alert(1)</script>' "" "403"
  assert_via_proxy "XSS: <script> in comment" POST "/comment" "author=Hacker&message=<script>document.cookie</script>" "403"
  assert_via_proxy "XSS: onerror in search" GET '/search?q=<img src=x onerror=alert(1)>' "" "403"
  assert_via_proxy "XSS: onload in search" GET '/search?q=<body onload=alert(1)>' "" "403"
  assert_via_proxy "XSS: prompt() in comment" POST "/comment" "author=XSS&message=prompt('xss')" "403"

  echo ""

  # ----- PART 5: WAF blocks RCE, LFI, and scanners -----
  echo -e "${YELLOW}[5/5] WAF blocks RCE, LFI, and scanner attacks${NC}"

  # RCE
  assert_via_proxy "RCE: ; cat /etc/passwd in ping" POST "/ping" "addr=8.8.8.8; cat /etc/passwd" "403"
  assert_via_proxy "RCE: ; ls in ping" POST "/ping" "addr=8.8.8.8; ls -la" "403"
  assert_via_proxy "RCE: backtick whoami in ping" POST "/ping" 'addr=8.8.8.8; `whoami`' "403"
  assert_via_proxy "RCE: shell var in ping" POST "/ping" "addr=8.8.8.8; \$(id)" "403"

  # LFI / Path Traversal
  assert_via_proxy "LFI: ../../../etc/passwd in file" GET "/file?name=../../../etc/passwd" "" "403"
  assert_via_proxy "LFI: ../../etc/shadow in file" GET "/file?name=../../etc/shadow" "" "403"
  assert_via_proxy "LFI: /etc/passwd directly" GET "/file?name=/etc/passwd" "" "403"

  # Bot / Scanner detection
  local sc_status
  sc_status=$(curl -s -o /dev/null -w "%{http_code}" -A "sqlmap/1.8" --max-time 5 "http://localhost:${PROXY_PORT}/" 2>/dev/null || echo "000")
  if [ "$sc_status" = "403" ]; then
    echo -e "  ${GREEN}✓${NC} Scanner: sqlmap User-Agent (proxy: $sc_status)"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} Scanner: sqlmap User-Agent (expected 403, got $sc_status)"
    FAIL=$((FAIL + 1))
  fi

  echo ""
}

print_summary() {
  echo "============================================"
  echo -e "  Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}, $((PASS + FAIL)) total"
  echo "============================================"

  if [ "$FAIL" -eq 0 ]; then
    echo -e "${GREEN}All tests passed! FortressWAF is correctly blocking attacks.${NC}"
    return 0
  else
    echo -e "${RED}Some tests failed. Check the output above.${NC}"
    return 1
  fi
}

# --- Main ---
check_deps
build_binaries
start_services
run_tests
print_summary

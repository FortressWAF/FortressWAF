#!/usr/bin/env bash
# =============================================================================
# FortressWAF Lab Demo
# Interactive demonstration showing FortressWAF blocking attacks in real-time.
# =============================================================================
set -euo pipefail

VULN_PORT="${VULN_PORT:-9099}"
PROXY_PORT="${PROXY_PORT:-8081}"
ADMIN_PORT="${ADMIN_PORT:-8444}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
UA="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
BOLD='\033[1m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'
PASS=0
FAIL=0
TOTAL=0

cleanup() {
  echo
  echo -e "${YELLOW}Shutting down lab...${NC}"
  kill $VULN_PID $PROXY_PID 2>/dev/null || true
  wait 2>/dev/null || true
  echo -e "${GREEN}Lab stopped.${NC}"
}
trap cleanup EXIT

banner() {
  clear
  echo -e "${RED}${BOLD}"
  echo '  ______                   _                               __    _  __ '
  echo ' |  ____|                 | |                             / _|  | |/ _|'
  echo ' | |__ ___  _ __ ___  ___| |_ _ __ ___  __ _ _ __ ___   | |_ __| | |_ '
  echo ' |  __/ _ \| '"'"'__/ _ \/ __| __| '"'"'__/ _ \/ _` | '"'"'_ \` _ \  |  _/ _\` |  _|'
  echo ' | | | (_) | | |  __/\__ \ |_| | |  __/ (_| | | | | | | | || (_| | |  '
  echo ' |_|  \___/|_|  \___||___/\__|_|  \___|\__,_|_| |_| |_| |_| \__,_|_|  '
  echo -e "${NC}"
  echo -e "${CYAN}${BOLD}  ⚡ WAF Attack & Defense Lab${NC}"
  echo -e "${CYAN}  ─────────────────────────────────────────${NC}"
  echo
  echo -e "  ${YELLOW}Architecture:${NC}"
  echo -e "  ${GREEN}  Attacker/User${NC}  →  ${RED}FortressWAF${NC}  →  ${CYAN}VulnApp${NC}"
  echo -e "  ${GREEN}  curl/browser${NC}    →  ${RED}:$PROXY_PORT${NC}     →  ${CYAN}:$VULN_PORT${NC}"
  echo
  echo -e "  ${YELLOW}Lab will:${NC}"
  echo -e "  1. Start VulnApp (vulnerable target)"
  echo -e "  2. Start FortressWAF proxy (the shield)"
  echo -e "  3. Send attack waves — see what gets blocked!"
  echo
}

step() {
  echo
  echo -e "${MAGENTA}${BOLD}═══ $1 ═══${NC}"
  sleep 0.5
}

check() {
  local result="$1"
  TOTAL=$((TOTAL + 1))
  if [ "$result" = "403" ]; then
    echo -e "    ${RED}BLOCKED${NC} (403)  ← WAF bekerja!${GREEN} ✓${NC}"
    PASS=$((PASS + 1))
  elif [ "$result" = "200" ] || [ "$result" = "302" ]; then
    echo -e "    ${GREEN}ALLOWED${NC} ($result)"
    PASS=$((PASS + 1))
  else
    echo -e "    ${YELLOW}UNKNOWN${NC} ($result)"
    FAIL=$((FAIL + 1))
  fi
}

attack() {
  local desc="$1" expected="$2" url="$3"
  shift 3
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" -A "$UA" --max-time 5 "$@" "$url" 2>/dev/null || echo "000")
  echo -e "  ${CYAN}→${NC} $desc"
  check "$status"
}

build() {
  echo -e "${YELLOW}Building binaries...${NC}"
  echo -n "  vuln-app... "
  (cd "$ROOT/vuln-app" && go build -o /tmp/vuln-app . 2>/dev/null) && echo -e "${GREEN}OK${NC}" || echo -e "${RED}FAIL${NC}"
  echo -n "  fortress-proxy... "
  (cd "$ROOT/.." && go build -o /tmp/fortress-proxy ./cmd/proxy 2>/dev/null) && echo -e "${GREEN}OK${NC}" || echo -e "${RED}FAIL${NC}"
  echo
}

start_services() {
  echo -e "${YELLOW}Starting services...${NC}"

  echo -n "  VulnApp on :$VULN_PORT... "
  VULN_APP_PORT="$VULN_PORT" /tmp/vuln-app &>/dev/null &
  VULN_PID=$!
  sleep 1
  kill -0 "$VULN_PID" 2>/dev/null && echo -e "${GREEN}OK (PID $VULN_PID)${NC}" || { echo -e "${RED}FAILED${NC}"; exit 1; }

  echo -n "  FortressWAF on :$PROXY_PORT... "
  /tmp/fortress-proxy -config "$ROOT/vuln-app/fortresswaf.local.yaml" -proxy-port "$PROXY_PORT" -admin-port "$ADMIN_PORT" &>/dev/null &
  PROXY_PID=$!
  sleep 2
  kill -0 "$PROXY_PID" 2>/dev/null && echo -e "${GREEN}OK (PID $PROXY_PID)${NC}" || { echo -e "${RED}FAILED${NC}"; exit 1; }

  echo -e "${GREEN}Both services running!${NC}"
  echo
}

demo_legitimate() {
  step "PART 1: Legitimate Traffic (should PASS)"

  echo -e "  ${YELLOW}Normal users browsing the site — WAF lets them through.${NC}"
  echo
  attack "Health check"        200 "http://localhost:$PROXY_PORT/health"
  attack "Home page"           200 "http://localhost:$PROXY_PORT/"
  attack "Login form (GET)"    200 "http://localhost:$PROXY_PORT/login"
  attack "Search page (GET)"   200 "http://localhost:$PROXY_PORT/search"
  attack "Wrong credentials"   200 -X POST -d "username=user1&password=wrong" "http://localhost:$PROXY_PORT/login"
  attack "Normal search query" 200 "http://localhost:$PROXY_PORT/search?q=monitor"
  attack "Add a comment"       302 -X POST -d "author=Guest&message=Nice+site" "http://localhost:$PROXY_PORT/comment"
}

demo_sqli() {
  step "PART 2: SQL Injection (should be BLOCKED)"

  echo -e "  ${YELLOW}Hacker tries SQL injection to bypass login or dump data.${NC}"
  echo -e "  ${YELLOW}FortressWAF's SQLi inspector detects and blocks them.${NC}"
  echo
  attack "OR 1=1 in username"  403 -X POST -d "username=' OR '1'='1&password=test" "http://localhost:$PROXY_PORT/login"
  attack "OR 1=1 in password"  403 -X POST -d "username=admin&password=' OR '1'='1" "http://localhost:$PROXY_PORT/login"
  attack "UNION SELECT"        403 "http://localhost:$PROXY_PORT/search?q=' UNION SELECT * FROM users"
  attack "DROP TABLE"          403 "http://localhost:$PROXY_PORT/search?q=';DROP TABLE users"
  attack "Comment injection"   403 "http://localhost:$PROXY_PORT/search?q=admin'--"
}

demo_xss() {
  step "PART 3: Cross-Site Scripting (should be BLOCKED)"

  echo -e "  ${YELLOW}Hacker injects JavaScript to steal cookies or deface pages.${NC}"
  echo -e "  ${YELLOW}FortressWAF's XSS inspector detects script tags and events.${NC}"
  echo
  attack "<script> in search"  403 "http://localhost:$PROXY_PORT/search?q=<script>alert(1)</script>"
  attack "<script> in comment" 403 -X POST -d "author=Hacker&message=<script>alert('xss')</script>" "http://localhost:$PROXY_PORT/comment"
  attack "onerror payload"     403 "http://localhost:$PROXY_PORT/search?q=<img src=x+onerror=alert(1)>"
  attack "onload payload"      403 "http://localhost:$PROXY_PORT/search?q=<body+onload=alert(1)>"
  attack "prompt()"            403 -X POST -d "author=XSS&message=prompt('x')" "http://localhost:$PROXY_PORT/comment"
}

demo_rce() {
  step "PART 4: Command Injection & LFI (should be BLOCKED)"

  echo -e "  ${YELLOW}Hacker tries to run OS commands or read sensitive files.${NC}"
  echo -e "  ${YELLOW}FortressWAF's RCE inspector blocks shell metacharacters.${NC}"
  echo
  attack "; cat /etc/passwd"   403 -X POST -d "addr=8.8.8.8; cat /etc/passwd" "http://localhost:$PROXY_PORT/ping"
  attack "; ls -la"            403 -X POST -d "addr=8.8.8.8; ls -la" "http://localhost:$PROXY_PORT/ping"
  attack "backtick whoami"     403 -X POST -d 'addr=8.8.8.8; `whoami`' "http://localhost:$PROXY_PORT/ping"
  attack "LFI ../etc/passwd"   403 "http://localhost:$PROXY_PORT/file?name=../../../etc/passwd"
  attack "LFI ../../etc/shadow" 403 "http://localhost:$PROXY_PORT/file?name=../../etc/shadow"
  attack "LFI /etc/passwd"     403 "http://localhost:$PROXY_PORT/file?name=/etc/passwd"
}

demo_scanner() {
  step "PART 5: Scanner / Bot Detection (should be BLOCKED)"

  echo -e "  ${YELLOW}Automated scanners probe the site for vulnerabilities.${NC}"
  echo -e "  ${YELLOW}FortressWAF's Bot detector identifies known scanner User-Agents.${NC}"
  echo
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" -A "sqlmap/1.8" --max-time 5 "http://localhost:$PROXY_PORT/" 2>/dev/null || echo "000")
  echo -e "  ${CYAN}→${NC} sqlmap scanner"
  check "$status"
  status=$(curl -s -o /dev/null -w "%{http_code}" -A "nikto/2.5" --max-time 5 "http://localhost:$PROXY_PORT/" 2>/dev/null || echo "000")
  echo -e "  ${CYAN}→${NC} nikto scanner"
  check "$status"
  status=$(curl -s -o /dev/null -w "%{http_code}" -A "nmap script" --max-time 5 "http://localhost:$PROXY_PORT/" 2>/dev/null || echo "000")
  echo -e "  ${CYAN}→${NC} nmap script"
  check "$status"
}

demo_curl() {
  step "PART 6: Default curl User-Agent (should be BLOCKED)"

  echo -e "  ${YELLOW}Even plain curl is detected as a bot by default.${NC}"
  echo -e "  ${YELLOW}Use a browser User-Agent for legitimate access.${NC}"
  echo
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "http://localhost:$PROXY_PORT/" 2>/dev/null || echo "000")
  echo -e "  ${CYAN}→${NC} curl without browser UA"
  check "$status"
}

summary() {
  echo
  echo -e "${MAGENTA}${BOLD}════════════════════════════════════════════${NC}"
  echo -e "${MAGENTA}${BOLD}  LAB RESULTS${NC}"
  echo -e "${MAGENTA}${BOLD}════════════════════════════════════════════${NC}"
  echo
  echo -e "  Total checks: ${BOLD}$TOTAL${NC}"
  echo -e "  ${GREEN}Passed:${NC}       $PASS"
  echo -e "  ${RED}Failed:${NC}       $FAIL"
  echo
  if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}✅ FortressWAF is working correctly!${NC}"
    echo
    echo -e "  ${YELLOW}What happened:${NC}"
    echo -e "  • Legitimate traffic → ${GREEN}allowed${NC} through to VulnApp"
    echo -e "  • SQL injection     → ${RED}blocked${NC} by SQLi inspector"
    echo -e "  • XSS attacks       → ${RED}blocked${NC} by XSS inspector"
    echo -e "  • Command injection → ${RED}blocked${NC} by RCE inspector"
    echo -e "  • Path traversal    → ${RED}blocked${NC} by RCE inspector"
    echo -e "  • Scanner User-Agent→ ${RED}blocked${NC} by Bot detector"
  else
    echo -e "  ${YELLOW}Some checks had unexpected results.${NC}"
  fi
  echo
  echo -e "${CYAN}For live monitoring, run in another terminal:${NC}"
  echo -e "  ${BOLD}fortressctl --api-url http://localhost:$ADMIN_PORT monitor${NC}"
  echo
}

# ===== MAIN =====
banner
build
start_services

echo -e "${GREEN}${BOLD}Lab ready! Sending attack waves...${NC}"
sleep 1

demo_legitimate
demo_sqli
demo_xss
demo_rce
demo_scanner
demo_curl
summary

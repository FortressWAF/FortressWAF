#!/usr/bin/env bash
# =============================================================================
# FortressWAF Lab Launcher
# Quick-start the lab environment with live monitoring
# =============================================================================
set -euo pipefail

VULN_PORT="${VULN_PORT:-9099}"
PROXY_PORT="${PROXY_PORT:-8081}"
ADMIN_PORT="${ADMIN_PORT:-8444}"
ROOT="$(cd "$(dirname "$0")" && pwd)"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

cleanup() {
  echo
  echo -e "${YELLOW}Shutting down lab...${NC}"
  kill $VULN_PID $PROXY_PID 2>/dev/null || true
  wait 2>/dev/null || true
  echo -e "${GREEN}Done.${NC}"
}
trap cleanup EXIT

# Build if needed
if [ ! -f /tmp/vuln-app ] || [ ! -f /tmp/fortress-proxy ]; then
  echo -e "${YELLOW}Building binaries...${NC}"
  (cd "$ROOT/vuln-app" && go build -o /tmp/vuln-app .)
  (cd "$ROOT/.." && go build -o /tmp/fortress-proxy ./cmd/proxy)
  echo -e "${GREEN}Built.${NC}"
fi

# Start vuln-app
echo -n "VulnApp... "
VULN_APP_PORT="$VULN_PORT" /tmp/vuln-app &>/tmp/vuln-app.log &
VULN_PID=$!
sleep 1
kill -0 "$VULN_PID" 2>/dev/null && echo -e "${GREEN}PID $VULN_PID${NC}" || { echo -e "${RED}FAILED${NC}"; exit 1; }

# Start proxy
echo -n "FortressWAF... "
/tmp/fortress-proxy -config "$ROOT/vuln-app/fortresswaf.local.yaml" -proxy-port "$PROXY_PORT" -admin-port "$ADMIN_PORT" &>/tmp/fortress-proxy.log &
PROXY_PID=$!
sleep 2
kill -0 "$PROXY_PID" 2>/dev/null && echo -e "${GREEN}PID $PROXY_PID${NC}" || { echo -e "${RED}FAILED${NC}"; exit 1; }

echo
echo -e "${GREEN}${BOLD}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}${BOLD}║     FortressWAF Lab is RUNNING              ║${NC}"
echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════════╝${NC}"
echo
echo -e "  ${CYAN}VulnApp (target):${NC}  http://localhost:$VULN_PORT"
echo -e "  ${CYAN}WAF Proxy:${NC}        http://localhost:$PROXY_PORT"
echo -e "  ${CYAN}Admin API:${NC}        http://localhost:$ADMIN_PORT"
echo
echo -e "  ${YELLOW}Test the WAF:${NC}"
echo -e "    curl -A '${BOLD}Mozilla/5.0${NC}' http://localhost:$PROXY_PORT/"
echo -e "    curl -A '${BOLD}sqlmap/1.8${NC}' http://localhost:$PROXY_PORT/"
echo -e "    curl -X POST -d \"${BOLD}username=' OR '1'='1${NC}\" http://localhost:$PROXY_PORT/login"
echo
echo -e "  ${YELLOW}Live monitor:${NC}"
echo -e "    ${BOLD}fortressctl --api-url http://localhost:$ADMIN_PORT stats${NC}"
echo -e "    ${BOLD}fortressctl --api-url http://localhost:$ADMIN_PORT monitor${NC}"
echo
echo -e "  ${YELLOW}Run demo:${NC}"
echo -e "    ${BOLD}bash $ROOT/demo.sh${NC}"
echo
echo -e "  ${YELLOW}Logs:${NC}"
echo -e "    tail -f /tmp/fortress-proxy.log"
echo
echo -e "${RED}Press Ctrl+C to stop the lab${NC}"

wait

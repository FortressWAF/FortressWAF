#!/bin/bash
# ============================================================================
# FortressWAF - Docker Quick Start Script
# ============================================================================
# Usage: ./scripts/docker-start.sh [dev|prod|monitoring|stop|clean]
# ============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

print_banner() {
    echo -e "${CYAN}"
    echo "  ╔═══════════════════════════════════════════════════════╗"
    echo "  ║                                                       ║"
    echo "  ║     ███████╗ ██████╗ ██████╗ ████████╗               ║"
    echo "  ║     ██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝               ║"
    echo "  ║     █████╗  ██║   ██║██████╔╝   ██║                  ║"
    echo "  ║     ██╔══╝  ██║   ██║██╔══██╗   ██║                  ║"
    echo "  ║     ██║     ╚██████╔╝██║  ██║   ██║                  ║"
    echo "  ║     ╚═╝      ╚═════╝ ╚═╝  ╚═╝   ╚═╝                  ║"
    echo "  ║                    FortressWAF                        ║"
    echo "  ║          Enterprise Web Application Firewall          ║"
    echo "  ║                                                       ║"
    echo "  ╚═══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_dependencies() {
    local missing=()

    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    fi

    if ! docker compose version &> /dev/null 2>&1; then
        missing+=("docker-compose-plugin")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${RED}Error: Missing required dependencies: ${missing[*]}${NC}"
        echo -e "${YELLOW}Install Docker: https://docs.docker.com/engine/install/${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ All dependencies found${NC}"
}

setup_env() {
    if [ ! -f "${PROJECT_ROOT}/.env" ]; then
        echo -e "${YELLOW}No .env file found. Creating from .env.example...${NC}"
        cp "${PROJECT_ROOT}/.env.example" "${PROJECT_ROOT}/.env"
        echo -e "${GREEN}✓ Created .env file${NC}"
        echo -e "${YELLOW}⚠ Please review and update passwords in .env before production use!${NC}"
    else
        echo -e "${GREEN}✓ .env file found${NC}"
    fi
}

start_dev() {
    echo -e "${BLUE}Starting development stack...${NC}"
    docker compose -f "${PROJECT_ROOT}/deploy/docker-compose.dev.yml" up -d --build

    echo ""
    echo -e "${GREEN}${BOLD}FortressWAF Dev Stack Running!${NC}"
    echo -e "══════════════════════════════════════════"
    echo -e "  ${CYAN}Proxy:${NC}      http://localhost:8080"
    echo -e "  ${CYAN}Admin API:${NC}  http://localhost:8443"
    echo -e "  ${CYAN}Dashboard:${NC}  http://localhost:3000"
    echo -e "  ${CYAN}ML Engine:${NC}  http://localhost:8000"
    echo -e "  ${CYAN}Postgres:${NC}   localhost:5432"
    echo -e "  ${CYAN}Redis:${NC}      localhost:6379"
    echo -e "  ${CYAN}NATS:${NC}       localhost:4222"
    echo -e "══════════════════════════════════════════"
    echo ""
    echo -e "${YELLOW}View logs:${NC} docker compose -f deploy/docker-compose.dev.yml logs -f"
}

start_prod() {
    echo -e "${BLUE}Starting production stack...${NC}"
    docker compose -f "${PROJECT_ROOT}/deploy/docker-compose.yml" up -d

    echo ""
    echo -e "${GREEN}${BOLD}FortressWAF Production Stack Running!${NC}"
    echo -e "══════════════════════════════════════════"
    echo -e "  ${CYAN}Proxy:${NC}      http://localhost:${PROXY_HTTP_PORT:-80}"
    echo -e "  ${CYAN}Admin API:${NC}  https://localhost:${PROXY_ADMIN_PORT:-8443}"
    echo -e "  ${CYAN}Dashboard:${NC}  http://localhost:${DASHBOARD_PORT:-3000}"
    echo -e "  ${CYAN}ML Engine:${NC}  http://localhost:${ML_ENGINE_PORT:-8000}"
    echo -e "══════════════════════════════════════════"
}

start_monitoring() {
    echo -e "${BLUE}Starting production + monitoring stack...${NC}"
    docker compose -f "${PROJECT_ROOT}/deploy/docker-compose.yml" \
                   -f "${PROJECT_ROOT}/deploy/monitoring/docker-compose.monitoring.yml" up -d

    echo ""
    echo -e "${GREEN}${BOLD}FortressWAF + Monitoring Running!${NC}"
    echo -e "══════════════════════════════════════════"
    echo -e "  ${CYAN}Prometheus:${NC}   http://localhost:${PROMETHEUS_PORT:-9090}"
    echo -e "  ${CYAN}Grafana:${NC}      http://localhost:${GRAFANA_PORT:-3001}"
    echo -e "  ${CYAN}Alertmanager:${NC} http://localhost:${ALERTMANAGER_PORT:-9093}"
    echo -e "══════════════════════════════════════════"
}

stop_all() {
    echo -e "${YELLOW}Stopping all FortressWAF stacks...${NC}"
    docker compose -f "${PROJECT_ROOT}/deploy/docker-compose.yml" down 2>/dev/null || true
    docker compose -f "${PROJECT_ROOT}/deploy/docker-compose.dev.yml" down 2>/dev/null || true
    docker compose -f "${PROJECT_ROOT}/deploy/monitoring/docker-compose.monitoring.yml" down 2>/dev/null || true
    echo -e "${GREEN}✓ All stacks stopped${NC}"
}

clean_all() {
    echo -e "${RED}${BOLD}WARNING: This will remove ALL FortressWAF data volumes!${NC}"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker compose -f "${PROJECT_ROOT}/deploy/docker-compose.yml" down -v --rmi local 2>/dev/null || true
        docker compose -f "${PROJECT_ROOT}/deploy/docker-compose.dev.yml" down -v --rmi local 2>/dev/null || true
        docker compose -f "${PROJECT_ROOT}/deploy/monitoring/docker-compose.monitoring.yml" down -v --rmi local 2>/dev/null || true
        docker image rm fortresswaf/proxy:latest fortresswaf/ml-engine:latest fortresswaf/dashboard:latest 2>/dev/null || true
        echo -e "${GREEN}✓ All resources cleaned${NC}"
    else
        echo -e "${YELLOW}Cancelled.${NC}"
    fi
}

show_status() {
    echo -e "${BOLD}FortressWAF Container Status:${NC}"
    echo "══════════════════════════════════════════"
    docker ps --filter "name=fortress-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "No containers running."
}

usage() {
    echo -e "${BOLD}Usage:${NC} $0 [command]"
    echo ""
    echo -e "${BOLD}Commands:${NC}"
    echo -e "  ${CYAN}dev${NC}         Start development stack (hot-reload)"
    echo -e "  ${CYAN}prod${NC}        Start production stack"
    echo -e "  ${CYAN}monitoring${NC}  Start production + monitoring"
    echo -e "  ${CYAN}status${NC}      Show container status"
    echo -e "  ${CYAN}stop${NC}        Stop all stacks"
    echo -e "  ${CYAN}clean${NC}       Remove all data & images"
    echo ""
}

# ============================================================================
# Main
# ============================================================================

print_banner
check_dependencies

# Load .env if it exists
if [ -f "${PROJECT_ROOT}/.env" ]; then
    set -a
    source "${PROJECT_ROOT}/.env"
    set +a
fi

case "${1:-}" in
    dev)
        setup_env
        start_dev
        ;;
    prod)
        setup_env
        start_prod
        ;;
    monitoring)
        setup_env
        start_monitoring
        ;;
    status)
        show_status
        ;;
    stop)
        stop_all
        ;;
    clean)
        clean_all
        ;;
    *)
        usage
        ;;
esac

# Docker Deployment

This guide covers production Docker deployment scenarios for FortressWAF.

## Docker Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Docker Host                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                   FortressWAF Network                         ││
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     ││
│  │  │ FortressWAF │  │   Redis      │  │  PostgreSQL  │     ││
│  │  │   (WAF)     │◀─▶│  (State)    │  │   (Data)    │     ││
│  │  │   Port 8443 │  │   Port 6379 │  │   Port 5432 │     ││
│  │  └──────────────┘  └──────────────┘  └──────────────┘     ││
│  │         ▲                  │                  │              ││
│  │         │                  │                  │              ││
│  │         ▼                  ▼                  ▼              ││
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     ││
│  │  │  Prometheus  │  │   Grafana    │  │   Optional   │     ││
│  │  │  (Metrics)   │  │ (Dashboard)  │  │   Services   │     ││
│  │  └──────────────┘  └──────────────┘  └──────────────┘     ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Protected Applications                     ││
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     ││
│  │  │  Web App 1   │  │  Web App 2   │  │  API Server  │     ││
│  │  │  Port 8080   │  │  Port 8081   │  │  Port 8082   │     ││
│  │  └──────────────┘  └──────────────┘  └──────────────┘     ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Resource Requirements

| Deployment Size | CPU | Memory | Disk | Max RPS |
|----------------|-----|--------|------|---------|
| Development | 1 core | 1 GB | 10 GB | 1,000 |
| Small | 2 cores | 2 GB | 20 GB | 10,000 |
| Medium | 4 cores | 4 GB | 50 GB | 50,000 |
| Large | 8 cores | 8 GB | 100 GB | 100,000 |
| Production | 16 cores | 16 GB | 200 GB | 500,000 |

## Production Docker Compose

### Complete Production Configuration

```yaml
version: '3.8'

services:
  fortresswaf:
    image: fortresswaf/fortresswaf:2.0.0
    container_name: fortresswaf
    restart: unless-stopped
    ports:
      - "8443:8443"      # Dashboard
      - "8444:8444"      # API
      - "8080:8080"      # HTTP proxy
    environment:
      - FW_CONFIG_PATH=/app/config/config.yaml
      - FW_LOG_LEVEL=info
      - FW_SERVER_PORT=8443
      - FW_API_PORT=8444
      - FW_REDIS_HOST=redis
      - FW_REDIS_PORT=6379
      - FW_POSTGRES_HOST=postgres
      - FW_POSTGRES_PORT=5432
      - FW_POSTGRES_DB=fortresswaf
      - FW_POSTGRES_USER=fortresswaf
      - FW_POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - FW_ML_ENABLED=true
      - FW_TLS_ENABLED=true
    volumes:
      - ./config:/app/config:ro
      - ./logs:/app/logs
      - ./certs:/certs:ro
      - fortresswaf-data:/app/data
    depends_on:
      redis:
        condition: service_healthy
      postgres:
        condition: service_healthy
    networks:
      - fortresswaf-network
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
    healthcheck:
      test: ["CMD", "curl", "-kf", "https://localhost:8443/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 120s
    ulimits:
      nofile:
        soft: 65536
        hard: 65536

  redis:
    image: redis:7-alpine
    container_name: fortresswaf-redis
    restart: unless-stopped
    command:
      - redis-server
      - --appendonly yes
      - --maxmemory 1gb
      - --maxmemory-policy allkeys-lru
      - --save 900 1
      - --save 300 10
      - --save 60 10000
    volumes:
      - redis-data:/data
    networks:
      - fortresswaf-network
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 2G
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3

  postgres:
    image: postgres:15-alpine
    container_name: fortresswaf-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_DB=fortresswaf
      - POSTGRES_USER=fortresswaf
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_INITDB_ARGS=--encoding=UTF8 --locale=C
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./backups:/backups
    networks:
      - fortresswaf-network
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fortresswaf -d fortresswaf"]
      interval: 10s
      timeout: 5s
      retries: 3

  prometheus:
    image: prom/prometheus:v2.45.0
    container_name: fortresswaf-prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - fortresswaf-network
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 2G

  grafana:
    image: grafana/grafana:10.0.0
    container_name: fortresswaf-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
      - GF_USERS_ALLOW_SIGN_UP=false
      - GF_SERVER_ROOT_URL=https://grafana.localhost
      - GF_SERVER_SERVE_FROM_SUB_PATH=true
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
    networks:
      - fortresswaf-network
    depends_on:
      - prometheus

volumes:
  redis-data:
    driver: local
  postgres-data:
    driver: local
  prometheus-data:
    driver: local
  grafana-data:
    driver: local
  fortresswaf-data:
    driver: local

networks:
  fortresswaf-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets: []

scrape_configs:
  - job_name: 'fortresswaf'
    static_configs:
      - targets: ['fortresswaf:9090']
    metrics_path: /metrics
    scheme: https
    tls_config:
      insecure_skip_verify: true
```

## Image Registry Options

### Official Registry

```bash
# Pull from official Docker Hub
docker pull fortresswaf/fortresswaf:2.0.0
docker pull fortresswaf/fortresswaf:latest

# Pull specific version
docker pull fortresswaf/fortresswaf:2.0.0
```

### Private Registry

```bash
# Login to private registry
docker login registry.example.com

# Pull from private registry
docker pull registry.example.com/fortresswaf/fortresswaf:2.0.0

# Tag for private registry
docker tag fortresswaf/fortresswaf:2.0.0 registry.example.com/fortresswaf:2.0.0

# Push to private registry
docker push registry.example.com/fortresswaf:2.0.0
```

### Image Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `2.0.0` | Specific version |
| `2.0` | Minor version (2.0.x) |
| `2` | Major version (2.x.x) |
| `edge` | Latest development build |

## Volume Mounts

| Volume | Path | Description | Required |
|--------|------|-------------|----------|
| Config | `/app/config` | Configuration files | Yes |
| Logs | `/app/logs` | Application logs | Recommended |
| Certificates | `/certs` | TLS certificates | If TLS enabled |
| Data | `/app/data` | Runtime data | Recommended |

## Environment Variables

### Required Variables

```bash
# Database
FW_POSTGRES_HOST=postgres
FW_POSTGRES_PORT=5432
FW_POSTGRES_DB=fortresswaf
FW_POSTGRES_USER=fortresswaf
FW_POSTGRES_PASSWORD=secure_password

# Redis
FW_REDIS_HOST=redis
FW_REDIS_PORT=6379
```

### Optional Variables

```bash
# Server
FW_SERVER_PORT=8443
FW_API_PORT=8444
FW_TLS_ENABLED=true

# ML Engine
FW_ML_ENABLED=true
FW_ML_THRESHOLD=0.75

# Logging
FW_LOG_LEVEL=info
FW_LOG_FORMAT=json

# Performance
FW_WORKER_THREADS=auto
```

## SSL Certificate Handling

### Using Let's Encrypt

```yaml
# docker-compose.yml
services:
  certbot:
    image: certbot/certbot:latest
    volumes:
      - ./certs:/certs/live
      - ./webroot:/webroot
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew; sleep 86400; done'"

  fortresswaf:
    volumes:
      - ./certs:/certs:ro
```

### Self-Signed Certificates

```bash
# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -subj "/CN=fortresswaf.example.com/O=FortressWAF/C=US" \
  -addext "subjectAltName=DNS:fortresswaf.example.com,DNS:localhost,IP:127.0.0.1"
```

### Certificate Paths

```yaml
# config.yaml
server:
  tls:
    enabled: true
    cert_path: /certs/fullchain.pem
    key_path: /certs/privkey.pem
```

## Log Management

### JSON Logging

```yaml
logging:
  level: info
  format: json
  outputs:
    - type: file
      path: /app/logs/fortresswaf.log
      rotation:
        max_size: 100MB
        max_age: 7
        max_backups: 10
        compress: true
    - type: stdout
```

### Log Rotation

```yaml
# /etc/logrotate.d/fortresswaf
/home/fortresswaf/logs/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0640 fortresswaf fortresswaf
    postrotate
        docker-compose -f /home/fortresswaf/docker-compose.yml restart fortresswaf
    endscript
}
```

### Centralized Logging

```yaml
logging:
  outputs:
    - type: syslog
      host: syslog.example.com
      port: 514
      protocol: tcp
      facility: local0
```

## Backup and Restore

### Database Backup

```bash
# Create backup script
cat > backup.sh << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="./backups"
CONTAINER="fortresswaf-postgres"

mkdir -p $BACKUP_DIR

# Backup PostgreSQL
docker exec $CONTAINER pg_dump -U fortresswaf -d fortresswaf | gzip > $BACKUP_DIR/postgres_$DATE.sql.gz

# Backup Redis
docker exec fortresswaf-redis redis-cli SAVE
docker cp fortresswaf-redis:/data/dump.rdb $BACKUP_DIR/redis_$DATE.rdb

# Backup config
tar -czf $BACKUP_DIR/config_$DATE.tar.gz ./config

# Remove backups older than 30 days
find $BACKUP_DIR -type f -mtime +30 -delete

echo "Backup completed: $DATE"
EOF

chmod +x backup.sh
```

### Restore from Backup

```bash
# Restore PostgreSQL
gunzip < backups/postgres_20240101_120000.sql.gz | docker exec -i fortresswaf-postgres psql -U fortresswaf -d fortresswaf

# Restore Redis
docker cp backups/redis_20240101_120000.rdb fortresswaf-redis:/data/dump.rdb
docker exec fortresswaf-redis redis-cli DEBUG RELOAD
```

## Docker Swarm Deployment

```yaml
# docker-compose.swarm.yml
version: '3.8'

services:
  fortresswaf:
    image: fortresswaf/fortresswaf:2.0.0
    deploy:
      replicas: 3
      placement:
        constraints:
          - node.role == worker
      resources:
        limits:
          cpus: '2'
          memory: 4G
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
      restart_policy:
        condition: on-failure
        max_attempts: 3
    volumes:
      - config:/app/config:ro
      - logs:/app/logs
      - certs:/certs:ro
    networks:
      - fortresswaf-network
    ports:
      - "8443:8443"

volumes:
  config:
  logs:
  certs:

networks:
  fortresswaf-network:
    driver: overlay
    attachable: true
```

Deploy with:
```bash
docker stack deploy -c docker-compose.swarm.yml fortresswaf
```

## Health Checks

```yaml
# Add to docker-compose.yml
healthcheck:
  test: ["CMD", "curl", "-kf", "https://localhost:8443/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 120s
```

### Manual Health Check

```bash
# Check service health
curl -k https://localhost:8443/health

# Expected response:
# {"status": "healthy", "timestamp": "2024-01-01T00:00:00Z", "version": "2.0.0"}
```

## Upgrading

```bash
# 1. Pull new image
docker pull fortresswaf/fortresswaf:2.1.0

# 2. Stop services
docker-compose down

# 3. Update image tag in docker-compose.yml
# sed -i 's/fortresswaf:2.0.0/fortresswaf:2.1.0/g' docker-compose.yml

# 4. Start services
docker-compose up -d

# 5. Verify
docker-compose ps
curl -k https://localhost:8443/health
```

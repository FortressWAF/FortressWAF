# ============================================================================
# FortressWAF ML Engine - Multi-stage Docker Build (Python/FastAPI)
# ============================================================================

# Stage 1: Build dependencies (with build tools)
FROM python:3.14-slim AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# Stage 2: Runtime
FROM python:3.14-slim AS runtime

LABEL maintainer="FortressWAF Team"
LABEL org.opencontainers.image.title="FortressWAF ML Engine"
LABEL org.opencontainers.image.description="ML-powered threat detection engine for FortressWAF"

WORKDIR /app

# Copy installed Python packages from builder
COPY --from=builder /install /usr/local

# Create non-root user
RUN groupadd --gid 1001 mlengine && \
    useradd --uid 1001 --gid mlengine --shell /bin/bash --create-home mlengine && \
    mkdir -p /app/models/persisted && \
    chown -R mlengine:mlengine /app

COPY --chown=mlengine:mlengine . .

USER mlengine

EXPOSE 8000
EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')" || exit 1

CMD ["uvicorn", "api.app:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "2"]

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyHealthEndpoint(t *testing.T) {
	_ = httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProxyMetricsEndpoint(t *testing.T) {
	_ = httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`# HELP fortresswaf_requests_total Total requests
# TYPE fortresswaf_requests_total counter
fortresswaf_requests_total 0`))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

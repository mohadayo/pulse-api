package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupRouter() *http.ServeMux {
	store = NewStore()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ingest", ingestHandler)
	mux.HandleFunc("/aggregate", aggregateHandler)
	mux.HandleFunc("/reset", resetHandler)
	return mux
}

func TestHealthEndpoint(t *testing.T) {
	mux := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", resp["status"])
	}
	if resp["service"] != "aggregator" {
		t.Fatalf("expected service aggregator, got %v", resp["service"])
	}
}

func TestIngestMetric(t *testing.T) {
	mux := setupRouter()
	body, _ := json.Marshal(Metric{Name: "cpu", Value: 75.5})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
}

func TestIngestMissingName(t *testing.T) {
	mux := setupRouter()
	body, _ := json.Marshal(map[string]float64{"value": 10})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIngestInvalidJSON(t *testing.T) {
	mux := setupRouter()
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIngestMethodNotAllowed(t *testing.T) {
	mux := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/ingest", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestAggregateByName(t *testing.T) {
	mux := setupRouter()

	for _, v := range []float64{10, 20, 30} {
		body, _ := json.Marshal(Metric{Name: "mem", Value: v})
		req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/aggregate?name=mem", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var agg AggregatedMetric
	json.NewDecoder(w.Body).Decode(&agg)
	if agg.Count != 3 {
		t.Fatalf("expected count 3, got %d", agg.Count)
	}
	if agg.Sum != 60 {
		t.Fatalf("expected sum 60, got %f", agg.Sum)
	}
	if agg.Avg != 20 {
		t.Fatalf("expected avg 20, got %f", agg.Avg)
	}
	if agg.Min != 10 {
		t.Fatalf("expected min 10, got %f", agg.Min)
	}
	if agg.Max != 30 {
		t.Fatalf("expected max 30, got %f", agg.Max)
	}
}

func TestAggregateNotFound(t *testing.T) {
	mux := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/aggregate?name=nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAggregateAll(t *testing.T) {
	mux := setupRouter()
	for _, m := range []Metric{{Name: "a", Value: 1}, {Name: "b", Value: 2}} {
		body, _ := json.Marshal(m)
		req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/aggregate", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var aggs []AggregatedMetric
	json.NewDecoder(w.Body).Decode(&aggs)
	if len(aggs) != 2 {
		t.Fatalf("expected 2 aggregates, got %d", len(aggs))
	}
}

func TestResetStore(t *testing.T) {
	mux := setupRouter()
	body, _ := json.Marshal(Metric{Name: "x", Value: 5})
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodPost, "/reset", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]int
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["cleared"] != 1 {
		t.Fatalf("expected cleared 1, got %d", resp["cleared"])
	}
}

package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

type Metric struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp float64           `json:"timestamp"`
}

type AggregatedMetric struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Sum   float64 `json:"sum"`
	Avg   float64 `json:"avg"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type Store struct {
	mu      sync.RWMutex
	metrics map[string][]float64
}

func NewStore() *Store {
	return &Store{metrics: make(map[string][]float64)}
}

func (s *Store) Add(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics[name] = append(s.metrics[name], value)
}

func (s *Store) Aggregate(name string) *AggregatedMetric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values, ok := s.metrics[name]
	if !ok || len(values) == 0 {
		return nil
	}
	agg := &AggregatedMetric{
		Name:  name,
		Count: len(values),
		Min:   math.MaxFloat64,
		Max:   -math.MaxFloat64,
	}
	for _, v := range values {
		agg.Sum += v
		if v < agg.Min {
			agg.Min = v
		}
		if v > agg.Max {
			agg.Max = v
		}
	}
	agg.Avg = agg.Sum / float64(agg.Count)
	return agg
}

func (s *Store) AggregateAll() []AggregatedMetric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]AggregatedMetric, 0, len(s.metrics))
	for name := range s.metrics {
		s.mu.RUnlock()
		agg := s.Aggregate(name)
		s.mu.RLock()
		if agg != nil {
			results = append(results, *agg)
		}
	}
	return results
}

func (s *Store) Clear() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, v := range s.metrics {
		count += len(v)
	}
	s.metrics = make(map[string][]float64)
	return count
}

var store = NewStore()

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"service":   "aggregator",
		"timestamp": time.Now().Unix(),
	})
}

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var metric Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		log.Printf("[WARN] Invalid payload: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON payload"})
		return
	}
	if metric.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Field 'name' is required"})
		return
	}
	store.Add(metric.Name, metric.Value)
	log.Printf("[INFO] Ingested metric: %s = %f", metric.Name, metric.Value)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func aggregateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("name")
	w.Header().Set("Content-Type", "application/json")
	if name != "" {
		agg := store.Aggregate(name)
		if agg == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "No metrics found for " + name})
			return
		}
		json.NewEncoder(w).Encode(agg)
		return
	}
	json.NewEncoder(w).Encode(store.AggregateAll())
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	count := store.Clear()
	log.Printf("[INFO] Reset store, cleared %d values", count)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"cleared": count})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ingest", ingestHandler)
	mux.HandleFunc("/aggregate", aggregateHandler)
	mux.HandleFunc("/reset", resetHandler)

	log.Printf("[INFO] Starting aggregator on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("[FATAL] Server failed: %v", err)
	}
}

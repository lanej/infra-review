package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lanej/statecraft/internal/adapters/httpapi"
	"github.com/lanej/statecraft/internal/adapters/mock"
	"github.com/lanej/statecraft/internal/service"
)

func main() {
	store := mock.NewReviewStore()
	reviews := service.NewReviews(store)
	workflow := service.NewDemoWorkflow(store, mock.Planner{}, mock.Policies{}, mock.Policies{}, time.Now)

	mux := http.NewServeMux()
	demo := httpapi.DemoHandler(workflow, mock.SeedReview)
	mux.Handle("/api/demos", demo)
	mux.Handle("/api/demos/", demo)
	mux.HandleFunc("/api/reviews/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/reviews/")
		review, err := reviews.Get(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(review); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	port := os.Getenv("STATECRAFT_PORT")
	if port == "" {
		port = "8081"
	}
	address := net.JoinHostPort("127.0.0.1", port)
	log.Printf("Statecraft mock API listening on http://%s (no live infrastructure)", address)
	server := &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}

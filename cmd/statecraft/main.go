package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/lanej/infra-review/internal/adapters/mock"
	"github.com/lanej/infra-review/internal/service"
)

func main() {
	reviews := service.NewReviews(mock.NewReviewStore())

	mux := http.NewServeMux()
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
	log.Println("Statecraft API listening on http://127.0.0.1:8081")
	log.Fatal(http.ListenAndServe("127.0.0.1:8081", mux))
}

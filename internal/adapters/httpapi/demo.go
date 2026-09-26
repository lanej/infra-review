package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/lanej/statecraft/internal/domain"
	"github.com/lanej/statecraft/internal/ports"
	"github.com/lanej/statecraft/internal/service"
)

// DemoHandler exposes the isolated simulation, not the production adapters.
func DemoHandler(workflow *service.DemoWorkflow, seed func(string, time.Time) (domain.Review, error)) http.Handler {
	mux := http.NewServeMux()
	respond := func(w http.ResponseWriter, r domain.Review, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err != nil {
			status := http.StatusInternalServerError
			message := "The demo could not complete this action."
			switch {
			case errors.Is(err, ports.ErrNotFound):
				status = http.StatusNotFound
				message = err.Error()
			case errors.Is(err, ports.ErrConflict):
				status = http.StatusConflict
				message = err.Error()
			case errors.Is(err, service.ErrInvalid):
				status = http.StatusBadRequest
				message = err.Error()
			case errors.Is(err, service.ErrDenied):
				status = http.StatusUnprocessableEntity
				message = err.Error()
			case errors.Is(err, ports.ErrCapacity):
				status = http.StatusServiceUnavailable
				message = err.Error()
			}
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
			return
		}
		_ = json.NewEncoder(w).Encode(r)
	}
	decode := func(w http.ResponseWriter, r *http.Request, v any) error {
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if err := d.Decode(v); err != nil {
			return service.ErrInvalid
		}
		if err := d.Decode(new(any)); err != io.EOF {
			return service.ErrInvalid
		}
		return nil
	}
	mux.HandleFunc("POST /api/demos", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Scenario string `json:"scenario"`
		}
		if err := decode(w, r, &request); err != nil {
			respond(w, domain.Review{}, err)
			return
		}
		value, err := seed(request.Scenario, time.Now())
		if err != nil {
			respond(w, domain.Review{}, service.ErrInvalid)
			return
		}
		value, err = workflow.Create(r.Context(), value)
		respond(w, value, err)
	})
	mux.HandleFunc("GET /api/demos/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := workflow.Get(r.Context(), r.PathValue("id"))
		respond(w, value, err)
	})
	mux.HandleFunc("POST /api/demos/{id}/actions", func(w http.ResponseWriter, r *http.Request) {
		var cmd domain.WorkflowCommand
		if err := decode(w, r, &cmd); err != nil {
			respond(w, domain.Review{}, err)
			return
		}
		value, err := workflow.Act(r.Context(), r.PathValue("id"), cmd)
		respond(w, value, err)
	})
	// The demo has no authentication and must remain loopback-only. Reject browser
	// requests from other sites; JSON POSTs and no CORS prevent form-based writes.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "cross-site demo request rejected", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "application/json required", http.StatusUnsupportedMediaType)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

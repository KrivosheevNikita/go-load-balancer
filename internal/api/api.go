package api

import (
	"encoding/json"
	"net/http"

	"loadbalancer/internal/ratelimiter"
	"loadbalancer/internal/storage"
)

// Handler обрабатывает HTTP-запросы, связанные с клиентами
type Handler struct {
	store *ratelimiter.Store
}

func NewHandler(store *ratelimiter.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/clients", h.handleClients)
	mux.HandleFunc("/clients/", h.handleClient)
}

// Обрабатывает методы GET и POST по пути /clients
func (h *Handler) handleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Возвращаем JSON со всеми клиентами и их лимитами
		w.Header().Set("Content-Type", "application/json;")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(h.store.ListClients())

	case http.MethodPost:
		// Создание или обновление клиента
		var in struct {
			ClientID   string `json:"client_id"`
			Capacity   int64  `json:"capacity"`
			RatePerSec int64  `json:"rate_per_sec"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := h.store.AddClient(in.ClientID, storage.ClientConfig{
			ClientID:   in.ClientID,
			Capacity:   in.Capacity,
			RatePerSec: in.RatePerSec,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Обрабатывает DELETE-запросы по /clients/{id}
func (h *Handler) handleClient(w http.ResponseWriter, r *http.Request) {
	// Получаем client_id из URL
	id := r.URL.Path[len("/clients/"):]
	switch r.Method {
	case http.MethodDelete:
		// Удаление клиента
		if err := h.store.DeleteClient(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

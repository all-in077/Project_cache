package handler

import (
	"CacheProject/internal/cache"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	cache *cache.LRUcache
}

type setRequest struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

type response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func NewHandler(cache *cache.LRUcache) *Handler {
	return &Handler{cache: cache}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats := h.cache.Stats()
	writeJSON(w, http.StatusOK, response{Data: stats})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, response{Error: "key is required"})
		return
	}

	res, ok := h.cache.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, response{Error: "key not found"})
		return
	}

	writeJSON(w, http.StatusOK, response{Data: res})
}

func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	var req setRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: "invalid request body"})
		return
	}

	if req.Key == "" || req.Value == nil {
		writeJSON(w, http.StatusBadRequest, response{Error: "key and value are required"})
		return
	}

	h.cache.Set(req.Key, req.Value)
	writeJSON(w, http.StatusOK, response{Data: "ok"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, response{Error: "key is required"})
		return
	}

	ok := h.cache.Delete(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, response{Error: "key not found"})
		return
	}

	writeJSON(w, http.StatusOK, response{Data: "deleted"})
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/get", h.Get)
	r.Post("/set", h.Set)
	r.Delete("/delete", h.Delete)
	r.Get("/stats", h.Stats)
}

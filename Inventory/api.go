package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func itoa(i int) string { return strconv.Itoa(i) }

// RegisterAPI wires the JSON API. Mirrors the spirit of the original stuff-org
// API but reshaped around machines and parts.
//
//	/api/search?q=...          -> ranked machines + parts
//	/api/machine?id=N          -> a machine with its installed parts
//	/api/part?id=N             -> a single part
//	/api/inventory             -> loose (uninstalled) parts
func (h *Handlers) RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/search", h.apiSearch)
	mux.HandleFunc("/api/machine", h.apiMachine)
	mux.HandleFunc("/api/part", h.apiPart)
	mux.HandleFunc("/api/inventory", h.apiInventory)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func (h *Handlers) apiSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	count := atoiDefault(r.URL.Query().Get("count"), 100)
	res := h.store.Search(q)
	if len(res.Results) > count {
		res.Results = res.Results[:count]
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"query":   q,
		"count":   len(res.Results),
		"results": res.Results,
	})
}

func (h *Handlers) apiMachine(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.URL.Query().Get("id"), 0)
	m := h.store.FindMachine(id)
	if m == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"available": false})
		return
	}
	m.Parts = h.store.PartsForMachine(m.Id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"available": true, "machine": m})
}

func (h *Handlers) apiPart(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.URL.Query().Get("id"), 0)
	p := h.store.FindPart(id)
	if p == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"available": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"available": true, "part": p})
}

func (h *Handlers) apiInventory(w http.ResponseWriter, r *http.Request) {
	loose := h.store.PartsForMachine(0)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(loose),
		"parts": loose,
	})
}

package main

import (
	"encoding/json"
	"net/http"
)

func (h *Handlers) RegisterSearch(mux *http.ServeMux) {
	mux.HandleFunc("/search", h.searchPage)
	mux.HandleFunc("/api/live", h.searchLive) // used by search-as-you-type
}

func (h *Handlers) searchPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var result *SearchResult
	if q != "" {
		result = h.store.Search(q)
	} else {
		result = &SearchResult{}
	}
	h.tmpl.Render(w, http.StatusOK, "search.html", &page{
		Title:    "Search",
		Editable: h.canEdit(r),
		Active:   "search",
		Data:     result,
	})
}

// searchLive returns JSON for the type-ahead box. Kept small and flat so the
// front-end JS can render rows without a template round-trip.
type liveRow struct {
	Kind     string `json:"kind"`
	Id       int    `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Badge    string `json:"badge"`
	Href     string `json:"href"`
}

func (h *Handlers) searchLive(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	out := struct {
		Query string     `json:"query"`
		Rows  []*liveRow `json:"rows"`
	}{Query: q, Rows: []*liveRow{}}

	if q != "" {
		res := h.store.Search(q)
		for i, hit := range res.Results {
			if i >= 25 {
				break
			}
			if hit.Kind == "machine" {
				m := hit.Machine
				out.Rows = append(out.Rows, &liveRow{
					Kind: "machine", Id: m.Id,
					Title: m.Name, Subtitle: m.Type, Badge: m.Status,
					Href: "/machine?id=" + itoa(m.Id),
				})
			} else {
				p := hit.Part
				href := "/inventory"
				if p.MachineId != 0 {
					href = "/machine?id=" + itoa(p.MachineId)
				}
				out.Rows = append(out.Rows, &liveRow{
					Kind: "part", Id: p.Id,
					Title: p.Model, Subtitle: p.Category, Badge: p.Condition,
					Href: href,
				})
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

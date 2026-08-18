package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
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
// front-end JS can render rows without a template round-trip. Location is
// its own field (not folded into Subtitle) so the front end can lay it out
// as a fixed-width column that lines up the same way across every row.
type liveRow struct {
	Kind     string `json:"kind"`
	Id       int    `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Location string `json:"location"`
	Badge    string `json:"badge"`
	Href     string `json:"href"`
}

// recentLimit caps both the idle "recently added" list and the number of
// scored search results sent to the front end.
const recentLimit = 8

// formatLocation renders a Facility/Sub-location pair the same way
// everywhere, falling back to a placeholder when neither is set.
func formatLocation(facility, subLocation string) string {
	loc := strings.TrimSpace(facility)
	if subLocation != "" {
		if loc != "" {
			loc += ", "
		}
		loc += subLocation
	}
	if loc == "" {
		return "Unassigned"
	}
	return loc
}

// buildMachineRow renders a Machine as a live-search row.
func buildMachineRow(m *Machine) *liveRow {
	title := m.Asset
	if title == "" {
		title = m.Name
	}
	if title == "" {
		title = m.Type
	}

	subtitleParts := []string{}
	if m.Name != "" {
		subtitleParts = append(subtitleParts, m.Name)
	}
	if m.Type != "" {
		subtitleParts = append(subtitleParts, m.Type)
	}

	return &liveRow{
		Kind:     "machine",
		Id:       m.Id,
		Title:    title,
		Subtitle: strings.Join(subtitleParts, " · "),
		Location: formatLocation(m.Facility, m.SubLocation),
		Badge:    m.Status,
		Href:     "/machine?id=" + itoa(m.Id),
	}
}

// buildPartRow renders a Part as a live-search row. Subtitle is always just
// the category; Location says where to actually go find the part — either
// its own Facility/Sub-location if it's loose, or which machine it's
// installed in, so a search result tells you not just what a part is, but
// where to look for it.
func (h *Handlers) buildPartRow(p *Part) *liveRow {
	href := "/inventory"
	location := formatLocation(p.Facility, p.SubLocation)

	if p.MachineId != 0 {
		href = "/machine?id=" + itoa(p.MachineId)
		if m := h.store.FindMachine(p.MachineId); m != nil {
			label := strings.TrimSpace(m.Asset + " " + m.Name)
			if label == "" {
				label = m.Type
			}
			if label != "" {
				location = "in " + label
			}
		}
	}

	return &liveRow{
		Kind:     "part",
		Id:       p.Id,
		Title:    p.Model,
		Subtitle: p.Category,
		Location: location,
		Badge:    p.Condition,
		Href:     href,
	}
}

// recentRows returns the most recently created machines and parts, newest
// first, combined into a single list — what the search box shows before the
// user has typed anything.
func (h *Handlers) recentRows(limit int) []*liveRow {
	type entry struct {
		id  int
		row *liveRow
	}
	entries := make([]entry, 0, limit*2)
	for _, m := range h.store.AllMachines() {
		entries = append(entries, entry{id: m.Id, row: buildMachineRow(m)})
	}
	for _, p := range h.store.AllParts() {
		entries = append(entries, entry{id: p.Id, row: h.buildPartRow(p)})
	}
	// Ids are assigned in creation order within each table, so sorting each
	// kind by id descending and merging is equivalent to sorting by
	// created-time descending, without depending on timestamp precision.
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].id > entries[j].id })

	if len(entries) > limit {
		entries = entries[:limit]
	}
	out := make([]*liveRow, len(entries))
	for i, e := range entries {
		out[i] = e.row
	}
	return out
}

func (h *Handlers) searchLive(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	out := struct {
		Query  string     `json:"query"`
		Rows   []*liveRow `json:"rows"`
		Recent bool       `json:"recent"`
	}{Query: q, Rows: []*liveRow{}}

	if q == "" {
		out.Recent = true
		out.Rows = h.recentRows(recentLimit)
	} else {
		res := h.store.Search(q)
		for i, hit := range res.Results {
			if i >= 25 {
				break
			}
			if hit.Kind == "machine" {
				out.Rows = append(out.Rows, buildMachineRow(hit.Machine))
			} else {
				out.Rows = append(out.Rows, h.buildPartRow(hit.Part))
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

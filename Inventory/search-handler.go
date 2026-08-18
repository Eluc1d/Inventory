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

// searchPageData wraps a SearchResult (embedded, so existing template
// references like .Query/.Results keep working unchanged) with the extra
// state the scope dropdown needs.
type searchPageData struct {
	*SearchResult
	Scope          string
	CategoryScopes []string
}

// searchScopeOptions lists every distinct category actually in use across
// all parts (installed or loose — search covers both), ordered the same way
// the Inventory category filter orders them: canonical PartCategories order
// first, free-text categories alphabetically after.
func (h *Handlers) searchScopeOptions() []string {
	seen := map[string]bool{}
	for _, p := range h.store.AllParts() {
		c := p.Category
		if c == "" {
			c = "Uncategorized"
		}
		seen[c] = true
	}
	names := make([]string, 0, len(seen))
	for c := range seen {
		names = append(names, c)
	}
	sort.SliceStable(names, func(i, j int) bool {
		ri, rj := categoryRank(names[i]), categoryRank(names[j])
		if ri != rj {
			return ri < rj
		}
		return names[i] < names[j]
	})
	return names
}

// matchesScope narrows results to one type: "" (no filter), "machine", or a
// specific part category (with blank categories bucketed as "Uncategorized",
// matching the Inventory category filter's convention).
func matchesScope(kind, category, scope string) bool {
	if scope == "" {
		return true
	}
	if scope == "machine" {
		return kind == "machine"
	}
	if kind != "part" {
		return false
	}
	if category == "" {
		category = "Uncategorized"
	}
	return category == scope
}

func (h *Handlers) searchPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	scope := r.URL.Query().Get("scope")
	var result *SearchResult
	if q != "" {
		result = h.store.Search(q)
	} else {
		result = &SearchResult{}
	}
	if scope != "" && result.Results != nil {
		filtered := make([]*SearchHit, 0, len(result.Results))
		for _, hit := range result.Results {
			category := ""
			if hit.Part != nil {
				category = hit.Part.Category
			}
			if matchesScope(hit.Kind, category, scope) {
				filtered = append(filtered, hit)
			}
		}
		result.Results = filtered
	}
	h.tmpl.Render(w, http.StatusOK, "search.html", &page{
		Title:    "Search",
		Editable: h.canEdit(r),
		Active:   "search",
		Data: &searchPageData{
			SearchResult:   result,
			Scope:          scope,
			CategoryScopes: h.searchScopeOptions(),
		},
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
	title := m.Name
	if title == "" {
		title = m.Type
	}
	if title == "" {
		title = m.Asset
	}

	subtitleParts := []string{}
	if m.Asset != "" {
		subtitleParts = append(subtitleParts, m.Asset)
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

// recentRows returns the most recently created machines and parts matching
// scope (see matchesScope), newest first, combined into a single list —
// what the search box shows before the user has typed anything. Filtering
// happens before limit is applied, so narrowing to e.g. "RAM" doesn't get
// crowded out by unrelated recently-added machines.
func (h *Handlers) recentRows(limit int, scope string) []*liveRow {
	type entry struct {
		id  int
		row *liveRow
	}
	entries := make([]entry, 0, limit*2)
	for _, m := range h.store.AllMachines() {
		if matchesScope("machine", "", scope) {
			entries = append(entries, entry{id: m.Id, row: buildMachineRow(m)})
		}
	}
	for _, p := range h.store.AllParts() {
		if matchesScope("part", p.Category, scope) {
			entries = append(entries, entry{id: p.Id, row: h.buildPartRow(p)})
		}
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
	scope := r.URL.Query().Get("scope")
	out := struct {
		Query  string     `json:"query"`
		Rows   []*liveRow `json:"rows"`
		Recent bool       `json:"recent"`
	}{Query: q, Rows: []*liveRow{}}

	if q == "" {
		out.Recent = true
		out.Rows = h.recentRows(recentLimit, scope)
	} else {
		res := h.store.Search(q)
		count := 0
		for _, hit := range res.Results {
			if count >= 25 {
				break
			}
			category := ""
			if hit.Part != nil {
				category = hit.Part.Category
			}
			if !matchesScope(hit.Kind, category, scope) {
				continue
			}
			if hit.Kind == "machine" {
				out.Rows = append(out.Rows, buildMachineRow(hit.Machine))
			} else {
				out.Rows = append(out.Rows, h.buildPartRow(hit.Part))
			}
			count++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

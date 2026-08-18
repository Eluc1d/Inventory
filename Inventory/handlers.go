package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Handlers bundles the store, templates and edit-permission config.
type Handlers struct {
	store    Store
	tmpl     *Templates
	editNets []*net.IPNet
}

func NewHandlers(store Store, tmpl *Templates, editNets []*net.IPNet) *Handlers {
	return &Handlers{store: store, tmpl: tmpl, editNets: editNets}
}

// canEdit reports whether the request comes from an allowed editor network.
// With no networks configured, everyone may edit (single-user / trusted LAN).
func (h *Handlers) canEdit(r *http.Request) bool {
	if len(h.editNets) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range h.editNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func atoiDefault(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return v
	}
	return def
}

// page is the common envelope passed to every template.
type page struct {
	Title    string
	Editable bool
	Active   string // nav highlight: "machines" | "inventory" | "search"
	Flash    string
	Data     interface{}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.dashboard)
	mux.HandleFunc("/machine/group-edit", h.machineGroupEdit)
	mux.HandleFunc("/machine/group-edit/", h.machineGroupEdit)
	mux.HandleFunc("/machine", h.machineDetail)
	mux.HandleFunc("/machine/edit", h.machineEdit)
	mux.HandleFunc("/machine/delete", h.machineDelete)
	mux.HandleFunc("/machine/delete-to-inventory", h.machineDeleteToInventory)
	mux.HandleFunc("/machine/group-move-parts", h.machineGroupMoveParts)
	mux.HandleFunc("/inventory", h.inventory)
	mux.HandleFunc("/part/edit", h.partEdit)
	mux.HandleFunc("/part/delete", h.partDelete)
	mux.HandleFunc("/part/unassign", h.partUnassign)
	mux.HandleFunc("/part/pull", h.partPull)
	mux.HandleFunc("/export/machines", h.ExportMachinesHandler)
	mux.HandleFunc("/export/parts", h.ExportPartsHandler)
}

// ------------------------------------------------------------- dashboard

type dashboardData struct {
	Machines     []*Machine
	Counts       map[string]int // status -> count
	Statuses     []string
	LooseCount   int
	Total        int
	View         string // "bubble" | "rows"
	StatusFilter string
	Sort         string   // "" | "asset" | "name" | "type" | "status" | "condition" | "location"
	SortDir      string   // "asc" | "desc"
	ClientTag    string   // current client/group tag filter, "" = none
	ClientTags   []string // distinct tags in use, for the filter chip row
}

// distinctClientTags returns the sorted set of non-empty client/group tags
// currently in use, for the Workshop filter chips and the form's datalist of
// existing tags (so operators reuse "Smith Family Order" instead of retyping
// slight variations of it).
func distinctClientTags(machines []*Machine) []string {
	set := map[string]bool{}
	for _, m := range machines {
		if m.ClientTag != "" {
			set[m.ClientTag] = true
		}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// machineSortKeys maps a Workshop column-sort key to the string it sorts on.
// "location" sorts on Facility first, Sub-location second.
var machineSortKeys = map[string]func(*Machine) string{
	"asset":     func(m *Machine) string { return m.Asset },
	"name":      func(m *Machine) string { return m.Name },
	"type":      func(m *Machine) string { return m.Type },
	"status":    func(m *Machine) string { return m.Status },
	"condition": func(m *Machine) string { return m.Condition },
	"location":  func(m *Machine) string { return m.Facility + "\x00" + m.SubLocation },
}

func (h *Handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	machines := h.store.AllMachines()
	filter := r.URL.Query().Get("status")
	clientTag := r.URL.Query().Get("client_tag")
	view := r.URL.Query().Get("view")
	if view != "rows" {
		view = "bubble"
	}
	sortCol := r.URL.Query().Get("sort")
	sortDir := r.URL.Query().Get("dir")
	if sortDir != "desc" {
		sortDir = "asc"
	}

	counts := map[string]int{}
	for _, m := range machines {
		counts[m.Status]++
	}
	// Chip lists reflect the full unfiltered set so they don't shift around as
	// the other facet (status/client tag) narrows the view.
	clientTags := distinctClientTags(machines)

	shown := machines
	if filter != "" {
		shown = shown[:0]
		for _, m := range machines {
			if m.Status == filter {
				shown = append(shown, m)
			}
		}
	}
	if clientTag != "" {
		tagged := make([]*Machine, 0, len(shown))
		for _, m := range shown {
			if m.ClientTag == clientTag {
				tagged = append(tagged, m)
			}
		}
		shown = tagged
	}

	if keyFn, ok := machineSortKeys[sortCol]; ok {
		sort.SliceStable(shown, func(i, j int) bool {
			a, b := strings.ToLower(keyFn(shown[i])), strings.ToLower(keyFn(shown[j]))
			if sortDir == "desc" {
				a, b = b, a
			}
			return a < b
		})
	} else {
		sortCol = ""
		// Newest first on the board.
		sort.SliceStable(shown, func(i, j int) bool { return shown[i].Id > shown[j].Id })
	}

	flash := ""
	created := atoiDefault(r.URL.Query().Get("created"), 0)
	if created > 0 {
		if created == 1 {
			flash = "1 machine created successfully."
		} else {
			flash = strconv.Itoa(created) + " machines created successfully."
		}
	}

	data := &dashboardData{
		Machines:     shown,
		Counts:       counts,
		Statuses:     MachineStatuses,
		LooseCount:   len(h.store.PartsForMachine(0)),
		Total:        len(machines),
		View:         view,
		StatusFilter: filter,
		Sort:         sortCol,
		SortDir:      sortDir,
		ClientTag:    clientTag,
		ClientTags:   clientTags,
	}
	h.tmpl.Render(w, http.StatusOK, "machine-list.html", &page{
		Title:    "Workbench",
		Editable: h.canEdit(r),
		Active:   "machines",
		Flash:    flash,
		Data:     data,
	})
}

// ---------------------------------------------------------- machine detail

func (h *Handlers) machineDetail(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.URL.Query().Get("id"), 0)
	m := h.store.FindMachine(id)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	m.Parts = h.store.PartsForMachine(m.Id)
	h.tmpl.Render(w, http.StatusOK, "machine-detail.html", &page{
		Title:    m.Asset + " " + m.Name,
		Editable: h.canEdit(r),
		Active:   "machines",
		Data:     m,
	})
}

// ------------------------------------------------------------ machine edit

func (h *Handlers) machineEdit(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}

	idParam := r.URL.Query().Get("id")
	isNew := idParam == "" || idParam == "new"

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form submission.", http.StatusBadRequest)
			return
		}

		asset := strings.TrimSpace(r.FormValue("asset"))
		name := strings.TrimSpace(r.FormValue("name"))
		typ := strings.TrimSpace(r.FormValue("type"))
		status := strings.TrimSpace(r.FormValue("status"))
		condition := strings.TrimSpace(r.FormValue("condition"))
		facility := strings.TrimSpace(r.FormValue("facility"))
		subLocation := strings.TrimSpace(r.FormValue("sub_location"))
		notes := strings.TrimSpace(r.FormValue("notes"))
		clientTag := strings.TrimSpace(r.FormValue("client_tag"))
		stay := r.FormValue("stay") == "1"

		if isNew {
			m, errStr := h.store.CreateMachine(func(m *Machine) bool {
				m.Asset = asset
				m.Name = name
				m.Type = typ
				m.Status = status
				m.Condition = condition
				m.Facility = facility
				m.SubLocation = subLocation
				m.Notes = notes
				m.ClientTag = clientTag
				return true
			})
			if m == nil {
				http.Error(w, errStr, http.StatusBadRequest)
				return
			}

			h.applyStagedParts(r, m.Id)
			if stay {
				h.tmpl.Render(w, http.StatusOK, "machine-form.html", &page{
					Title:    "Log a machine",
					Editable: true,
					Active:   "machines",
					Data: machineFormData{
						Machine:                m,
						IsNew:                  true,
						FacilityOptions:        FacilityOptions,
						SubLocationsByFacility: SubLocationsByFacility,
						MachineStatuses:        MachineStatuses,
						ClientTags:             distinctClientTags(h.store.AllMachines()),
					},
				})
				return
			}

			http.Redirect(w, r, "/?created=1", http.StatusSeeOther)
			return
		}

		id := atoiDefault(idParam, 0)
		if ok, errStr := h.store.EditMachine(id, func(m *Machine) bool {
			m.Asset = asset
			m.Name = name
			m.Type = typ
			m.Status = status
			m.Condition = condition
			m.Facility = facility
			m.SubLocation = subLocation
			m.Notes = notes
			m.ClientTag = clientTag
			return true
		}); !ok {
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}

		h.applyStagedParts(r, id)
		http.Redirect(w, r, "/machine?id="+strconv.Itoa(id), http.StatusSeeOther)
		return
	}

	m := &Machine{}
	if !isNew {
		id := atoiDefault(idParam, 0)
		if found := h.store.FindMachine(id); found == nil {
			http.NotFound(w, r)
			return
		} else {
			m = found
		}
	}

	h.tmpl.Render(w, http.StatusOK, "machine-form.html", &page{
		Title:    "Edit machine",
		Editable: true,
		Active:   "machines",
		Data: machineFormData{
			Machine:                m,
			IsNew:                  isNew,
			FacilityOptions:        FacilityOptions,
			SubLocationsByFacility: SubLocationsByFacility,
			MachineStatuses:        MachineStatuses,
			ClientTags:             distinctClientTags(h.store.AllMachines()),
		},
	})
}

// ------------------------------------------------------------ machine edit

// stagedPartRow is a part staged in the create-machine form's "Parts" panel:
// either an existing loose-inventory part to reassign, or a new part to create.
type stagedPartRow struct {
	Source     string `json:"source"` // "existing" | "new"
	ExistingId int    `json:"existing_id"`
	Category   string `json:"category"`
	Model      string `json:"model"`
	Spec       string `json:"spec"`
	Quantity   int    `json:"quantity"`
	Condition  string `json:"condition"`
	Serial     string `json:"serial"`
	Notes      string `json:"notes"`
}

// applyStagedParts attaches the parts staged in the create-machine form to the
// newly created machine: existing loose parts are reassigned, new parts are
// created outright. Individual row failures are logged and skipped — the
// machine itself is already created and must not be lost over a bad row.
func (h *Handlers) applyStagedParts(r *http.Request, machineId int) {
	raw := r.FormValue("parts_json")
	if raw == "" {
		return
	}
	var rows []stagedPartRow
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		log.Printf("applyStagedParts: bad parts_json: %s", err)
		return
	}
	for _, row := range rows {
		switch row.Source {
		case "existing":
			if row.ExistingId == 0 {
				continue
			}
			h.pullPartOntoMachine(row.ExistingId, machineId, row.Quantity)
		case "new":
			category := strings.TrimSpace(row.Category)
			model := strings.TrimSpace(row.Model)
			spec := strings.TrimSpace(row.Spec)
			if category == "" && model == "" && spec == "" {
				continue
			}
			if _, errStr := h.store.CreatePart(func(p *Part) bool {
				p.MachineId = machineId
				p.Category = category
				p.Model = model
				p.Spec = spec
				p.Quantity = row.Quantity
				p.Condition = strings.TrimSpace(row.Condition)
				p.Serial = strings.TrimSpace(row.Serial)
				p.Notes = strings.TrimSpace(row.Notes)
				return true
			}); errStr != "" {
				log.Printf("applyStagedParts: create part: %s", errStr)
			}
		}
	}
}

// pullPartOntoMachine moves qty units of loose part sourceId onto machineId.
// If qty covers the whole line (or is missing/invalid), the part is reassigned
// outright; otherwise the source line is decremented and a new part row is
// created on the machine with the requested quantity, so the rest stays put
// in loose inventory. sourceId not found or machineId == 0 is a no-op.
func (h *Handlers) pullPartOntoMachine(sourceId, machineId, qty int) {
	src := h.store.FindPart(sourceId)
	if src == nil || machineId == 0 {
		return
	}
	if qty < 1 || qty > src.Quantity {
		qty = src.Quantity
	}
	if qty >= src.Quantity {
		h.store.EditPart(sourceId, func(p *Part) bool {
			p.MachineId = machineId
			return true
		})
		return
	}
	h.store.EditPart(sourceId, func(p *Part) bool {
		p.Quantity -= qty
		return true
	})
	h.store.CreatePart(func(p *Part) bool {
		p.MachineId = machineId
		p.Category = src.Category
		p.Model = src.Model
		p.Spec = src.Spec
		p.Quantity = qty
		p.Condition = src.Condition
		p.Serial = src.Serial
		p.Facility = src.Facility
		p.SubLocation = src.SubLocation
		p.Notes = src.Notes
		return true
	})
}

// partPull handles the "Add to machine" picker on part-form.html: pulling a
// specified quantity of a loose part onto a specific machine.
func (h *Handlers) partPull(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	id := atoiDefault(r.FormValue("id"), 0)
	machineId := atoiDefault(r.FormValue("machine_id"), 0)
	qty := atoiDefault(r.FormValue("quantity"), 0)
	h.pullPartOntoMachine(id, machineId, qty)
	if machineId != 0 {
		http.Redirect(w, r, "/machine?id="+strconv.Itoa(machineId), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

// ------------------------------------------------------------ machine group edit

func (h *Handlers) machineGroupEdit(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form submission.", http.StatusBadRequest)
			return
		}
		selectedIds := parseIntList(r.Form["selected_ids"])
		if len(selectedIds) == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		typ := strings.TrimSpace(r.FormValue("type"))
		status := strings.TrimSpace(r.FormValue("status"))
		condition := strings.TrimSpace(r.FormValue("condition"))
		facility := strings.TrimSpace(r.FormValue("facility"))
		subLocation := strings.TrimSpace(r.FormValue("sub_location"))
		notes := strings.TrimSpace(r.FormValue("notes"))
		clientTag := strings.TrimSpace(r.FormValue("client_tag"))

		for _, id := range selectedIds {
			if _, errStr := h.store.EditMachine(id, func(m *Machine) bool {
				if name != "" {
					m.Name = name
				}
				if typ != "" {
					m.Type = typ
				}
				if status != "" {
					m.Status = status
				}
				if condition != "" {
					m.Condition = condition
				}
				if facility != "" {
					m.Facility = facility
				}
				if subLocation != "" {
					m.SubLocation = subLocation
				}
				if notes != "" {
					m.Notes = notes
				}
				if clientTag != "" {
					m.ClientTag = clientTag
				}
				return true
			}); errStr != "" {
				log.Printf("machineGroupEdit: failed to update machine %d: %s", id, errStr)
			}
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	selectedIds := parseIntList(r.URL.Query()["ids"])
	if len(selectedIds) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	machines := make([]*Machine, 0, len(selectedIds))
	for _, id := range selectedIds {
		if m := h.store.FindMachine(id); m != nil {
			machines = append(machines, m)
		}
	}
	if len(machines) == 0 {
		http.NotFound(w, r)
		return
	}

	h.tmpl.Render(w, http.StatusOK, "machine-form.html", &page{
		Title:    "Group edit machines",
		Editable: true,
		Active:   "machines",
		Data: machineFormData{
			Machine:                machines[0],
			IsGroupEdit:            true,
			SelectedIds:            selectedIds,
			Shared:                 sharedMachineValues(machines),
			Machines:               machines,
			FacilityOptions:        FacilityOptions,
			SubLocationsByFacility: SubLocationsByFacility,
			MachineStatuses:        MachineStatuses,
			ClientTags:             distinctClientTags(h.store.AllMachines()),
		},
	})
}

// --------------------------------------------------------------- inventory

// facilityRank/subLocationRank order location groups the same way the
// facility/sub-location <select>s do, with unrecognized or blank values
// ("Unspecified") sorted last.
func facilityRank(f string) int {
	for i, opt := range FacilityOptions {
		if opt == f {
			return i
		}
	}
	return len(FacilityOptions)
}

func subLocationRank(facility, sub string) int {
	for i, opt := range SubLocationsByFacility[facility] {
		if opt == sub {
			return i
		}
	}
	return len(SubLocationsByFacility[facility])
}

// categoryRank orders categories the same way facilityRank orders facilities:
// by position in the canonical PartCategories list, with free-text categories
// not in that list sorted last (alphabetically among themselves).
func categoryRank(c string) int {
	for i, opt := range PartCategories {
		if opt == c {
			return i
		}
	}
	return len(PartCategories)
}

type subGroup struct {
	SubLocation string
	Parts       []*Part
	Units       int
}
type facGroup struct {
	Facility string
	Subs     []*subGroup
	Units    int
}
type catGroup struct {
	Category string
	Parts    []*Part
	Units    int
}
type categoryTag struct {
	Name   string
	Count  int
	Active bool
	Href   template.URL
}

type inventoryData struct {
	Groups         []*facGroup // used when GroupMode == "location" (default)
	CategoryGroups []*catGroup // used when GroupMode == "category"
	Total          int         // count after the category-tag filter
	GroupMode      string      // "location" | "category"
	Tags           []*categoryTag
	AnyTagActive   bool
	LocationHref   template.URL
	CategoryHref   template.URL
	ClearTagsHref  template.URL
}

// buildInventoryHref builds a safe, fully query-encoded /inventory URL for a
// given group mode and category selection, toggling toggleCat in/out of the
// selection when non-empty. Wrapped in template.URL so html/template treats
// it as an already-safe URL rather than re-escaping the "&"/"=" it
// necessarily contains — the arity here is unbounded (any number of selected
// tags), so the literal-inlining convention used for view/status/sort
// elsewhere in this app can't express it; this is Go's documented escape
// hatch for exactly that case.
func buildInventoryHref(group string, selected []string, toggleCat string) template.URL {
	newCats := make([]string, 0, len(selected)+1)
	found := false
	for _, c := range selected {
		if c == toggleCat {
			found = true
			continue
		}
		newCats = append(newCats, c)
	}
	if toggleCat != "" && !found {
		newCats = append(newCats, toggleCat)
	}
	v := url.Values{}
	if group == "category" {
		v.Set("group", "category")
	}
	for _, c := range newCats {
		v.Add("cat", c)
	}
	if len(v) == 0 {
		return "/inventory"
	}
	return template.URL("/inventory?" + v.Encode())
}

func (h *Handlers) inventory(w http.ResponseWriter, r *http.Request) {
	loose := h.store.PartsForMachine(0)

	groupMode := r.URL.Query().Get("group")
	if groupMode != "category" {
		groupMode = "location"
	}
	selectedCats := r.URL.Query()["cat"]
	selectedSet := map[string]bool{}
	for _, c := range selectedCats {
		selectedSet[c] = true
	}

	catKey := func(p *Part) string {
		if p.Category == "" {
			return "Uncategorized"
		}
		return p.Category
	}

	// Category tags + their counts, computed on the full unfiltered set so
	// the chips don't shift around as more tags get selected.
	catCounts := map[string]int{}
	for _, p := range loose {
		catCounts[catKey(p)]++
	}
	catNames := make([]string, 0, len(catCounts))
	for c := range catCounts {
		catNames = append(catNames, c)
	}
	sort.SliceStable(catNames, func(i, j int) bool {
		ri, rj := categoryRank(catNames[i]), categoryRank(catNames[j])
		if ri != rj {
			return ri < rj
		}
		return catNames[i] < catNames[j]
	})
	tags := make([]*categoryTag, 0, len(catNames))
	for _, c := range catNames {
		tags = append(tags, &categoryTag{
			Name:   c,
			Count:  catCounts[c],
			Active: selectedSet[c],
			Href:   buildInventoryHref(groupMode, selectedCats, c),
		})
	}

	// Narrow to the selected tags (OR across tags; no selection = no filter).
	filtered := loose
	if len(selectedSet) > 0 {
		filtered = make([]*Part, 0, len(loose))
		for _, p := range loose {
			if selectedSet[catKey(p)] {
				filtered = append(filtered, p)
			}
		}
	}

	var groups []*facGroup
	var catGroups []*catGroup

	if groupMode == "category" {
		byCat := map[string][]*Part{}
		for _, p := range filtered {
			byCat[catKey(p)] = append(byCat[catKey(p)], p)
		}
		names := make([]string, 0, len(byCat))
		for c := range byCat {
			names = append(names, c)
		}
		sort.SliceStable(names, func(i, j int) bool {
			ri, rj := categoryRank(names[i]), categoryRank(names[j])
			if ri != rj {
				return ri < rj
			}
			return names[i] < names[j]
		})
		for _, c := range names {
			units := 0
			for _, p := range byCat[c] {
				units += p.Quantity
			}
			catGroups = append(catGroups, &catGroup{Category: c, Parts: byCat[c], Units: units})
		}
	} else {
		// Group loose parts by Facility, then Sub-location, for a shelf-by-shelf view.
		byFacility := map[string]map[string][]*Part{}
		for _, p := range filtered {
			fac, sub := p.Facility, p.SubLocation
			if fac == "" {
				fac = "Unspecified"
			}
			if sub == "" {
				sub = "Unspecified"
			}
			if byFacility[fac] == nil {
				byFacility[fac] = map[string][]*Part{}
			}
			byFacility[fac][sub] = append(byFacility[fac][sub], p)
		}

		facs := make([]string, 0, len(byFacility))
		for f := range byFacility {
			facs = append(facs, f)
		}
		sort.SliceStable(facs, func(i, j int) bool {
			ri, rj := facilityRank(facs[i]), facilityRank(facs[j])
			if ri != rj {
				return ri < rj
			}
			return facs[i] < facs[j]
		})

		for _, f := range facs {
			subs := make([]string, 0, len(byFacility[f]))
			for s := range byFacility[f] {
				subs = append(subs, s)
			}
			sort.SliceStable(subs, func(i, j int) bool {
				ri, rj := subLocationRank(f, subs[i]), subLocationRank(f, subs[j])
				if ri != rj {
					return ri < rj
				}
				return subs[i] < subs[j]
			})

			fg := &facGroup{Facility: f}
			for _, s := range subs {
				units := 0
				for _, p := range byFacility[f][s] {
					units += p.Quantity
				}
				fg.Subs = append(fg.Subs, &subGroup{SubLocation: s, Parts: byFacility[f][s], Units: units})
				fg.Units += units
			}
			groups = append(groups, fg)
		}
	}

	h.tmpl.Render(w, http.StatusOK, "inventory.html", &page{
		Title:    "Parts inventory",
		Editable: h.canEdit(r),
		Active:   "inventory",
		Data: &inventoryData{
			Groups:         groups,
			CategoryGroups: catGroups,
			Total:          len(filtered),
			GroupMode:      groupMode,
			Tags:           tags,
			AnyTagActive:   len(selectedSet) > 0,
			LocationHref:   buildInventoryHref("location", selectedCats, ""),
			CategoryHref:   buildInventoryHref("category", selectedCats, ""),
			ClearTagsHref:  buildInventoryHref(groupMode, nil, ""),
		},
	})
}

// --------------------------------------------------------------- part edit

type partFormData struct {
	Part                   *Part
	IsNew                  bool
	Machine                *Machine
	Machines               []*Machine
	LooseParts             []*Part
	FacilityOptions        []string
	SubLocationsByFacility map[string][]string
	CreatedPartLabel       string
}

func (h *Handlers) partEdit(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	idParam := r.URL.Query().Get("id")
	isNew := idParam == "" || idParam == "new"

	if r.Method == http.MethodPost {
		stay := r.FormValue("stay") == "1"
		apply := func(p *Part) bool {
			p.MachineId = atoiDefault(r.FormValue("machine_id"), 0)
			p.Category = strings.TrimSpace(r.FormValue("category"))
			p.Model = strings.TrimSpace(r.FormValue("model"))
			p.Spec = strings.TrimSpace(r.FormValue("spec"))
			p.Quantity = atoiDefault(r.FormValue("quantity"), 1)
			p.Condition = strings.TrimSpace(r.FormValue("condition"))
			p.Serial = strings.TrimSpace(r.FormValue("serial"))
			p.Facility = strings.TrimSpace(r.FormValue("facility"))
			p.SubLocation = strings.TrimSpace(r.FormValue("sub_location"))
			p.Notes = strings.TrimSpace(r.FormValue("notes"))
			return true
		}

		var machineId int
		if isNew {
			p, errStr := h.store.CreatePart(apply)
			if p == nil {
				http.Error(w, errStr, http.StatusBadRequest)
				return
			}
			machineId = p.MachineId

			if stay {
				var machine *Machine
				if p.MachineId != 0 {
					machine = h.store.FindMachine(p.MachineId)
				}

				createdLabel := strings.TrimSpace(p.Category + " " + p.Model)
				if createdLabel == "" {
					createdLabel = p.Serial
				}

				h.tmpl.Render(w, http.StatusOK, "part-form.html", &page{
					Title:    "Add part",
					Editable: true,
					Active:   "inventory",
					Data: partFormData{
						Part:                   p,
						IsNew:                  true,
						Machine:                machine,
						Machines:               h.store.AllMachines(),
						LooseParts:             nil,
						FacilityOptions:        FacilityOptions,
						SubLocationsByFacility: SubLocationsByFacility,
						CreatedPartLabel:       createdLabel,
					},
				})
				return
			}
		} else {
			id := atoiDefault(idParam, 0)
			h.store.EditPart(id, apply)
			if p := h.store.FindPart(id); p != nil {
				machineId = p.MachineId
			}
		}
		redirectAfterPart(w, r, machineId)
		return
	}

	// GET: render the form.
	p := &Part{Quantity: 1, Condition: "Untested"}
	if !isNew {
		if found := h.store.FindPart(atoiDefault(idParam, 0)); found != nil {
			p = found
		}
	} else {
		// New part may be pre-assigned to a machine via ?machine=N.
		p.MachineId = atoiDefault(r.URL.Query().Get("machine"), 0)
	}

	var machine *Machine
	if p.MachineId != 0 {
		machine = h.store.FindMachine(p.MachineId)
	}
	// Offer the "pull from inventory" picker only when adding a brand-new part
	// to a specific machine — pulling a loose part into loose inventory is a
	// no-op, and it doesn't apply when editing an already-attached part.
	var looseParts []*Part
	if isNew && machine != nil {
		looseParts = h.store.PartsForMachine(0)
	}
	h.tmpl.Render(w, http.StatusOK, "part-form.html", &page{
		Title:    "Edit part",
		Editable: true,
		Active:   "inventory",
		Data: partFormData{
			Part:                   p,
			IsNew:                  isNew,
			Machine:                machine,
			Machines:               h.store.AllMachines(),
			LooseParts:             looseParts,
			FacilityOptions:        FacilityOptions,
			SubLocationsByFacility: SubLocationsByFacility,
		},
	})
}

func (h *Handlers) partDelete(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	id := atoiDefault(r.FormValue("id"), 0)
	machineId := 0
	if p := h.store.FindPart(id); p != nil {
		machineId = p.MachineId
	}
	h.store.DeletePart(id)
	redirectAfterPart(w, r, machineId)
}

// partUnassign detaches a part from its machine and returns it to loose
// inventory, rather than deleting it outright. This is what "Remove" on a
// machine's installed-parts list does — the part still exists, it's just no
// longer installed anywhere.
func (h *Handlers) partUnassign(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	id := atoiDefault(r.FormValue("id"), 0)
	facility := strings.TrimSpace(r.FormValue("facility"))
	subLocation := strings.TrimSpace(r.FormValue("sub_location"))
	src := h.store.FindPart(id)
	if src == nil {
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)
		return
	}
	machineId := src.MachineId

	// If an identical line already sits in loose inventory, fold this part's
	// quantity into it instead of leaving a duplicate row behind — e.g.
	// returning 1 desk shouldn't create a second "1x desk" next to an
	// existing "13x desks" line. The existing line's own location wins; the
	// facility/sub_location picked on the way back only matters when this
	// part ends up staying on its own (no match to merge into).
	if target := h.findMatchingLoosePart(src); target != nil {
		h.store.EditPart(target.Id, func(p *Part) bool {
			p.Quantity += src.Quantity
			return true
		})
		h.store.DeletePart(id)
	} else {
		h.store.EditPart(id, func(p *Part) bool {
			p.MachineId = 0
			p.Facility = facility
			p.SubLocation = subLocation
			return true
		})
	}

	// Stay on the machine the part was just removed from, rather than jumping
	// to the inventory page — the machine view already reflects the change.
	if machineId != 0 {
		http.Redirect(w, r, "/machine?id="+strconv.Itoa(machineId), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

// findMatchingLoosePart looks for a loose-inventory part that's identical to
// src in everything but quantity, machine assignment, and location — the
// signal that it's the "same" line, so a returned part should merge into it
// rather than sit as a separate duplicate entry.
func (h *Handlers) findMatchingLoosePart(src *Part) *Part {
	for _, p := range h.store.PartsForMachine(0) {
		if p.Id == src.Id {
			continue
		}
		if p.Category == src.Category && p.Model == src.Model && p.Spec == src.Spec &&
			p.Condition == src.Condition && p.Serial == src.Serial {
			return p
		}
	}
	return nil
}

func (h *Handlers) machineDelete(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	id := atoiDefault(r.FormValue("id"), 0)
	if ok, _ := h.store.DeleteMachine(id); !ok {
		log.Printf("machineDelete: failed to delete machine %d", id)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// machineDeleteToInventory moves all parts from the machine to loose inventory
// and then deletes the machine.
func (h *Handlers) machineDeleteToInventory(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	id := atoiDefault(r.FormValue("id"), 0)
	// Move each installed part to loose inventory (machine_id = 0).
	parts := h.store.PartsForMachine(id)
	for _, p := range parts {
		// best-effort; log failures but continue
		if _, errStr := h.store.EditPart(p.Id, func(pp *Part) bool {
			pp.MachineId = 0
			return true
		}); errStr != "" {
			log.Printf("machineDeleteToInventory: failed to move part %d: %s", p.Id, errStr)
		}
	}
	// Now delete the machine (no parts should remain).
	if ok, _ := h.store.DeleteMachine(id); !ok {
		log.Printf("machineDeleteToInventory: failed to delete machine %d", id)
	}
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

func redirectAfterPart(w http.ResponseWriter, r *http.Request, machineId int) {
	if machineId != 0 {
		http.Redirect(w, r, "/machine?id="+strconv.Itoa(machineId), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)
	}
}

type groupEditShared struct {
	Asset             string
	AssetVaried       bool
	Name              string
	NameVaried        bool
	Type              string
	TypeVaried        bool
	Status            string
	StatusVaried      bool
	Condition         string
	ConditionVaried   bool
	Facility          string
	FacilityVaried    bool
	SubLocation       string
	SubLocationVaried bool
	Notes             string
	NotesVaried       bool
	ClientTag         string
	ClientTagVaried   bool
}

type machineFormData struct {
	Machine                *Machine
	IsNew                  bool
	IsGroupEdit            bool
	SelectedIds            []int
	Shared                 groupEditShared
	Machines               []*Machine
	FacilityOptions        []string
	SubLocationsByFacility map[string][]string
	MachineStatuses        []string
	ClientTags             []string // distinct client/group tags in use, for the datalist
	SuccessMessage         string
	CreatedAsset           string
	CreatedName            string
	Quantity               int
}

func parseIntList(values []string) []int {
	out := make([]int, 0, len(values))
	seen := make(map[int]bool)
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		id, err := strconv.Atoi(raw)
		if err != nil {
			continue
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func sharedMachineValues(machines []*Machine) groupEditShared {
	shared := groupEditShared{}
	if len(machines) == 0 {
		return shared
	}

	first := machines[0]
	shared.Asset = first.Asset
	shared.Name = first.Name
	shared.Type = first.Type
	shared.Status = first.Status
	shared.Condition = first.Condition
	shared.Facility = first.Facility
	shared.SubLocation = first.SubLocation
	shared.Notes = first.Notes
	shared.ClientTag = first.ClientTag

	for _, m := range machines[1:] {
		if m.Asset != shared.Asset {
			shared.AssetVaried = true
			shared.Asset = ""
		}
		if m.Name != shared.Name {
			shared.NameVaried = true
			shared.Name = ""
		}
		if m.Type != shared.Type {
			shared.TypeVaried = true
			shared.Type = ""
		}
		if m.Status != shared.Status {
			shared.StatusVaried = true
			shared.Status = ""
		}
		if m.Condition != shared.Condition {
			shared.ConditionVaried = true
			shared.Condition = ""
		}
		if m.Facility != shared.Facility {
			shared.FacilityVaried = true
			shared.Facility = ""
		}
		if m.SubLocation != shared.SubLocation {
			shared.SubLocationVaried = true
			shared.SubLocation = ""
		}
		if m.Notes != shared.Notes {
			shared.NotesVaried = true
			shared.Notes = ""
		}
		if m.ClientTag != shared.ClientTag {
			shared.ClientTagVaried = true
			shared.ClientTag = ""
		}
	}

	return shared
}

func (h *Handlers) machineGroupMoveParts(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission.", http.StatusBadRequest)
		return
	}

	selectedIds := parseIntList(r.Form["selected_ids"])
	if len(selectedIds) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	for _, id := range selectedIds {
		parts := h.store.PartsForMachine(id)
		for _, p := range parts {
			if _, errStr := h.store.EditPart(p.Id, func(pp *Part) bool {
				pp.MachineId = 0
				return true
			}); errStr != "" {
				log.Printf("machineGroupMoveParts: failed to move part %d from machine %d: %s", p.Id, id, errStr)
			}
		}
	}

	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

// ExportMachinesHandler exports all machines to TSV format
func (h *Handlers) ExportMachinesHandler(w http.ResponseWriter, r *http.Request) {
	machines := h.store.AllMachines()

	w.Header().Set("Content-Type", "text/tab-separated-values; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=machines.tsv")

	// Write BOM for Google Docs compatibility
	w.Write([]byte("\uFEFF"))

	// Write header
	fmt.Fprintln(w, "Asset\tName\tType\tStatus\tCondition\tFacility\tSub-Location\tNotes")

	// Write data rows
	for _, machine := range machines {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			machine.Asset,
			machine.Name,
			machine.Type,
			machine.Status,
			machine.Condition,
			machine.Facility,
			machine.SubLocation,
			machine.Notes,
		)
	}
}

// ExportPartsHandler exports all parts to TSV format
func (h *Handlers) ExportPartsHandler(w http.ResponseWriter, r *http.Request) {
	parts := h.store.PartsForMachine(0) // Get only loose parts for inventory export

	w.Header().Set("Content-Type", "text/tab-separated-values; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=parts.tsv")

	// Write BOM for Google Docs compatibility
	w.Write([]byte("\uFEFF"))

	// Write header
	fmt.Fprintln(w, "Category\tModel\tSpec\tQuantity\tCondition\tSerial\tFacility\tSub-Location\tNotes")

	// Write data rows
	for _, part := range parts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
			part.Category,
			part.Model,
			part.Spec,
			part.Quantity,
			part.Condition,
			part.Serial,
			part.Facility,
			part.SubLocation,
			part.Notes,
		)
	}
}

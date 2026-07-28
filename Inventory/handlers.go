package main

import (
	"log"
	"net"
	"net/http"
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
	mux.HandleFunc("/machine", h.machineDetail)
	mux.HandleFunc("/machine/edit", h.machineEdit)
	mux.HandleFunc("/inventory", h.inventory)
	mux.HandleFunc("/part/edit", h.partEdit)
	mux.HandleFunc("/part/delete", h.partDelete)
}

// ------------------------------------------------------------- dashboard

type dashboardData struct {
	Machines   []*Machine
	Counts     map[string]int // status -> count
	Statuses   []string
	LooseCount int
	Total      int
}

func (h *Handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	machines := h.store.AllMachines()
	filter := r.URL.Query().Get("status")

	counts := map[string]int{}
	for _, m := range machines {
		counts[m.Status]++
	}

	shown := machines
	if filter != "" {
		shown = shown[:0]
		for _, m := range machines {
			if m.Status == filter {
				shown = append(shown, m)
			}
		}
	}
	// Newest first on the board.
	sort.SliceStable(shown, func(i, j int) bool { return shown[i].Id > shown[j].Id })

	data := &dashboardData{
		Machines:   shown,
		Counts:     counts,
		Statuses:   MachineStatuses,
		LooseCount: len(h.store.PartsForMachine(0)),
		Total:      len(machines),
	}
	h.tmpl.Render(w, http.StatusOK, "machine-list.html", &page{
		Title:    "Workbench",
		Editable: h.canEdit(r),
		Active:   "machines",
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
		apply := func(m *Machine) bool {
			m.Name = strings.TrimSpace(r.FormValue("name"))
			m.Type = strings.TrimSpace(r.FormValue("type"))
			m.Status = strings.TrimSpace(r.FormValue("status"))
			m.Condition = strings.TrimSpace(r.FormValue("condition"))
			m.Location = strings.TrimSpace(r.FormValue("location"))
			m.Notes = strings.TrimSpace(r.FormValue("notes"))
			return true
		}
		if isNew {
			m, errStr := h.store.CreateMachine(apply)
			if m == nil {
				http.Error(w, errStr, http.StatusBadRequest)
				return
			}
			http.Redirect(w, r, "/machine?id="+strconv.Itoa(m.Id), http.StatusSeeOther)
			return
		}
		id := atoiDefault(idParam, 0)
		if _, errStr := h.store.EditMachine(id, apply); errStr != "" && errStr != "no change" {
			log.Printf("machineEdit: %s", errStr)
		}
		http.Redirect(w, r, "/machine?id="+strconv.Itoa(id), http.StatusSeeOther)
		return
	}

	// GET: render the form (blank for new).
	m := &Machine{Status: MachineStatuses[0]}
	if !isNew {
		if found := h.store.FindMachine(atoiDefault(idParam, 0)); found != nil {
			m = found
		}
	}
	h.tmpl.Render(w, http.StatusOK, "machine-form.html", &page{
		Title:    "Edit machine",
		Editable: true,
		Active:   "machines",
		Data: struct {
			Machine *Machine
			IsNew   bool
		}{m, isNew},
	})
}

// --------------------------------------------------------------- inventory

func (h *Handlers) inventory(w http.ResponseWriter, r *http.Request) {
	loose := h.store.PartsForMachine(0)
	// Group loose parts by category for a tidy shelf view.
	byCat := map[string][]*Part{}
	for _, p := range loose {
		byCat[p.Category] = append(byCat[p.Category], p)
	}
	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	type catGroup struct {
		Category string
		Parts    []*Part
		Units    int
	}
	groups := make([]*catGroup, 0, len(cats))
	for _, c := range cats {
		units := 0
		for _, p := range byCat[c] {
			units += p.Quantity
		}
		groups = append(groups, &catGroup{Category: c, Parts: byCat[c], Units: units})
	}

	h.tmpl.Render(w, http.StatusOK, "inventory.html", &page{
		Title:    "Parts inventory",
		Editable: h.canEdit(r),
		Active:   "inventory",
		Data: struct {
			Groups []*catGroup
			Total  int
		}{groups, len(loose)},
	})
}

// --------------------------------------------------------------- part edit

func (h *Handlers) partEdit(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	idParam := r.URL.Query().Get("id")
	isNew := idParam == "" || idParam == "new"

	if r.Method == http.MethodPost {
		apply := func(p *Part) bool {
			p.MachineId = atoiDefault(r.FormValue("machine_id"), 0)
			p.Category = strings.TrimSpace(r.FormValue("category"))
			p.Model = strings.TrimSpace(r.FormValue("model"))
			p.Spec = strings.TrimSpace(r.FormValue("spec"))
			p.Quantity = atoiDefault(r.FormValue("quantity"), 1)
			p.Condition = strings.TrimSpace(r.FormValue("condition"))
			p.Serial = strings.TrimSpace(r.FormValue("serial"))
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
	h.tmpl.Render(w, http.StatusOK, "part-form.html", &page{
		Title:    "Edit part",
		Editable: true,
		Active:   "inventory",
		Data: struct {
			Part     *Part
			IsNew    bool
			Machine  *Machine
			Machines []*Machine
		}{p, isNew, machine, h.store.AllMachines()},
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

func redirectAfterPart(w http.ResponseWriter, r *http.Request, machineId int) {
	if machineId != 0 {
		http.Redirect(w, r, "/machine?id="+strconv.Itoa(machineId), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)
	}
}

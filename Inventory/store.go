package main

import "time"

// Machine is a computer that has come through TechToss. It moves through a
// workflow (see MachineStatuses) and contains Parts.
type Machine struct {
	Id          int       `json:"id"`
	Asset       string    `json:"asset"` // human-facing asset tag, e.g. "TT-0007"
	Name        string    `json:"name"`  // short label, e.g. "Dell OptiPlex 7060"
	Type        string    `json:"type"`  // Desktop, Laptop, Server, ...
	Status      string    `json:"status"`
	Condition   string    `json:"condition,omitempty"`
	Facility    string    `json:"facility"`
	SubLocation string    `json:"sub_location"`
	Notes       string    `json:"notes,omitempty"`
	Created     time.Time `json:"created,omitempty"`
	Updated     time.Time `json:"updated,omitempty"`

	// Populated on demand by handlers; not stored on the machine row.
	Parts []*Part `json:"parts,omitempty"`
}

// Part is a component: RAM, GPU, CPU, storage, PSU, motherboard, etc. A part is
// either installed in a Machine (MachineId != 0) or sits in loose inventory
// (MachineId == 0).
type Part struct {
	Id        int    `json:"id"`
	MachineId int    `json:"machine_id,omitempty"` // 0 == loose inventory
	Category  string `json:"category"`             // RAM, GPU, CPU, Storage, ...
	Model     string `json:"model"`                // "Corsair Vengeance 16GB DDR4-3200"
	Spec      string `json:"spec,omitempty"`       // free-form specs / description
	Quantity  int    `json:"quantity"`
	Condition string `json:"condition,omitempty"` // Working, Untested, Faulty
	Serial    string `json:"serial,omitempty"`
	// Facility/SubLocation only apply while a part sits in loose inventory —
	// once installed in a machine, the part's location is the machine's.
	Facility    string    `json:"facility,omitempty"`
	SubLocation string    `json:"sub_location,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Created     time.Time `json:"created,omitempty"`
	Updated     time.Time `json:"updated,omitempty"`
}

var FacilityOptions = []string{
	"Techtoss",
	"Bruce's Storage",
	"Connor's Garage",
	"Almond Orchard",
	"Other",
}

var SubLocationsByFacility = map[string][]string{
	"Techtoss":        {"Rack A", "Rack B", "Shelf 1"},
	"Bruce's Storage": {"Floor"},
	"Connor's Garage": {"Floor"},
	"Almond Orchard":  {"Floor"},
	"Other":           {"N/A"},
}

// The canonical workflow a machine moves through. Order matters: it is used to
// render the status picker and to colour-code chips.
var MachineStatuses = []string{
	"Intake",
	"Testing",
	"Repair",
	"Refurbished",
	"For Parts",
	"Sold",
	"Tossed",
}

// Common component categories, offered as suggestions in the part form.
var PartCategories = []string{
	"CPU",
	"GPU",
	"RAM",
	"Storage",
	"Motherboard",
	"PSU",
	"Cooling",
	"Case",
	"Cabling",
	"Network",
	"Optical",
	"Peripheral",
	"Other",
}

var PartConditions = []string{"Working", "Untested", "Faulty"}

// ModifyMachine mutates a machine in place; returning true commits the change.
type ModifyMachine func(m *Machine) bool

// ModifyPart mutates a part in place; returning true commits the change.
type ModifyPart func(p *Part) bool

// SearchResult is a flat, ranked mix of machines and parts matching a query.
type SearchResult struct {
	Query   string
	Results []*SearchHit
}

// SearchHit is one row in a search result: either a machine or a part.
type SearchHit struct {
	Kind    string   `json:"kind"` // "machine" or "part"
	Machine *Machine `json:"machine,omitempty"`
	Part    *Part    `json:"part,omitempty"`
	Score   float32  `json:"-"`
}

// Store is the persistence boundary. The web/API handlers only ever talk to
// this interface, so the SQLite backend can be swapped without touching them.
type Store interface {
	// --- Machines ---
	FindMachine(id int) *Machine
	EditMachine(id int, updater ModifyMachine) (bool, string)
	CreateMachine(updater ModifyMachine) (*Machine, string)
	AllMachines() []*Machine
	// DeleteMachine removes a machine and its installed parts. It returns the
	// number of parts that were removed along with it.
	DeleteMachine(id int) (ok bool, partsRemoved int)

	// --- Parts ---
	FindPart(id int) *Part
	EditPart(id int, updater ModifyPart) (bool, string)
	CreatePart(updater ModifyPart) (*Part, string)
	DeletePart(id int) bool
	// PartsForMachine returns the parts installed in a machine (machineId != 0),
	// or the loose-inventory parts when machineId == 0.
	PartsForMachine(machineId int) []*Part
	AllParts() []*Part

	// --- Search ---
	Search(term string) *SearchResult
}

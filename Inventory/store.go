package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

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
	ClientTag   string    `json:"client_tag,omitempty"` // free-text label grouping machines built for the same client/order
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

// Ticket is a client's order/request — what they want (Items) and any
// documents/photos they sent (Attachments), tracked separately from the
// Machines/Parts that may eventually fulfill it.
type Ticket struct {
	Id       int       `json:"id"`
	Number   string    `json:"number"` // auto-generated, e.g. "TK-0007"
	Title    string    `json:"title"`
	Client   string    `json:"client"`
	Deadline string    `json:"deadline,omitempty"` // "YYYY-MM-DD", optional
	Status   string    `json:"status"`
	Notes    string    `json:"notes,omitempty"`
	Created  time.Time `json:"created,omitempty"`
	Updated  time.Time `json:"updated,omitempty"`

	// Populated on demand by handlers; not stored on the ticket row.
	Items       []*TicketItem       `json:"items,omitempty"`
	Attachments []*TicketAttachment `json:"attachments,omitempty"`
}

// TicketItem is a single line of what the client wants — a free-text
// description tagged by kind. It does not link to a real Machine/Part row: a
// ticket is a request, not a reservation against existing stock.
type TicketItem struct {
	Id          int       `json:"id"`
	TicketId    int       `json:"ticket_id"`
	Kind        string    `json:"kind"` // Machine, Part, or Custom
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	Created     time.Time `json:"created,omitempty"`
}

// TicketAttachment is a document/image the client sent, stored on disk under
// a random filename (never the client's original name — see attachment
// upload handling for why).
type TicketAttachment struct {
	Id           int       `json:"id"`
	TicketId     int       `json:"ticket_id"`
	Filename     string    `json:"filename"`      // on-disk name under uploadsDir
	OriginalName string    `json:"original_name"` // client's filename, display-only
	ContentType  string    `json:"content_type,omitempty"`
	Size         int64     `json:"size"`
	Created      time.Time `json:"created,omitempty"`
}

// IsImage reports whether the attachment's extension is one of the image
// types this app accepts, for choosing an inline thumbnail vs. a document
// icon in ticket-detail.html.
func (a *TicketAttachment) IsImage() bool {
	switch strings.ToLower(filepath.Ext(a.Filename)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	}
	return false
}

// SizeLabel formats the attachment's size for display, e.g. "245 KB".
func (a *TicketAttachment) SizeLabel() string {
	const kb = 1024
	const mb = kb * 1024
	switch {
	case a.Size >= mb:
		return fmt.Sprintf("%.1f MB", float64(a.Size)/float64(mb))
	case a.Size >= kb:
		return fmt.Sprintf("%.0f KB", float64(a.Size)/float64(kb))
	default:
		return fmt.Sprintf("%d B", a.Size)
	}
}

// The workflow a ticket moves through.
var TicketStatuses = []string{
	"Open",
	"In Progress",
	"Fulfilled",
	"Cancelled",
}

// The three kinds of thing a ticket item can request.
var TicketItemKinds = []string{"Machine", "Part", "Custom"}

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

// ModifyTicket mutates a ticket in place; returning true commits the change.
type ModifyTicket func(t *Ticket) bool

// ModifyTicketAttachment mutates an attachment record in place; returning
// true commits the change.
type ModifyTicketAttachment func(a *TicketAttachment) bool

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

	// --- Tickets ---
	FindTicket(id int) *Ticket
	EditTicket(id int, updater ModifyTicket) (bool, string)
	CreateTicket(updater ModifyTicket) (*Ticket, string)
	AllTickets() []*Ticket
	// DeleteTicket removes a ticket, its items, and its attachments (including
	// the on-disk files); callers still need to know which files were removed
	// to report/verify cleanup, so it returns them.
	DeleteTicket(id int) (ok bool, removedAttachments []*TicketAttachment)

	ItemsForTicket(ticketId int) []*TicketItem
	// ReplaceTicketItems atomically replaces every item on a ticket with the
	// given set, so a mid-operation failure can't leave the ticket with none.
	ReplaceTicketItems(ticketId int, items []*TicketItem) (bool, string)

	AttachmentsForTicket(ticketId int) []*TicketAttachment
	FindAttachment(id int) *TicketAttachment
	CreateAttachment(updater ModifyTicketAttachment) (*TicketAttachment, string)
	// DeleteAttachment removes the DB row only; callers are responsible for
	// removing the on-disk file themselves (the store doesn't know uploadsDir).
	DeleteAttachment(id int) bool

	// --- Search ---
	Search(term string) *SearchResult
}

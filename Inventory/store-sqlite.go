package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

var createSchema = `
create table if not exists machine (
    id        integer primary key autoincrement,
    asset     varchar(20),
    name      varchar(120),
    type      varchar(40),
    status    varchar(20),
    condition varchar(40),
    facility  varchar(60),
		sub_location varchar(60),
    notes     text,
    created   timestamp,
    updated   timestamp
);

create table if not exists part (
    id         integer primary key autoincrement,
    machine_id integer,               -- null => loose inventory
    category   varchar(40),
    model      varchar(160),
    spec       text,
    quantity   integer default 1,
    condition  varchar(20),
    serial     varchar(80),
    notes      text,
    created    timestamp,
    updated    timestamp,
    foreign key(machine_id) references machine(id)
);

create index if not exists idx_part_machine on part(machine_id);
`

func str(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func fromNull(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type SqlStore struct {
	db  *sql.DB
	fts *Search
}

func NewSqlStore(db *sql.DB) (*SqlStore, error) {
	if _, err := db.Exec(createSchema); err != nil {
		return nil, err
	}

	if err := ensureMachineColumns(db); err != nil {
		return nil, err
	}
	if err := ensurePartColumns(db); err != nil {
		return nil, err
	}
	if err := renameFacility(db, "Workshop", "Techtoss"); err != nil {
		return nil, err
	}

	// SQLite ignores FK constraints unless asked.
	db.Exec("PRAGMA foreign_keys = ON")

	s := &SqlStore{db: db, fts: NewSearch()}

	// Warm the in-memory search index from what's already on disk.
	nm, np := 0, 0
	for _, m := range s.AllMachines() {
		s.fts.UpdateMachine(m)
		nm++
	}
	for _, p := range s.AllParts() {
		s.fts.UpdatePart(p)
		np++
	}
	log.Printf("Search index warmed: %d machines, %d parts", nm, np)
	return s, nil
}

func ensureMachineColumns(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE machine ADD COLUMN facility varchar(60)`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	_, err = db.Exec(`ALTER TABLE machine ADD COLUMN sub_location varchar(60)`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	_, err = db.Exec(`ALTER TABLE machine ADD COLUMN client_tag varchar(120)`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	return nil
}

// renameFacility is a one-time, idempotent fixup for facility taxonomy
// renames: any machine/part still tagged with the old name is moved to the
// new one, so the rename doesn't orphan existing rows outside FacilityOptions.
func renameFacility(db *sql.DB, from, to string) error {
	if _, err := db.Exec("UPDATE machine SET facility=? WHERE facility=?", to, from); err != nil {
		return err
	}
	if _, err := db.Exec("UPDATE part SET facility=? WHERE facility=?", to, from); err != nil {
		return err
	}
	return nil
}

func ensurePartColumns(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE part ADD COLUMN facility varchar(60)`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	_, err = db.Exec(`ALTER TABLE part ADD COLUMN sub_location varchar(60)`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	return nil
}

// ---------------------------------------------------------------- machines

func scanMachine(rows *sql.Rows) (*Machine, error) {
	var (
		id                                                             int
		asset, name, typ, status, cond, fac, sub_loc, notes, clientTag *string
		created, updated                                               *time.Time
	)
	err := rows.Scan(&id, &asset, &name, &typ, &status, &cond, &fac, &sub_loc, &notes, &clientTag, &created, &updated)
	if err != nil {
		return nil, err
	}
	m := &Machine{
		Id:          id,
		Asset:       fromNull(asset),
		Name:        fromNull(name),
		Type:        fromNull(typ),
		Status:      fromNull(status),
		Condition:   fromNull(cond),
		Facility:    fromNull(fac),
		SubLocation: fromNull(sub_loc),
		Notes:       fromNull(notes),
		ClientTag:   fromNull(clientTag),
	}
	if created != nil {
		m.Created = *created
	}
	if updated != nil {
		m.Updated = *updated
	}
	return m, nil
}

const machineCols = "id, asset, name, type, status, condition, facility, sub_location, notes, client_tag, created, updated"

// machineFieldsEqual compares the user-editable fields (ignoring timestamps and
// the on-demand Parts slice, which makes Machine non-comparable with ==).
func machineFieldsEqual(a, b *Machine) bool {
	return a.Name == b.Name && a.Type == b.Type && a.Status == b.Status &&
		a.Condition == b.Condition && a.Facility == b.Facility &&
		a.SubLocation == b.SubLocation && a.Notes == b.Notes && a.ClientTag == b.ClientTag
}

func (d *SqlStore) FindMachine(id int) *Machine {
	rows, err := d.db.Query("SELECT "+machineCols+" FROM machine WHERE id=?", id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		m, _ := scanMachine(rows)
		return m
	}
	return nil
}

func (d *SqlStore) AllMachines() []*Machine {
	out := make([]*Machine, 0, 32)
	rows, err := d.db.Query("SELECT " + machineCols + " FROM machine ORDER BY id")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		if m, err := scanMachine(rows); err == nil {
			out = append(out, m)
		}
	}
	return out
}

func (d *SqlStore) CreateMachine(updater ModifyMachine) (*Machine, string) {
	m := &Machine{Status: MachineStatuses[0]}
	if !updater(m) {
		return nil, "cancelled"
	}
	now := time.Now()
	res, err := d.db.Exec(
		"INSERT INTO machine (name, type, status, condition, facility, sub_location, notes, client_tag, created, updated) VALUES (?,?,?,?,?,?,?,?,?,?)",
		str(m.Name), str(m.Type), str(m.Status), str(m.Condition), str(m.Facility), str(m.SubLocation), str(m.Notes), str(m.ClientTag), now, now)
	if err != nil {
		return nil, err.Error()
	}
	id64, _ := res.LastInsertId()
	m.Id = int(id64)
	m.Asset = fmt.Sprintf("TT-%04d", m.Id)
	m.Created, m.Updated = now, now
	// Backfill the generated asset tag.
	d.db.Exec("UPDATE machine SET asset=? WHERE id=?", m.Asset, m.Id)
	d.fts.UpdateMachine(m)
	return m, ""
}

func (d *SqlStore) EditMachine(id int, updater ModifyMachine) (bool, string) {
	m := d.FindMachine(id)
	if m == nil {
		return false, "machine not found"
	}
	before := *m
	if !updater(m) {
		return false, ""
	}
	if machineFieldsEqual(m, &before) {
		return false, "no change"
	}
	now := time.Now()
	_, err := d.db.Exec(
		"UPDATE machine SET name=?, type=?, status=?, condition=?, facility=?, sub_location=?, notes=?, client_tag=?, updated=? WHERE id=?",
		str(m.Name), str(m.Type), str(m.Status), str(m.Condition), str(m.Facility), str(m.SubLocation), str(m.Notes), str(m.ClientTag), now, id)
	if err != nil {
		return false, err.Error()
	}
	m.Updated = now
	d.fts.UpdateMachine(m)
	return true, ""
}

// ------------------------------------------------------------------- parts

func scanPart(rows *sql.Rows) (*Part, error) {
	var (
		id                                                       int
		machineId, quantity                                      *int
		category, model, spec, cond, serial, fac, sub_loc, notes *string
		created, updated                                         *time.Time
	)
	err := rows.Scan(&id, &machineId, &category, &model, &spec, &quantity, &cond, &serial, &fac, &sub_loc, &notes, &created, &updated)
	if err != nil {
		return nil, err
	}
	p := &Part{
		Id:          id,
		Category:    fromNull(category),
		Model:       fromNull(model),
		Spec:        fromNull(spec),
		Condition:   fromNull(cond),
		Serial:      fromNull(serial),
		Facility:    fromNull(fac),
		SubLocation: fromNull(sub_loc),
		Notes:       fromNull(notes),
	}
	if machineId != nil {
		p.MachineId = *machineId
	}
	if quantity != nil {
		p.Quantity = *quantity
	}
	if created != nil {
		p.Created = *created
	}
	if updated != nil {
		p.Updated = *updated
	}
	return p, nil
}

const partCols = "id, machine_id, category, model, spec, quantity, condition, serial, facility, sub_location, notes, created, updated"

func (d *SqlStore) FindPart(id int) *Part {
	rows, err := d.db.Query("SELECT "+partCols+" FROM part WHERE id=?", id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		p, _ := scanPart(rows)
		return p
	}
	return nil
}

func (d *SqlStore) AllParts() []*Part {
	out := make([]*Part, 0, 64)
	rows, err := d.db.Query("SELECT " + partCols + " FROM part ORDER BY id")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		if p, err := scanPart(rows); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func machineIdOrNull(machineId int) interface{} {
	if machineId == 0 {
		return nil
	}
	return machineId
}

func (d *SqlStore) PartsForMachine(machineId int) []*Part {
	out := make([]*Part, 0, 8)
	var (
		rows *sql.Rows
		err  error
	)
	if machineId == 0 {
		rows, err = d.db.Query("SELECT " + partCols + " FROM part WHERE machine_id IS NULL ORDER BY category, id")
	} else {
		rows, err = d.db.Query("SELECT "+partCols+" FROM part WHERE machine_id=? ORDER BY category, id", machineId)
	}
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		if p, err := scanPart(rows); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func (d *SqlStore) CreatePart(updater ModifyPart) (*Part, string) {
	p := &Part{Quantity: 1, Condition: "Untested"}
	if !updater(p) {
		return nil, "cancelled"
	}
	if p.Quantity < 1 {
		p.Quantity = 1
	}
	now := time.Now()
	res, err := d.db.Exec(
		"INSERT INTO part (machine_id, category, model, spec, quantity, condition, serial, facility, sub_location, notes, created, updated) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)",
		machineIdOrNull(p.MachineId), str(p.Category), str(p.Model), str(p.Spec), p.Quantity, str(p.Condition), str(p.Serial), str(p.Facility), str(p.SubLocation), str(p.Notes), now, now)
	if err != nil {
		return nil, err.Error()
	}
	id64, _ := res.LastInsertId()
	p.Id = int(id64)
	p.Created, p.Updated = now, now
	d.fts.UpdatePart(p)
	return p, ""
}

func (d *SqlStore) EditPart(id int, updater ModifyPart) (bool, string) {
	p := d.FindPart(id)
	if p == nil {
		return false, "part not found"
	}
	before := *p
	if !updater(p) {
		return false, ""
	}
	if *p == before {
		return false, "no change"
	}
	if p.Quantity < 1 {
		p.Quantity = 1
	}
	now := time.Now()
	_, err := d.db.Exec(
		"UPDATE part SET machine_id=?, category=?, model=?, spec=?, quantity=?, condition=?, serial=?, facility=?, sub_location=?, notes=?, updated=? WHERE id=?",
		machineIdOrNull(p.MachineId), str(p.Category), str(p.Model), str(p.Spec), p.Quantity, str(p.Condition), str(p.Serial), str(p.Facility), str(p.SubLocation), str(p.Notes), now, id)
	if err != nil {
		return false, err.Error()
	}
	p.Updated = now
	d.fts.UpdatePart(p)
	return true, ""
}

func (d *SqlStore) DeletePart(id int) bool {
	_, err := d.db.Exec("DELETE FROM part WHERE id=?", id)
	if err != nil {
		return false
	}
	d.fts.RemovePart(id)
	return true
}

// DeleteMachine removes a machine and any parts installed in it. Parts you want
// to keep should first be reassigned to loose inventory (machine 0) via the part
// edit form; anything still attached is deleted with the machine.
func (d *SqlStore) DeleteMachine(id int) (bool, int) {
	// Capture the installed parts up front so we can drop them from the search
	// index after the rows are gone.
	parts := d.PartsForMachine(id)

	if _, err := d.db.Exec("DELETE FROM part WHERE machine_id=?", id); err != nil {
		return false, 0
	}
	if _, err := d.db.Exec("DELETE FROM machine WHERE id=?", id); err != nil {
		return false, 0
	}

	for _, p := range parts {
		d.fts.RemovePart(p.Id)
	}
	d.fts.RemoveMachine(id)
	return true, len(parts)
}

func (d *SqlStore) Search(term string) *SearchResult {
	return d.fts.Search(term, d)
}

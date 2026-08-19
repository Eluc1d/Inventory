package main

import (
	"database/sql"
	"fmt"
	"time"
)

var ticketSchema = `
create table if not exists ticket (
    id       integer primary key autoincrement,
    number   varchar(20),
    title    varchar(160),
    client   varchar(160),
    deadline varchar(10) not null default '', -- "YYYY-MM-DD" or "", never NULL (see AllTickets)
    status   varchar(20),
    notes    text,
    created  timestamp,
    updated  timestamp
);

create table if not exists ticket_item (
    id          integer primary key autoincrement,
    ticket_id   integer not null,
    kind        varchar(20),
    description text,
    quantity    integer default 1,
    created     timestamp,
    foreign key(ticket_id) references ticket(id)
);

create table if not exists ticket_attachment (
    id            integer primary key autoincrement,
    ticket_id     integer not null,
    filename      varchar(200),
    original_name varchar(200),
    content_type  varchar(100),
    size          integer,
    created       timestamp,
    foreign key(ticket_id) references ticket(id)
);

create index if not exists idx_ticket_item_ticket on ticket_item(ticket_id);
create index if not exists idx_ticket_attachment_ticket on ticket_attachment(ticket_id);
`

// initTicketSchema creates the ticket tables if they don't already exist.
// Called from NewSqlStore alongside the machine/part schema.
func initTicketSchema(db *sql.DB) error {
	_, err := db.Exec(ticketSchema)
	return err
}

// ------------------------------------------------------------------ tickets

func scanTicket(rows *sql.Rows) (*Ticket, error) {
	var (
		id                                   int
		number, title, client, status, notes *string
		deadline                             string
		created, updated                     *time.Time
	)
	err := rows.Scan(&id, &number, &title, &client, &deadline, &status, &notes, &created, &updated)
	if err != nil {
		return nil, err
	}
	t := &Ticket{
		Id:       id,
		Number:   fromNull(number),
		Title:    fromNull(title),
		Client:   fromNull(client),
		Deadline: deadline,
		Status:   fromNull(status),
		Notes:    fromNull(notes),
	}
	if created != nil {
		t.Created = *created
	}
	if updated != nil {
		t.Updated = *updated
	}
	return t, nil
}

const ticketCols = "id, number, title, client, deadline, status, notes, created, updated"

// ticketFieldsEqual compares the user-editable fields. Ticket can't use a
// plain == comparison (like Part does) because its Items/Attachments slice
// fields make it non-comparable.
func ticketFieldsEqual(a, b *Ticket) bool {
	return a.Title == b.Title && a.Client == b.Client && a.Deadline == b.Deadline &&
		a.Status == b.Status && a.Notes == b.Notes
}

func (d *SqlStore) FindTicket(id int) *Ticket {
	rows, err := d.db.Query("SELECT "+ticketCols+" FROM ticket WHERE id=?", id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		t, _ := scanTicket(rows)
		return t
	}
	return nil
}

// AllTickets returns every ticket sorted by deadline ascending, with blank
// deadlines sorted LAST. Plain ASC on the deadline column alone would put ""
// before any real date (both in SQLite and in a naive Go string compare),
// which is backwards for a worklist of actionable tickets — sorting first by
// the boolean "is this blank" avoids that.
func (d *SqlStore) AllTickets() []*Ticket {
	out := make([]*Ticket, 0, 16)
	rows, err := d.db.Query("SELECT " + ticketCols + " FROM ticket ORDER BY (deadline = '') ASC, deadline ASC, id ASC")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		if t, err := scanTicket(rows); err == nil {
			out = append(out, t)
		}
	}
	return out
}

func (d *SqlStore) CreateTicket(updater ModifyTicket) (*Ticket, string) {
	t := &Ticket{Status: TicketStatuses[0]}
	if !updater(t) {
		return nil, "cancelled"
	}
	now := time.Now()
	res, err := d.db.Exec(
		"INSERT INTO ticket (title, client, deadline, status, notes, created, updated) VALUES (?,?,?,?,?,?,?)",
		str(t.Title), str(t.Client), t.Deadline, str(t.Status), str(t.Notes), now, now)
	if err != nil {
		return nil, err.Error()
	}
	id64, _ := res.LastInsertId()
	t.Id = int(id64)
	t.Number = fmt.Sprintf("TK-%04d", t.Id)
	t.Created, t.Updated = now, now
	// Backfill the generated ticket number, same as Machine's Asset tag.
	d.db.Exec("UPDATE ticket SET number=? WHERE id=?", t.Number, t.Id)
	return t, ""
}

func (d *SqlStore) EditTicket(id int, updater ModifyTicket) (bool, string) {
	t := d.FindTicket(id)
	if t == nil {
		return false, "ticket not found"
	}
	before := *t
	if !updater(t) {
		return false, ""
	}
	if ticketFieldsEqual(t, &before) {
		return false, "no change"
	}
	now := time.Now()
	_, err := d.db.Exec(
		"UPDATE ticket SET title=?, client=?, deadline=?, status=?, notes=?, updated=? WHERE id=?",
		str(t.Title), str(t.Client), t.Deadline, str(t.Status), str(t.Notes), now, id)
	if err != nil {
		return false, err.Error()
	}
	t.Updated = now
	return true, ""
}

// DeleteTicket removes a ticket, its items, and its attachment DB rows. It
// returns the removed attachments so the caller — which owns uploadsDir, not
// the store — can also delete the on-disk files.
func (d *SqlStore) DeleteTicket(id int) (bool, []*TicketAttachment) {
	attachments := d.AttachmentsForTicket(id)

	if _, err := d.db.Exec("DELETE FROM ticket_item WHERE ticket_id=?", id); err != nil {
		return false, nil
	}
	if _, err := d.db.Exec("DELETE FROM ticket_attachment WHERE ticket_id=?", id); err != nil {
		return false, nil
	}
	if _, err := d.db.Exec("DELETE FROM ticket WHERE id=?", id); err != nil {
		return false, nil
	}
	return true, attachments
}

// ------------------------------------------------------------- ticket items

func scanTicketItem(rows *sql.Rows) (*TicketItem, error) {
	var (
		id, ticketId, quantity int
		kind, description      *string
		created                *time.Time
	)
	err := rows.Scan(&id, &ticketId, &kind, &description, &quantity, &created)
	if err != nil {
		return nil, err
	}
	it := &TicketItem{
		Id:          id,
		TicketId:    ticketId,
		Kind:        fromNull(kind),
		Description: fromNull(description),
		Quantity:    quantity,
	}
	if created != nil {
		it.Created = *created
	}
	return it, nil
}

const ticketItemCols = "id, ticket_id, kind, description, quantity, created"

func (d *SqlStore) ItemsForTicket(ticketId int) []*TicketItem {
	out := make([]*TicketItem, 0, 4)
	rows, err := d.db.Query("SELECT "+ticketItemCols+" FROM ticket_item WHERE ticket_id=? ORDER BY id", ticketId)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		if it, err := scanTicketItem(rows); err == nil {
			out = append(out, it)
		}
	}
	return out
}

// ReplaceTicketItems atomically deletes every existing item on ticketId and
// inserts the given set in its place. This is the one place in this codebase
// that uses a transaction (everywhere else is a single Exec) — unlike
// applyStagedParts (purely additive), a delete-then-recreate is destructive
// if interrupted mid-loop, so the whole thing must commit or roll back as a
// unit rather than risk leaving a ticket with zero items.
func (d *SqlStore) ReplaceTicketItems(ticketId int, items []*TicketItem) (bool, string) {
	tx, err := d.db.Begin()
	if err != nil {
		return false, err.Error()
	}
	defer tx.Rollback() // no-op once committed

	if _, err := tx.Exec("DELETE FROM ticket_item WHERE ticket_id=?", ticketId); err != nil {
		return false, err.Error()
	}

	now := time.Now()
	for _, it := range items {
		qty := it.Quantity
		if qty < 1 {
			qty = 1
		}
		if _, err := tx.Exec(
			"INSERT INTO ticket_item (ticket_id, kind, description, quantity, created) VALUES (?,?,?,?,?)",
			ticketId, str(it.Kind), str(it.Description), qty, now,
		); err != nil {
			return false, err.Error()
		}
	}

	if err := tx.Commit(); err != nil {
		return false, err.Error()
	}
	return true, ""
}

// ------------------------------------------------------------- attachments

func scanTicketAttachment(rows *sql.Rows) (*TicketAttachment, error) {
	var (
		id, ticketId                        int
		filename, originalName, contentType *string
		size                                int64
		created                             *time.Time
	)
	err := rows.Scan(&id, &ticketId, &filename, &originalName, &contentType, &size, &created)
	if err != nil {
		return nil, err
	}
	a := &TicketAttachment{
		Id:           id,
		TicketId:     ticketId,
		Filename:     fromNull(filename),
		OriginalName: fromNull(originalName),
		ContentType:  fromNull(contentType),
		Size:         size,
	}
	if created != nil {
		a.Created = *created
	}
	return a, nil
}

const ticketAttachmentCols = "id, ticket_id, filename, original_name, content_type, size, created"

func (d *SqlStore) AttachmentsForTicket(ticketId int) []*TicketAttachment {
	out := make([]*TicketAttachment, 0, 4)
	rows, err := d.db.Query("SELECT "+ticketAttachmentCols+" FROM ticket_attachment WHERE ticket_id=? ORDER BY id", ticketId)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		if a, err := scanTicketAttachment(rows); err == nil {
			out = append(out, a)
		}
	}
	return out
}

func (d *SqlStore) FindAttachment(id int) *TicketAttachment {
	rows, err := d.db.Query("SELECT "+ticketAttachmentCols+" FROM ticket_attachment WHERE id=?", id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		a, _ := scanTicketAttachment(rows)
		return a
	}
	return nil
}

func (d *SqlStore) CreateAttachment(updater ModifyTicketAttachment) (*TicketAttachment, string) {
	a := &TicketAttachment{}
	if !updater(a) {
		return nil, "cancelled"
	}
	now := time.Now()
	res, err := d.db.Exec(
		"INSERT INTO ticket_attachment (ticket_id, filename, original_name, content_type, size, created) VALUES (?,?,?,?,?,?)",
		a.TicketId, str(a.Filename), str(a.OriginalName), str(a.ContentType), a.Size, now)
	if err != nil {
		return nil, err.Error()
	}
	id64, _ := res.LastInsertId()
	a.Id = int(id64)
	a.Created = now
	return a, ""
}

// DeleteAttachment removes the DB row only. The caller is responsible for
// os.Remove-ing the on-disk file — the store doesn't know uploadsDir.
func (d *SqlStore) DeleteAttachment(id int) bool {
	_, err := d.db.Exec("DELETE FROM ticket_attachment WHERE id=?", id)
	return err == nil
}

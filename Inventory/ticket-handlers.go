package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (h *Handlers) RegisterTickets(mux *http.ServeMux) {
	mux.HandleFunc("/tickets", h.ticketList)
	mux.HandleFunc("/ticket", h.ticketDetail)
	mux.HandleFunc("/ticket/edit", h.ticketEdit)
	mux.HandleFunc("/ticket/delete", h.ticketDelete)
	mux.HandleFunc("/ticket/attachment/upload", h.attachmentUpload)
	mux.HandleFunc("/ticket/attachment/delete", h.attachmentDelete)
}

// ---------------------------------------------------------------- ticket list

type ticketListData struct {
	Tickets      []*Ticket
	Statuses     []string
	StatusFilter string
	Total        int
}

func (h *Handlers) ticketList(w http.ResponseWriter, r *http.Request) {
	tickets := h.store.AllTickets()
	filter := r.URL.Query().Get("status")

	shown := tickets
	if filter != "" {
		shown = shown[:0]
		for _, t := range tickets {
			if t.Status == filter {
				shown = append(shown, t)
			}
		}
	}

	// Attachment counts drive the card's "N files" badge. Small-scale
	// internal tool, so an attachment fetch per shown ticket is fine.
	for _, t := range shown {
		t.Attachments = h.store.AttachmentsForTicket(t.Id)
	}

	h.tmpl.Render(w, http.StatusOK, "ticket-list.html", &page{
		Title:    "Tickets",
		Editable: h.canEdit(r),
		Active:   "tickets",
		Data: &ticketListData{
			Tickets:      shown,
			Statuses:     TicketStatuses,
			StatusFilter: filter,
			Total:        len(tickets),
		},
	})
}

// -------------------------------------------------------------- ticket detail

type ticketDetailData struct {
	Ticket *Ticket
	// RejectedCount surfaces "N file(s) rejected" after an upload attempt
	// that included a disallowed extension, via a ?rejected=N redirect param.
	RejectedCount int
}

func (h *Handlers) ticketDetail(w http.ResponseWriter, r *http.Request) {
	id := atoiDefault(r.URL.Query().Get("id"), 0)
	t := h.store.FindTicket(id)
	if t == nil {
		http.NotFound(w, r)
		return
	}
	t.Items = h.store.ItemsForTicket(t.Id)
	t.Attachments = h.store.AttachmentsForTicket(t.Id)

	h.tmpl.Render(w, http.StatusOK, "ticket-detail.html", &page{
		Title:    t.Number + " " + t.Title,
		Editable: h.canEdit(r),
		Active:   "tickets",
		Data: &ticketDetailData{
			Ticket:        t,
			RejectedCount: atoiDefault(r.URL.Query().Get("rejected"), 0),
		},
	})
}

// ---------------------------------------------------------------- ticket edit

type ticketFormData struct {
	Ticket *Ticket
	IsNew  bool
}

// stagedTicketItemRow is the wire shape posted from ticket-form.html's item
// builder — kept distinct from TicketItem so client-supplied JSON can't
// reach Id/TicketId/Created, mirroring stagedPartRow's separation from Part.
type stagedTicketItemRow struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
}

func validTicketItemKind(k string) bool {
	for _, valid := range TicketItemKinds {
		if k == valid {
			return true
		}
	}
	return false
}

// applyTicketItems replaces ticketId's items from the posted items_json
// field. Requires r.ParseForm to have already been called. A form submitted
// without an items_json key at all (e.g. the page's JS failed to run) leaves
// existing items untouched; a present-but-empty "[]" is a legitimate "the
// user deleted every item" and is honored — the two are deliberately not
// conflated, since conflating them would silently wipe items on any request
// where the staging JS didn't execute.
func (h *Handlers) applyTicketItems(r *http.Request, ticketId int) {
	if _, present := r.Form["items_json"]; !present {
		return
	}
	raw := r.FormValue("items_json")
	var rows []stagedTicketItemRow
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		log.Printf("applyTicketItems: bad items_json: %s", err)
		return
	}
	items := make([]*TicketItem, 0, len(rows))
	for _, row := range rows {
		desc := strings.TrimSpace(row.Description)
		if desc == "" {
			continue
		}
		kind := strings.TrimSpace(row.Kind)
		if !validTicketItemKind(kind) {
			kind = "Custom"
		}
		qty := row.Quantity
		if qty < 1 {
			qty = 1
		}
		items = append(items, &TicketItem{Kind: kind, Description: desc, Quantity: qty})
	}
	if ok, errStr := h.store.ReplaceTicketItems(ticketId, items); !ok {
		log.Printf("applyTicketItems: failed to replace items for ticket %d: %s", ticketId, errStr)
	}
}

func (h *Handlers) ticketEdit(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}

	idParam := r.URL.Query().Get("id")
	isNew := idParam == "" || idParam == "new"

	if r.Method == http.MethodPost {
		// The form carries an optional file input alongside its text fields,
		// so it POSTs as multipart/form-data — parse it as such rather than
		// r.ParseForm (which ignores multipart bodies entirely). As with
		// attachmentUpload, MaxBytesReader must wrap the body before
		// anything reads it. ParseMultipartForm still populates r.Form for
		// the plain text fields (title/client/.../items_json), so
		// r.FormValue and the r.Form["items_json"] presence check below
		// continue to work unchanged.
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			http.Error(w, "Invalid form submission, or attachments too large.", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		client := strings.TrimSpace(r.FormValue("client"))
		deadline := strings.TrimSpace(r.FormValue("deadline"))
		status := strings.TrimSpace(r.FormValue("status"))
		notes := strings.TrimSpace(r.FormValue("notes"))

		var files []*multipart.FileHeader
		if r.MultipartForm != nil {
			files = r.MultipartForm.File["files"]
		}

		if isNew {
			t, errStr := h.store.CreateTicket(func(t *Ticket) bool {
				t.Title = title
				t.Client = client
				t.Deadline = deadline
				t.Status = status
				t.Notes = notes
				return true
			})
			if t == nil {
				http.Error(w, errStr, http.StatusBadRequest)
				return
			}
			h.applyTicketItems(r, t.Id)
			rejected := h.processAttachmentUploads(files, t.Id)
			http.Redirect(w, r, ticketRedirectURL(t.Id, rejected), http.StatusSeeOther)
			return
		}

		id := atoiDefault(idParam, 0)
		if ok, errStr := h.store.EditTicket(id, func(t *Ticket) bool {
			t.Title = title
			t.Client = client
			t.Deadline = deadline
			t.Status = status
			t.Notes = notes
			return true
		}); !ok && errStr != "" && errStr != "no change" {
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}
		// Items may have changed even when the ticket's own fields didn't
		// (EditTicket returning "no change" above must not skip this).
		h.applyTicketItems(r, id)
		rejected := h.processAttachmentUploads(files, id)
		http.Redirect(w, r, ticketRedirectURL(id, rejected), http.StatusSeeOther)
		return
	}

	t := &Ticket{}
	if !isNew {
		id := atoiDefault(idParam, 0)
		found := h.store.FindTicket(id)
		if found == nil {
			http.NotFound(w, r)
			return
		}
		t = found
		t.Items = h.store.ItemsForTicket(t.Id)
	}

	h.tmpl.Render(w, http.StatusOK, "ticket-form.html", &page{
		Title:    "Log a ticket",
		Editable: true,
		Active:   "tickets",
		Data: &ticketFormData{
			Ticket: t,
			IsNew:  isNew,
		},
	})
}

// ticketRedirectURL builds the post-save destination for a ticket, appending
// a rejected-file count for ticket-detail.html's flash message when relevant.
func ticketRedirectURL(ticketId, rejected int) string {
	url := "/ticket?id=" + strconv.Itoa(ticketId)
	if rejected > 0 {
		url += "&rejected=" + strconv.Itoa(rejected)
	}
	return url
}

func (h *Handlers) ticketDelete(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tickets", http.StatusSeeOther)
		return
	}
	id := atoiDefault(r.FormValue("id"), 0)
	ok, removed := h.store.DeleteTicket(id)
	if !ok {
		log.Printf("ticketDelete: failed to delete ticket %d", id)
	}
	for _, a := range removed {
		h.removeAttachmentFile(a)
	}
	http.Redirect(w, r, "/tickets", http.StatusSeeOther)
}

// removeAttachmentFile deletes an attachment's on-disk file. Missing files
// (already gone, or the row never had one) are not an error.
func (h *Handlers) removeAttachmentFile(a *TicketAttachment) {
	if a == nil || a.Filename == "" {
		return
	}
	path := filepath.Join(h.uploadsDir, a.Filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("removeAttachmentFile: failed to remove %q: %v", path, err)
	}
}

// ----------------------------------------------------------------- uploads

// maxUploadBytes caps the whole multipart request (covers multiple files in
// one submit), enforced via http.MaxBytesReader below — ParseMultipartForm's
// own size argument only bounds in-memory buffering, not total request size,
// so relying on that alone would let an oversized upload spool to disk
// unrejected.
const maxUploadBytes = 20 << 20 // 20MB

// allowedAttachmentExts is the actual safety mechanism for serving uploads
// back through the plain /uploads/ static mount: http.FileServer infers
// Content-Type from the file extension at serve time, so excluding
// .html/.svg/.js here (regardless of what Content-Type the uploader claims)
// is what prevents a malicious upload from becoming a same-origin stored-XSS
// payload when someone later opens /uploads/<file> in a browser.
var allowedAttachmentExts = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".txt": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
}

// randomAttachmentFilename generates the on-disk filename — never derived
// from the client's original filename, which blocks both path traversal and
// same-name collisions between different uploads.
func randomAttachmentFilename(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}

// processAttachmentUploads writes each uploaded file to disk (subject to the
// extension allow-list) and creates its attachment record against ticketId.
// Shared by attachmentUpload (the detail page's standalone uploader) and
// ticketEdit (attaching files at create/edit time) so this — the
// security-sensitive part — has exactly one implementation. Returns how many
// files were rejected for a disallowed extension, for the caller to surface.
func (h *Handlers) processAttachmentUploads(files []*multipart.FileHeader, ticketId int) int {
	rejected := 0
	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedAttachmentExts[ext] {
			rejected++
			continue
		}

		src, err := fh.Open()
		if err != nil {
			log.Printf("processAttachmentUploads: open %q: %v", fh.Filename, err)
			continue
		}

		storedName, err := randomAttachmentFilename(ext)
		if err != nil {
			src.Close()
			log.Printf("processAttachmentUploads: generating filename: %v", err)
			continue
		}
		storedPath := filepath.Join(h.uploadsDir, storedName)

		dst, err := os.OpenFile(storedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			src.Close()
			log.Printf("processAttachmentUploads: creating %q: %v", storedName, err)
			continue
		}
		written, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			os.Remove(storedPath)
			log.Printf("processAttachmentUploads: writing %q: %v", storedName, copyErr)
			continue
		}

		if _, errStr := h.store.CreateAttachment(func(a *TicketAttachment) bool {
			a.TicketId = ticketId
			a.Filename = storedName
			a.OriginalName = fh.Filename
			a.ContentType = fh.Header.Get("Content-Type")
			a.Size = written
			return true
		}); errStr != "" {
			os.Remove(storedPath)
			log.Printf("processAttachmentUploads: create attachment record: %s", errStr)
		}
	}
	return rejected
}

func (h *Handlers) attachmentUpload(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tickets", http.StatusSeeOther)
		return
	}

	// MaxBytesReader must wrap the body BEFORE anything reads it — including
	// the ticket_id lookup below. r.FormValue implicitly parses the
	// multipart body on first access; calling it first (as an earlier draft
	// of this handler did) would read the whole upload under Go's default
	// 32MB cap before this limit ever got a chance to apply.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		http.Error(w, "Upload too large or invalid.", http.StatusRequestEntityTooLarge)
		return
	}

	ticketId := atoiDefault(r.FormValue("ticket_id"), 0)
	if h.store.FindTicket(ticketId) == nil {
		http.NotFound(w, r)
		return
	}

	var files []*multipart.FileHeader
	if r.MultipartForm != nil {
		files = r.MultipartForm.File["files"]
	}
	rejected := h.processAttachmentUploads(files, ticketId)
	http.Redirect(w, r, ticketRedirectURL(ticketId, rejected), http.StatusSeeOther)
}

func (h *Handlers) attachmentDelete(w http.ResponseWriter, r *http.Request) {
	if !h.canEdit(r) {
		http.Error(w, "Editing not permitted from your network.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tickets", http.StatusSeeOther)
		return
	}
	id := atoiDefault(r.FormValue("id"), 0)
	ticketId := atoiDefault(r.FormValue("ticket_id"), 0)

	if a := h.store.FindAttachment(id); a != nil {
		h.store.DeleteAttachment(id)
		h.removeAttachmentFile(a)
	}
	http.Redirect(w, r, "/ticket?id="+strconv.Itoa(ticketId), http.StatusSeeOther)
}

// TechToss — inventory for computers being refurbished and their components.
// Backed by a single SQLite file. No web framework: net/http + html/template.
package main

import (
	"database/sql"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func parseEditNets(spec string) []*net.IPNet {
	var nets []*net.IPNet
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, n, err := net.ParseCIDR(part)
		if err != nil {
			log.Fatalf("--edit-permission-nets: need IP/mask CIDR: %v", err)
		}
		nets = append(nets, n)
	}
	return nets
}

// noDirListing wraps a file-serving handler so a request for a directory
// 404s instead of listing its contents — attachment filenames are random
// tokens, and a directory listing would let anyone enumerate them, defeating
// that as an access control. Checks for "" as well as a trailing "/": this
// sits behind http.StripPrefix("/uploads/", ...), so a request for exactly
// "/uploads/" arrives here with Path already stripped down to "", not "/".
func noDirListing(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	version = detectVersion()

	bindAddress := flag.String("bind-address", ":2000", "Listen address:port")
	dbFile := flag.String("dbfile", "techtoss.db", "SQLite database file")
	templateDir := flag.String("templatedir", "./template", "Directory with HTML templates")
	staticDir := flag.String("staticdir", "./static", "Directory with static assets")
	uploadsDir := flag.String("uploadsdir", "./uploads", "Directory for uploaded ticket attachments")
	cacheTemplates := flag.Bool("cache-templates", true, "Cache templates (false for live editing)")
	editNetsSpec := flag.String("edit-permission-nets", "",
		"Comma-separated CIDR networks allowed to edit (empty = everyone)")
	seed := flag.Bool("seed", false, "Seed the database with a few example machines/parts if empty")
	seedTickets := flag.Bool("seed-tickets", false, "Add sample tickets for testing (additive; safe to run even if tickets already exist)")
	flag.Parse()

	if _, err := os.Stat(*dbFile); err != nil {
		log.Printf("Creating new database file %q", *dbFile)
	}

	if err := os.MkdirAll(*uploadsDir, 0755); err != nil {
		log.Fatalf("creating uploads directory %q: %v", *uploadsDir, err)
	}

	db, err := sql.Open("sqlite3", *dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var store Store
	sqlStore, err := NewSqlStore(db)
	if err != nil {
		log.Fatal(err)
	}
	store = sqlStore

	if *seed {
		seedIfEmpty(store)
	}
	if *seedTickets {
		seedSampleTickets(store)
	}

	tmpl := NewTemplates(*templateDir, *cacheTemplates)
	editNets := parseEditNets(*editNetsSpec)
	handlers := NewHandlers(store, tmpl, editNets, *uploadsDir)

	mux := http.NewServeMux()
	handlers.Register(mux)
	handlers.RegisterSearch(mux)
	handlers.RegisterAPI(mux)
	handlers.RegisterTickets(mux)
	mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(*staticDir))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/",
		noDirListing(http.FileServer(http.Dir(*uploadsDir)))))

	log.Printf("TechToss listening on %q (db=%s, editable=%v)",
		*bindAddress, *dbFile, len(editNets) == 0)
	log.Fatal(http.ListenAndServe(*bindAddress, mux))
}

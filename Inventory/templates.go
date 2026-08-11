package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// funcs available inside every template.
var templateFuncs = template.FuncMap{
	"lower": strings.ToLower,
	// statusSlug turns "For Parts" into "for-parts" for CSS class hooks.
	"statusSlug": func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "-")
	},
	"machineStatuses": func() []string { return MachineStatuses },
	"partCategories":  func() []string { return PartCategories },
	"partConditions":  func() []string { return PartConditions },
	"facilityOptions": func() []string { return FacilityOptions },
}

type Templates struct {
	baseDir string
	cache   *template.Template
	doCache bool
}

func NewTemplates(baseDir string, doCache bool) *Templates {
	t := &Templates{baseDir: baseDir, doCache: doCache}
	if doCache {
		pattern := filepath.Join(baseDir, "*.html")
		t.cache = template.Must(
			template.New("").Funcs(templateFuncs).ParseGlob(pattern))
	}
	return t
}

// Render executes a named template (e.g. "machine-list.html") with data.
func (t *Templates) Render(w http.ResponseWriter, code int, name string, data interface{}) {
	var tmpl *template.Template
	if t.doCache {
		tmpl = t.cache
	} else {
		var err error
		tmpl, err = template.New("").Funcs(templateFuncs).ParseGlob(filepath.Join(t.baseDir, "*.html"))
		if err != nil {
			log.Printf("template parse: %v", err)
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template exec %s: %v", name, err)
	}
}

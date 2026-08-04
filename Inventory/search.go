package main

import (
	"sort"
	"strings"
	"sync"
)

// Search is a small in-memory index. Each machine and part contributes a bag of
// lowercased tokens plus a single "haystack" string used for scoring. Writes go
// through the Store, which calls UpdateMachine/UpdatePart to keep this current.
type Search struct {
	mu       sync.RWMutex
	machines map[int]string // id -> haystack
	parts    map[int]string // id -> haystack
}

func NewSearch() *Search {
	return &Search{
		machines: make(map[int]string),
		parts:    make(map[int]string),
	}
}

func normalize(s string) string {
	// Case-insensitive, and treat dashes as joiners so "DDR-4" == "ddr4".
	return strings.ReplaceAll(strings.ToLower(s), "-", "")
}

func (s *Search) UpdateMachine(m *Machine) {
	hay := normalize(strings.Join([]string{
		m.Asset, m.Name, m.Type, m.Status, m.Condition, m.Facility, m.SubLocation, m.Notes,
	}, " "))
	s.mu.Lock()
	s.machines[m.Id] = hay
	s.mu.Unlock()
}

func (s *Search) UpdatePart(p *Part) {
	hay := normalize(strings.Join([]string{
		p.Category, p.Model, p.Spec, p.Condition, p.Serial, p.Notes,
	}, " "))
	s.mu.Lock()
	s.parts[p.Id] = hay
	s.mu.Unlock()
}

func (s *Search) RemovePart(id int) {
	s.mu.Lock()
	delete(s.parts, id)
	s.mu.Unlock()
}

// score returns >0 when every whitespace-separated term in the (already
// normalized) query is present in the haystack. Exact word-boundary and prefix
// matches rank higher than mid-word substring matches.
func score(terms []string, hay string) float32 {
	if len(terms) == 0 {
		return 0
	}
	var total float32
	for _, t := range terms {
		pos := strings.Index(hay, t)
		if pos < 0 {
			return 0 // AND semantics: a missing term disqualifies the row
		}
		var s float32 = 1
		atWordStart := pos == 0 || hay[pos-1] == ' '
		end := pos + len(t)
		atWordEnd := end == len(hay) || hay[end] == ' '
		if atWordStart && atWordEnd {
			s = 4 // whole-word hit
		} else if atWordStart {
			s = 2 // prefix hit
		}
		total += s
	}
	return total
}

func (s *Search) Search(term string, store Store) *SearchResult {
	result := &SearchResult{Query: term}
	terms := strings.Fields(normalize(term))
	if len(terms) == 0 {
		return result
	}

	type scored struct {
		hit   *SearchHit
		score float32
	}
	var hits []scored

	s.mu.RLock()
	machineHay := make(map[int]string, len(s.machines))
	for id, h := range s.machines {
		machineHay[id] = h
	}
	partHay := make(map[int]string, len(s.parts))
	for id, h := range s.parts {
		partHay[id] = h
	}
	s.mu.RUnlock()

	for id, hay := range machineHay {
		if sc := score(terms, hay); sc > 0 {
			if m := store.FindMachine(id); m != nil {
				hits = append(hits, scored{&SearchHit{Kind: "machine", Machine: m, Score: sc}, sc})
			}
		}
	}
	for id, hay := range partHay {
		if sc := score(terms, hay); sc > 0 {
			if p := store.FindPart(id); p != nil {
				hits = append(hits, scored{&SearchHit{Kind: "part", Part: p, Score: sc}, sc})
			}
		}
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		// Stable tie-break: machines before parts, then by id.
		ki, kj := hits[i].hit.Kind, hits[j].hit.Kind
		if ki != kj {
			return ki == "machine"
		}
		return idOf(hits[i].hit) < idOf(hits[j].hit)
	})

	for _, h := range hits {
		result.Results = append(result.Results, h.hit)
	}
	return result
}

func idOf(h *SearchHit) int {
	if h.Machine != nil {
		return h.Machine.Id
	}
	if h.Part != nil {
		return h.Part.Id
	}
	return 0
}

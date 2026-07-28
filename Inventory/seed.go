package main

import "log"

// seedIfEmpty populates a fresh database with a few realistic examples so the
// UI isn't blank on first run. No-op if any machines already exist.
func seedIfEmpty(store Store) {
	if len(store.AllMachines()) > 0 {
		return
	}
	log.Printf("Seeding example data")

	type partSpec struct {
		category, model, spec, condition string
		qty                              int
	}
	examples := []struct {
		name, typ, status, condition, location, notes string
		parts                                         []partSpec
	}{
		{
			name: "Dell OptiPlex 7060", typ: "Desktop", status: "Testing",
			condition: "Good", location: "Bench 1",
			notes: "Donated batch #4. Boots, needs memtest.",
			parts: []partSpec{
				{"CPU", "Intel Core i5-8500", "6c/6t, 3.0GHz", "Working", 1},
				{"RAM", "Crucial 8GB DDR4-2666", "SODIMM", "Working", 2},
				{"Storage", "WD Blue 256GB SSD", "SATA 2.5\"", "Untested", 1},
				{"PSU", "Dell 200W", "internal SFF", "Working", 1},
			},
		},
		{
			name: "Custom Gaming Tower", typ: "Desktop", status: "Repair",
			condition: "Fair", location: "Bench 2",
			notes: "No POST. Suspect GPU or PSU.",
			parts: []partSpec{
				{"GPU", "NVIDIA RTX 2060", "6GB GDDR6", "Faulty", 1},
				{"RAM", "Corsair Vengeance 16GB DDR4-3200", "2x8GB", "Working", 2},
				{"Motherboard", "ASUS Prime B450M-A", "AM4", "Untested", 1},
			},
		},
		{
			name: "ThinkPad T480", typ: "Laptop", status: "Refurbished",
			condition: "Excellent", location: "Shelf A",
			notes: "Ready for sale. New battery installed.",
			parts: []partSpec{
				{"CPU", "Intel Core i7-8650U", "4c/8t", "Working", 1},
				{"Storage", "Samsung 970 EVO 500GB", "NVMe", "Working", 1},
			},
		},
	}

	for _, ex := range examples {
		ex := ex
		m, errStr := store.CreateMachine(func(m *Machine) bool {
			m.Name = ex.name
			m.Type = ex.typ
			m.Status = ex.status
			m.Condition = ex.condition
			m.Location = ex.location
			m.Notes = ex.notes
			return true
		})
		if m == nil {
			log.Printf("seed machine failed: %s", errStr)
			continue
		}
		for _, ps := range ex.parts {
			ps := ps
			store.CreatePart(func(p *Part) bool {
				p.MachineId = m.Id
				p.Category = ps.category
				p.Model = ps.model
				p.Spec = ps.spec
				p.Condition = ps.condition
				p.Quantity = ps.qty
				return true
			})
		}
	}

	// A couple of harvested, loose parts (no machine).
	loose := []partSpec{
		{"GPU", "NVIDIA GTX 1050 Ti", "4GB, harvested & tested", "Working", 1},
		{"RAM", "Kingston 4GB DDR3-1600", "DIMM, pulled from scrap", "Working", 6},
	}
	for _, ps := range loose {
		ps := ps
		store.CreatePart(func(p *Part) bool {
			p.MachineId = 0
			p.Category = ps.category
			p.Model = ps.model
			p.Spec = ps.spec
			p.Condition = ps.condition
			p.Quantity = ps.qty
			return true
		})
	}
}

package main

import "log"

// seedIfEmpty populates a fresh database with a broad set of realistic
// examples — every machine status, several facilities, and a mix of loose
// inventory categories and quantities — so the UI has enough variety to
// actually exercise on first run. No-op if any machines already exist.
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
		name, typ, status, condition, facility, sub_location, notes string
		parts                                                       []partSpec
	}{
		{
			name: "Dell OptiPlex 7060", typ: "Desktop", status: "Testing",
			condition: "Good", facility: "Techtoss", sub_location: "Rack A",
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
			condition: "Fair", facility: "Techtoss", sub_location: "Rack B",
			notes: "No POST. Suspect GPU or PSU.",
			parts: []partSpec{
				{"GPU", "NVIDIA RTX 2060", "6GB GDDR6", "Faulty", 1},
				{"RAM", "Corsair Vengeance 16GB DDR4-3200", "2x8GB", "Working", 2},
				{"Motherboard", "ASUS Prime B450M-A", "AM4", "Untested", 1},
			},
		},
		{
			name: "ThinkPad T480", typ: "Laptop", status: "Refurbished",
			condition: "Excellent", facility: "Bruce's Storage", sub_location: "Floor",
			notes: "Ready for sale. New battery installed.",
			parts: []partSpec{
				{"CPU", "Intel Core i7-8650U", "4c/8t", "Working", 1},
				{"Storage", "Samsung 970 EVO 500GB", "NVMe", "Working", 1},
			},
		},
		{
			name: "HP EliteDesk 800 G3", typ: "Mini PC", status: "Intake",
			condition: "Unknown", facility: "Techtoss", sub_location: "Shelf 1",
			notes: "Just dropped off, not yet tested.",
			parts: []partSpec{
				{"RAM", "HP 8GB DDR4-2400", "SODIMM", "Untested", 1},
			},
		},
		{
			name: "Dell PowerEdge R610", typ: "Server", status: "For Parts",
			condition: "Poor", facility: "Connor's Garage", sub_location: "Floor",
			notes: "Dead PSU, board seems fine — stripping for parts.",
			parts: []partSpec{
				{"CPU", "Intel Xeon L5640", "6c/12t, 2.26GHz", "Untested", 2},
				{"RAM", "Samsung 8GB DDR3 ECC", "RDIMM", "Working", 4},
			},
		},
		{
			name: "MacBook Pro 13\" 2015", typ: "Laptop", status: "Sold",
			condition: "Good", facility: "Techtoss", sub_location: "Rack A",
			notes: "Sold to a local student, picked up 3/2.",
			parts: []partSpec{
				{"Storage", "Apple 256GB SSD", "PCIe", "Working", 1},
			},
		},
		{
			name: "eMachines ET1831", typ: "Desktop", status: "Tossed",
			condition: "Poor", facility: "Almond Orchard", sub_location: "Floor",
			notes: "Blown capacitors, not economical to fix.",
			parts: []partSpec{},
		},
		{
			name: "Lenovo ThinkCentre M93p", typ: "Desktop", status: "Testing",
			condition: "Good", facility: "Bruce's Storage", sub_location: "Floor",
			notes: "Boots to BIOS, needs a Windows license.",
			parts: []partSpec{
				{"CPU", "Intel Core i5-4570", "4c/4t, 3.2GHz", "Working", 1},
				{"Storage", "Seagate 500GB HDD", "SATA 3.5\"", "Working", 1},
			},
		},
		{
			name: "ASUS ROG Strix Laptop", typ: "Laptop", status: "Repair",
			condition: "Fair", facility: "Techtoss", sub_location: "Rack B",
			notes: "Screen flickering — suspect ribbon cable.",
			parts: []partSpec{
				{"GPU", "NVIDIA GTX 1660 Ti (mobile)", "6GB GDDR6", "Working", 1},
			},
		},
		{
			name: "iMac 21.5\" 2013", typ: "All-in-One", status: "Intake",
			condition: "Unknown", facility: "Other", sub_location: "N/A",
			notes: "Donated, unknown condition — needs full triage.",
			parts: []partSpec{},
		},
		{
			name: "Custom Rack Server", typ: "Server", status: "Refurbished",
			condition: "Excellent", facility: "Connor's Garage", sub_location: "Floor",
			notes: "Built from harvested parts, burn-in tested 72h clean.",
			parts: []partSpec{
				{"Motherboard", "Supermicro X9DRi-LN4F+", "Dual socket", "Working", 1},
				{"RAM", "Samsung 16GB DDR3 ECC", "RDIMM", "Working", 8},
				{"Storage", "Intel S3500 480GB SSD", "SATA, enterprise", "Working", 2},
			},
		},
		{
			name: "Dell Precision Workstation", typ: "Workstation", status: "For Parts",
			condition: "Poor", facility: "Techtoss", sub_location: "Shelf 1",
			notes: "Cracked chassis, board untested — parting out.",
			parts: []partSpec{
				{"PSU", "Dell 685W", "ATX", "Untested", 1},
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
			m.Facility = ex.facility
			m.SubLocation = ex.sub_location
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

	// Loose inventory: a mix of categories, quantities, and locations so the
	// grouped inventory view and the pull-with-split flow both have real
	// variety to show off.
	type looseSpec struct {
		category, model, spec, condition, facility, sub_location string
		qty                                                      int
	}
	loose := []looseSpec{
		{"GPU", "NVIDIA GTX 1050 Ti", "4GB, harvested & tested", "Working", "Techtoss", "Shelf 1", 1},
		{"RAM", "Kingston 4GB DDR3-1600", "DIMM, pulled from scrap", "Working", "Techtoss", "Shelf 1", 6},
		{"Cooling", "120mm case fan", "3-pin", "Working", "Techtoss", "Rack A", 18},
		{"Peripheral", "SATA cable", "generic, various lengths", "Working", "Techtoss", "Shelf 1", 12},
		{"Furniture", "Office chair", "mesh back, adjustable height", "Working", "Bruce's Storage", "Floor", 4},
		{"Furniture", "Standing desk frame", "electric, no top", "Working", "Bruce's Storage", "Floor", 2},
		{"Peripheral", "Logitech K120 keyboard", "USB, tested", "Working", "Connor's Garage", "Floor", 8},
		{"Peripheral", "Dell 24\" Monitor P2419H", "1080p, IPS", "Working", "Connor's Garage", "Floor", 3},
		{"PSU", "Corsair CX450", "450W, 80+ Bronze", "Working", "Techtoss", "Rack B", 5},
		{"Storage", "WD Blue 1TB HDD", "SATA 3.5\"", "Working", "Almond Orchard", "Floor", 2},
		{"Motherboard", "Gigabyte B450M DS3H", "AM4, mATX", "Untested", "Techtoss", "Rack B", 1},
		{"Network", "TP-Link 8-port Gigabit switch", "unmanaged", "Working", "Other", "N/A", 3},
		{"Case", "Generic mid-tower ATX", "no PSU, some rust", "Working", "Almond Orchard", "Floor", 6},
		{"Optical", "LG DVD-RW drive", "SATA, internal", "Untested", "Techtoss", "Shelf 1", 4},
		{"CPU", "AMD Ryzen 5 2600", "6c/12t, 3.4GHz", "Working", "Bruce's Storage", "Floor", 1},
	}
	for _, ls := range loose {
		ls := ls
		store.CreatePart(func(p *Part) bool {
			p.MachineId = 0
			p.Category = ls.category
			p.Model = ls.model
			p.Spec = ls.spec
			p.Condition = ls.condition
			p.Facility = ls.facility
			p.SubLocation = ls.sub_location
			p.Quantity = ls.qty
			return true
		})
	}
}

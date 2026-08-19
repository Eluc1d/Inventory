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

		// The 38 below round the board out to 50 machines — roughly even
		// across all 7 statuses, a mix of types, and spread across every
		// facility — so status filtering, column sort, and Bubbles/Rows both
		// have enough volume to genuinely stress-test. None carry installed
		// parts; the 125-item loose inventory already covers part-level
		// testing, so these stay lean.

		// Intake (6)
		{name: "Dell OptiPlex 3070", typ: "Desktop", status: "Intake", condition: "Unknown", facility: "Techtoss", sub_location: "Shelf 1", notes: "Just arrived, untested.", parts: []partSpec{}},
		{name: "HP EliteBook 840 G3", typ: "Laptop", status: "Intake", condition: "Unknown", facility: "Bruce's Storage", sub_location: "Floor", notes: "Donated, charger included.", parts: []partSpec{}},
		{name: "Lenovo ThinkCentre M720q", typ: "Mini PC", status: "Intake", condition: "Unknown", facility: "Techtoss", sub_location: "Rack A", notes: "No accessories, unverified.", parts: []partSpec{}},
		{name: "HP ProLiant ML110", typ: "Server", status: "Intake", condition: "Unknown", facility: "Connor's Garage", sub_location: "Floor", notes: "Pulled from a closed office.", parts: []partSpec{}},
		{name: "Dell Inspiron AIO 3477", typ: "All-in-One", status: "Intake", condition: "Unknown", facility: "Other", sub_location: "N/A", notes: "Cracked stand, otherwise intact.", parts: []partSpec{}},
		{name: "Gateway DX4300", typ: "Desktop", status: "Intake", condition: "Unknown", facility: "Almond Orchard", sub_location: "Floor", notes: "Estate donation batch.", parts: []partSpec{}},

		// Testing (5)
		{name: "Dell Latitude E7450", typ: "Laptop", status: "Testing", condition: "Good", facility: "Techtoss", sub_location: "Rack B", notes: "Boots fine, testing battery health.", parts: []partSpec{}},
		{name: "HP ProDesk 600 G3", typ: "Desktop", status: "Testing", condition: "Good", facility: "Bruce's Storage", sub_location: "Floor", notes: "Passed POST, running memtest.", parts: []partSpec{}},
		{name: "Acer Aspire TC-895", typ: "Desktop", status: "Testing", condition: "Fair", facility: "Connor's Garage", sub_location: "Floor", notes: "Slow boot, checking drive health.", parts: []partSpec{}},
		{name: "Toshiba Satellite C55", typ: "Laptop", status: "Testing", condition: "Fair", facility: "Techtoss", sub_location: "Shelf 1", notes: "Screen flicker under testing.", parts: []partSpec{}},
		{name: "IBM System x3550", typ: "Server", status: "Testing", condition: "Good", facility: "Other", sub_location: "N/A", notes: "Burn-in testing in progress.", parts: []partSpec{}},

		// Repair (5)
		{name: "HP EliteDesk 705 G4", typ: "Desktop", status: "Repair", condition: "Fair", facility: "Techtoss", sub_location: "Rack A", notes: "No display output, checking GPU.", parts: []partSpec{}},
		{name: "Lenovo IdeaPad 320", typ: "Laptop", status: "Repair", condition: "Poor", facility: "Almond Orchard", sub_location: "Floor", notes: "Hinge broken, keyboard sticking.", parts: []partSpec{}},
		{name: "Dell PowerEdge R710", typ: "Server", status: "Repair", condition: "Fair", facility: "Connor's Garage", sub_location: "Floor", notes: "Fan error, replacing cooling.", parts: []partSpec{}},
		{name: "ASUS PN50", typ: "Mini PC", status: "Repair", condition: "Fair", facility: "Techtoss", sub_location: "Rack B", notes: "Won't power on consistently.", parts: []partSpec{}},
		{name: "HP Z420 Workstation", typ: "Workstation", status: "Repair", condition: "Fair", facility: "Bruce's Storage", sub_location: "Floor", notes: "Random shutdowns under load.", parts: []partSpec{}},

		// Refurbished (5)
		{name: "Dell Latitude 5490", typ: "Laptop", status: "Refurbished", condition: "Excellent", facility: "Techtoss", sub_location: "Rack A", notes: "New battery and SSD installed.", parts: []partSpec{}},
		{name: "Lenovo ThinkStation P330", typ: "Workstation", status: "Refurbished", condition: "Excellent", facility: "Techtoss", sub_location: "Shelf 1", notes: "Cleaned, retested, ready for sale.", parts: []partSpec{}},
		{name: "HP ProLiant DL380 G7", typ: "Server", status: "Refurbished", condition: "Good", facility: "Other", sub_location: "N/A", notes: "Reimaged, drives wiped and verified.", parts: []partSpec{}},
		{name: "Zotac ZBOX", typ: "Mini PC", status: "Refurbished", condition: "Excellent", facility: "Connor's Garage", sub_location: "Floor", notes: "New thermal paste, quiet fans.", parts: []partSpec{}},
		{name: "Dell Precision 5820", typ: "Workstation", status: "Refurbished", condition: "Excellent", facility: "Techtoss", sub_location: "Rack B", notes: "Upgraded RAM, retested stable.", parts: []partSpec{}},

		// For Parts (5)
		{name: "Compaq Presario CQ5210", typ: "Desktop", status: "For Parts", condition: "Poor", facility: "Almond Orchard", sub_location: "Floor", notes: "Cracked motherboard, stripping.", parts: []partSpec{}},
		{name: "Chromebook Acer C720", typ: "Laptop", status: "For Parts", condition: "Poor", facility: "Other", sub_location: "N/A", notes: "Locked bootloader, parting out.", parts: []partSpec{}},
		{name: "Supermicro SuperServer", typ: "Server", status: "For Parts", condition: "Poor", facility: "Techtoss", sub_location: "Shelf 1", notes: "Dead board, drives still good.", parts: []partSpec{}},
		{name: "HP Envy AIO 27", typ: "All-in-One", status: "For Parts", condition: "Poor", facility: "Bruce's Storage", sub_location: "Floor", notes: "Panel cracked, saving internals.", parts: []partSpec{}},
		{name: "Lenovo IdeaCentre AIO 3", typ: "All-in-One", status: "For Parts", condition: "Poor", facility: "Connor's Garage", sub_location: "Floor", notes: "Water damage, salvaging parts.", parts: []partSpec{}},

		// Sold (6)
		{name: "ThinkPad X1 Carbon", typ: "Laptop", status: "Sold", condition: "Excellent", facility: "Techtoss", sub_location: "Rack A", notes: "Sold at community sale.", parts: []partSpec{}},
		{name: "Dell XPS 13", typ: "Laptop", status: "Sold", condition: "Good", facility: "Techtoss", sub_location: "Rack B", notes: "Sold to a local nonprofit.", parts: []partSpec{}},
		{name: "MacBook Air 2017", typ: "Laptop", status: "Sold", condition: "Good", facility: "Bruce's Storage", sub_location: "Floor", notes: "Sold online, picked up 4/1.", parts: []partSpec{}},
		{name: "Custom Ryzen Build", typ: "Desktop", status: "Sold", condition: "Excellent", facility: "Techtoss", sub_location: "Shelf 1", notes: "Sold as a budget gaming PC.", parts: []partSpec{}},
		{name: "Dell OptiPlex 5060", typ: "Desktop", status: "Sold", condition: "Good", facility: "Connor's Garage", sub_location: "Floor", notes: "Sold to a small business.", parts: []partSpec{}},
		{name: "HP Pavilion Desktop", typ: "Desktop", status: "Sold", condition: "Good", facility: "Almond Orchard", sub_location: "Floor", notes: "Sold, picked up same week.", parts: []partSpec{}},

		// Tossed (6)
		{name: "HP ProBook 450 G5", typ: "Laptop", status: "Tossed", condition: "Poor", facility: "Other", sub_location: "N/A", notes: "Liquid damage beyond repair.", parts: []partSpec{}},
		{name: "Acer Aspire 5", typ: "Laptop", status: "Tossed", condition: "Poor", facility: "Techtoss", sub_location: "Rack A", notes: "Motherboard shorted, not economical.", parts: []partSpec{}},
		{name: "Dell Inspiron 3880", typ: "Desktop", status: "Tossed", condition: "Poor", facility: "Techtoss", sub_location: "Rack B", notes: "Multiple failed components.", parts: []partSpec{}},
		{name: "Intel NUC NUC7i5BNH", typ: "Mini PC", status: "Tossed", condition: "Poor", facility: "Bruce's Storage", sub_location: "Floor", notes: "Bad solder joint, unrepairable.", parts: []partSpec{}},
		{name: "Custom Gaming Tower II", typ: "Desktop", status: "Tossed", condition: "Poor", facility: "Connor's Garage", sub_location: "Floor", notes: "Fire damage from PSU failure.", parts: []partSpec{}},
		{name: "Toshiba Satellite Pro", typ: "Laptop", status: "Tossed", condition: "Poor", facility: "Almond Orchard", sub_location: "Floor", notes: "Screen and board both dead.", parts: []partSpec{}},
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

	// Loose inventory: 8-10 items in every category (13 canonical categories
	// plus the free-text "Furniture" bucket), spread across facilities and
	// with varied quantities/conditions, so the category filter, location
	// grouping, sort, and pull-with-split flows all have enough volume to
	// genuinely stress-test rather than just demo.
	type looseSpec struct {
		category, model, spec, condition, facility, sub_location string
		qty                                                      int
	}
	loose := []looseSpec{
		// CPU
		{"CPU", "AMD Ryzen 5 2600", "6c/12t, 3.4GHz", "Working", "Bruce's Storage", "Floor", 1},
		{"CPU", "Intel Core i5-8500", "6c/6t, 3.0GHz", "Working", "Techtoss", "Rack A", 1},
		{"CPU", "Intel Core i7-9700K", "8c/8t, 3.6GHz", "Working", "Techtoss", "Rack B", 1},
		{"CPU", "AMD Ryzen 7 5800X", "8c/16t, 3.8GHz", "Working", "Bruce's Storage", "Floor", 1},
		{"CPU", "Intel Core i3-10100", "4c/8t, 3.6GHz", "Untested", "Connor's Garage", "Floor", 2},
		{"CPU", "AMD Ryzen 5 2400G", "4c/8t, Vega 11 iGPU", "Working", "Techtoss", "Shelf 1", 1},
		{"CPU", "Intel Xeon E5-2670", "8c/16t, 2.6GHz", "Untested", "Almond Orchard", "Floor", 2},
		{"CPU", "Intel Core i9-9900K", "8c/16t, 3.6GHz", "Working", "Other", "N/A", 1},
		{"CPU", "Intel Pentium G4560", "2c/4t, 3.5GHz", "Working", "Techtoss", "Rack A", 3},

		// GPU
		{"GPU", "NVIDIA GTX 1050 Ti", "4GB, harvested & tested", "Working", "Techtoss", "Shelf 1", 1},
		{"GPU", "NVIDIA GTX 1660 Super", "6GB GDDR6", "Working", "Techtoss", "Rack B", 1},
		{"GPU", "NVIDIA RTX 2060", "6GB GDDR6", "Faulty", "Connor's Garage", "Floor", 1},
		{"GPU", "NVIDIA RTX 3060", "12GB GDDR6", "Working", "Bruce's Storage", "Floor", 1},
		{"GPU", "AMD Radeon RX 570", "4GB GDDR5", "Working", "Techtoss", "Shelf 1", 1},
		{"GPU", "AMD Radeon RX 580", "8GB GDDR5", "Untested", "Almond Orchard", "Floor", 1},
		{"GPU", "NVIDIA GT 710", "2GB DDR3", "Working", "Other", "N/A", 2},
		{"GPU", "NVIDIA Quadro P400", "2GB GDDR5", "Working", "Techtoss", "Rack A", 1},
		{"GPU", "NVIDIA GTX 1080", "8GB GDDR5X", "Faulty", "Connor's Garage", "Floor", 1},

		// RAM
		{"RAM", "Kingston 4GB DDR3-1600", "DIMM, pulled from scrap", "Working", "Techtoss", "Shelf 1", 6},
		{"RAM", "Crucial 8GB DDR4-2666", "SODIMM", "Working", "Techtoss", "Rack A", 4},
		{"RAM", "Corsair Vengeance 16GB DDR4-3200", "2x8GB", "Working", "Techtoss", "Rack B", 2},
		{"RAM", "G.Skill Ripjaws 32GB DDR4-3600", "2x16GB", "Working", "Bruce's Storage", "Floor", 2},
		{"RAM", "Samsung 8GB DDR3 ECC", "RDIMM", "Working", "Connor's Garage", "Floor", 6},
		{"RAM", "SK Hynix 4GB DDR4-2400", "DIMM", "Untested", "Almond Orchard", "Floor", 5},
		{"RAM", "Crucial 16GB DDR4-2400", "DIMM", "Working", "Other", "N/A", 3},
		{"RAM", "Kingston HyperX 8GB DDR4-3000", "DIMM", "Working", "Techtoss", "Shelf 1", 2},
		{"RAM", "Corsair Dominator 32GB DDR4-3200", "2x16GB", "Working", "Techtoss", "Rack A", 1},

		// Storage
		{"Storage", "WD Blue 1TB HDD", "SATA 3.5\"", "Working", "Almond Orchard", "Floor", 2},
		{"Storage", "WD Blue 256GB SSD", "SATA 2.5\"", "Working", "Techtoss", "Rack A", 3},
		{"Storage", "Samsung 970 EVO 500GB", "NVMe", "Working", "Techtoss", "Rack B", 2},
		{"Storage", "Seagate Barracuda 2TB HDD", "SATA 3.5\"", "Working", "Bruce's Storage", "Floor", 1},
		{"Storage", "Crucial MX500 1TB SSD", "SATA 2.5\"", "Working", "Connor's Garage", "Floor", 2},
		{"Storage", "Intel S3500 480GB SSD", "SATA, enterprise", "Working", "Almond Orchard", "Floor", 1},
		{"Storage", "Toshiba 500GB HDD", "SATA 2.5\"", "Untested", "Other", "N/A", 3},
		{"Storage", "SanDisk 128GB SSD", "SATA 2.5\"", "Working", "Techtoss", "Shelf 1", 4},
		{"Storage", "Kingston A400 240GB SSD", "SATA 2.5\"", "Working", "Techtoss", "Rack A", 2},

		// Motherboard
		{"Motherboard", "Gigabyte B450M DS3H", "AM4, mATX", "Untested", "Techtoss", "Rack B", 1},
		{"Motherboard", "ASUS Prime B450M-A", "AM4, mATX", "Untested", "Techtoss", "Rack B", 1},
		{"Motherboard", "MSI B550 Tomahawk", "AM4, ATX", "Working", "Bruce's Storage", "Floor", 1},
		{"Motherboard", "ASRock H310M-HDV", "LGA1151, mATX", "Working", "Connor's Garage", "Floor", 1},
		{"Motherboard", "ASUS ROG Strix Z390-E", "LGA1151, ATX", "Working", "Almond Orchard", "Floor", 1},
		{"Motherboard", "Biostar A320MH", "AM4, mATX", "Untested", "Other", "N/A", 2},
		{"Motherboard", "Gigabyte Z390 UD", "LGA1151, ATX", "Working", "Techtoss", "Shelf 1", 1},
		{"Motherboard", "MSI H610M-E", "LGA1700, mATX", "Working", "Techtoss", "Rack A", 1},
		{"Motherboard", "Supermicro X8DTL-i", "dual LGA1366, server", "Untested", "Techtoss", "Rack B", 1},

		// PSU
		{"PSU", "Corsair CX450", "450W, 80+ Bronze", "Working", "Techtoss", "Rack B", 5},
		{"PSU", "Dell 200W", "internal SFF", "Working", "Techtoss", "Rack A", 2},
		{"PSU", "EVGA 600W BQ", "80+ Bronze, semi-modular", "Working", "Bruce's Storage", "Floor", 1},
		{"PSU", "Seasonic Focus 750W", "80+ Gold", "Working", "Connor's Garage", "Floor", 1},
		{"PSU", "Cooler Master MWE 550W", "80+ Bronze", "Working", "Almond Orchard", "Floor", 2},
		{"PSU", "Dell 685W", "ATX, server/workstation", "Untested", "Other", "N/A", 1},
		{"PSU", "Thermaltake Smart 500W", "80+ White", "Working", "Techtoss", "Shelf 1", 1},
		{"PSU", "Antec VP450P", "450W", "Untested", "Techtoss", "Rack B", 1},
		{"PSU", "be quiet! Pure Power 11 600W", "80+ Gold", "Working", "Techtoss", "Rack A", 1},

		// Cooling
		{"Cooling", "120mm case fan", "3-pin", "Working", "Techtoss", "Rack A", 18},
		{"Cooling", "Cooler Master Hyper 212", "tower air cooler", "Working", "Techtoss", "Rack B", 2},
		{"Cooling", "Noctua NF-A12x25", "120mm premium fan", "Working", "Bruce's Storage", "Floor", 4},
		{"Cooling", "Corsair H100i", "240mm AIO liquid cooler", "Untested", "Connor's Garage", "Floor", 1},
		{"Cooling", "Arctic Freezer 34", "tower air cooler", "Working", "Almond Orchard", "Floor", 2},
		{"Cooling", "Deepcool Gammaxx 400", "tower air cooler", "Working", "Other", "N/A", 1},
		{"Cooling", "140mm case fan", "3-pin", "Working", "Techtoss", "Shelf 1", 10},
		{"Cooling", "Intel stock cooler", "LGA115x", "Working", "Techtoss", "Rack A", 5},
		{"Cooling", "Arctic MX-4 thermal paste", "4g tube", "Working", "Techtoss", "Rack B", 6},

		// Case
		{"Case", "Generic mid-tower ATX", "no PSU, some rust", "Working", "Almond Orchard", "Floor", 6},
		{"Case", "Corsair Carbide 4000D", "mid-tower ATX", "Working", "Techtoss", "Rack A", 1},
		{"Case", "NZXT H510", "mid-tower ATX", "Working", "Bruce's Storage", "Floor", 1},
		{"Case", "Fractal Design Meshify C", "mid-tower ATX", "Working", "Connor's Garage", "Floor", 1},
		{"Case", "Cooler Master MasterBox Q300L", "mATX", "Untested", "Almond Orchard", "Floor", 2},
		{"Case", "Antec P101", "full-tower ATX", "Working", "Other", "N/A", 1},
		{"Case", "Thermaltake Versa H26", "mid-tower ATX", "Working", "Techtoss", "Shelf 1", 1},
		{"Case", "Generic mini-ITX case", "SFF", "Untested", "Techtoss", "Rack B", 3},
		{"Case", "Rosewill FBM-01", "mATX", "Working", "Techtoss", "Rack A", 1},

		// Cabling
		{"Cabling", "SATA data cable", "18in", "Working", "Techtoss", "Shelf 1", 12},
		{"Cabling", "24-pin ATX power extension", "300mm, sleeved", "Working", "Techtoss", "Shelf 1", 5},
		{"Cabling", "Molex to SATA power adapter", "generic", "Working", "Bruce's Storage", "Floor", 8},
		{"Cabling", "8-pin PCIe power cable", "generic", "Working", "Connor's Garage", "Floor", 6},
		{"Cabling", "USB 3.0 front-panel header cable", "generic", "Working", "Almond Orchard", "Floor", 4},
		{"Cabling", "Front-panel audio cable", "HD Audio", "Working", "Other", "N/A", 5},
		{"Cabling", "HDMI cable", "6ft", "Working", "Techtoss", "Rack A", 7},
		{"Cabling", "Ethernet cable", "Cat6, 3ft", "Working", "Techtoss", "Rack B", 15},
		{"Cabling", "4-pin fan splitter cable", "generic", "Working", "Techtoss", "Shelf 1", 9},

		// Network
		{"Network", "TP-Link 8-port Gigabit switch", "unmanaged", "Working", "Other", "N/A", 3},
		{"Network", "Netgear GS108 switch", "8-port, unmanaged", "Working", "Techtoss", "Rack A", 2},
		{"Network", "TP-Link Archer C6 router", "AC1200", "Working", "Bruce's Storage", "Floor", 1},
		{"Network", "Intel Gigabit CT PCIe NIC", "single port", "Working", "Connor's Garage", "Floor", 3},
		{"Network", "Ubiquiti UniFi AP-AC-Lite", "802.11ac AP", "Untested", "Almond Orchard", "Floor", 1},
		{"Network", "Netgear Nighthawk R7000", "AC1900 router", "Working", "Other", "N/A", 1},
		{"Network", "D-Link 5-port switch", "unmanaged", "Working", "Techtoss", "Shelf 1", 2},
		{"Network", "Realtek USB-to-Ethernet adapter", "USB 3.0", "Working", "Techtoss", "Rack B", 4},
		{"Network", "Cisco SG110-16 switch", "unmanaged", "Untested", "Techtoss", "Rack A", 1},

		// Optical
		{"Optical", "LG DVD-RW drive", "SATA, internal", "Untested", "Techtoss", "Shelf 1", 4},
		{"Optical", "Samsung Blu-ray combo drive", "SATA, internal", "Untested", "Techtoss", "Rack A", 2},
		{"Optical", "ASUS external DVD writer", "USB", "Working", "Bruce's Storage", "Floor", 1},
		{"Optical", "Sony CD-ROM drive", "IDE", "Untested", "Connor's Garage", "Floor", 1},
		{"Optical", "LiteOn DVD-ROM", "SATA, internal", "Working", "Almond Orchard", "Floor", 2},
		{"Optical", "Pioneer BDR internal Blu-ray burner", "SATA", "Working", "Other", "N/A", 1},
		{"Optical", "HP slim external DVD writer", "USB", "Working", "Techtoss", "Shelf 1", 3},
		{"Optical", "Generic CD/DVD combo drive", "IDE", "Untested", "Techtoss", "Rack B", 2},
		{"Optical", "External Blu-ray burner", "USB 3.0", "Working", "Techtoss", "Rack A", 1},

		// Peripheral
		{"Peripheral", "Logitech K120 keyboard", "USB, tested", "Working", "Connor's Garage", "Floor", 8},
		{"Peripheral", "Dell 24\" Monitor P2419H", "1080p, IPS", "Working", "Connor's Garage", "Floor", 3},
		{"Peripheral", "Logitech MX Master 3 mouse", "wireless", "Working", "Techtoss", "Rack A", 2},
		{"Peripheral", "HP wired keyboard", "generic USB", "Working", "Bruce's Storage", "Floor", 5},
		{"Peripheral", "Dell wired optical mouse", "generic USB", "Working", "Connor's Garage", "Floor", 6},
		{"Peripheral", "Acer 22-inch monitor", "1080p, VA", "Untested", "Almond Orchard", "Floor", 2},
		{"Peripheral", "Logitech C920 webcam", "1080p", "Working", "Other", "N/A", 3},
		{"Peripheral", "Generic USB keyboard", "no branding", "Working", "Techtoss", "Shelf 1", 4},
		{"Peripheral", "Generic wireless mouse", "2.4GHz", "Working", "Techtoss", "Rack B", 5},

		// Other
		{"Other", "Screws & standoffs assortment kit", "mixed sizes", "Working", "Techtoss", "Rack A", 6},
		{"Other", "Thermal pad assortment", "various thickness", "Working", "Techtoss", "Rack B", 10},
		{"Other", "VESA mount bracket", "75/100mm", "Working", "Bruce's Storage", "Floor", 4},
		{"Other", "Generic laptop battery", "untested capacity", "Untested", "Connor's Garage", "Floor", 3},
		{"Other", "Power strip", "6-outlet, surge protected", "Working", "Almond Orchard", "Floor", 3},
		{"Other", "Cable ties assortment", "100-pack", "Working", "Other", "N/A", 8},
		{"Other", "Anti-static wrist strap", "generic", "Working", "Techtoss", "Shelf 1", 2},
		{"Other", "USB flash drive 16GB", "generic", "Working", "Techtoss", "Rack A", 5},
		{"Other", "External HDD enclosure", "2.5\", USB 3.0", "Working", "Techtoss", "Rack B", 4},

		// Furniture (free-text category — not one of PartCategories, and
		// deliberately left that way to keep exercising the "Uncategorized"/
		// unknown-category path in the grouped and filtered views)
		{"Furniture", "Office chair", "mesh back, adjustable height", "Working", "Bruce's Storage", "Floor", 4},
		{"Furniture", "Standing desk frame", "electric, no top", "Working", "Bruce's Storage", "Floor", 2},
		{"Furniture", "Filing cabinet", "2-drawer, metal", "Working", "Connor's Garage", "Floor", 3},
		{"Furniture", "Bookshelf", "5-tier, metal", "Working", "Connor's Garage", "Floor", 2},
		{"Furniture", "Rolling utility cart", "3-tier", "Working", "Almond Orchard", "Floor", 2},
		{"Furniture", "Workbench", "steel frame, wood top", "Working", "Other", "N/A", 1},
		{"Furniture", "Task lamp", "LED, adjustable arm", "Working", "Techtoss", "Shelf 1", 6},
		{"Furniture", "Whiteboard", "4x3 ft, magnetic", "Working", "Techtoss", "Rack A", 2},
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

// seedSampleTickets adds a handful of realistic client-request tickets, with
// varied statuses/deadlines/item kinds, for exercising the Tickets page.
// Unlike seedIfEmpty, this is purely additive (CreateTicket + item inserts,
// never a delete) and doesn't check whether tickets already exist first —
// it's meant to be run on demand via -seed-tickets, including against a
// database that already has real tickets in it.
func seedSampleTickets(store Store) {
	log.Printf("Seeding sample tickets")

	type itemSpec struct {
		kind, description string
		qty               int
	}
	tickets := []struct {
		title, client, deadline, status, notes string
		items                                  []itemSpec
	}{
		{
			title: "Refurb 3 laptops for office move", client: "Meridian Nonprofit Partners",
			deadline: "2026-09-05", status: "In Progress",
			notes: "Client needs Windows 11 Pro preinstalled on all three.",
			items: []itemSpec{
				{"Machine", "3x business laptops, 16GB RAM minimum", 3},
				{"Part", "USB-C docking stations", 3},
				{"Custom", "Data migration from their old machines", 1},
			},
		},
		{
			title: "Custom gaming build for donor's grandson", client: "Diane R.",
			deadline: "2026-08-25", status: "Open",
			notes: "Donor specifically requested a photo of the finished build before pickup.",
			items: []itemSpec{
				{"Machine", "1x custom gaming tower, RTX-class GPU", 1},
				{"Part", "RGB case fans", 4},
				{"Custom", "Cable management + benchmark video for the donor", 1},
			},
		},
		{
			title: "Bulk desktop order — Almond Orchard volunteers", client: "Almond Orchard Community Center",
			deadline: "2026-08-01", status: "Fulfilled",
			notes: "Delivered and set up on-site; all five tested working before handoff.",
			items: []itemSpec{
				{"Machine", "5x refurbished desktops, basic office use", 5},
				{"Custom", "Delivery + on-site setup for 5 volunteer stations", 1},
			},
		},
		{
			title: "Replacement PSU + diagnostics", client: "Marcus T.",
			deadline: "", status: "Cancelled",
			notes: "Client found a cheaper option elsewhere — cancelled before work started.",
			items: []itemSpec{
				{"Part", "650W 80+ Gold PSU", 1},
				{"Custom", "Full diagnostic run before return", 1},
			},
		},
		{
			title: "School lab refresh — 8 workstations", client: "Lincoln Elementary STEM Lab",
			deadline: "2026-10-01", status: "Open",
			notes: "Grant-funded order — an invoice is needed for their reimbursement paperwork.",
			items: []itemSpec{
				{"Machine", "8x mini PCs for classroom lab", 8},
				{"Part", "USB keyboard/mouse combos", 8},
				{"Part", "8-port network switch", 1},
				{"Custom", "Asset tagging + inventory list for school records", 1},
			},
		},
	}

	for _, ts := range tickets {
		ts := ts
		t, errStr := store.CreateTicket(func(t *Ticket) bool {
			t.Title = ts.title
			t.Client = ts.client
			t.Deadline = ts.deadline
			t.Status = ts.status
			t.Notes = ts.notes
			return true
		})
		if t == nil {
			log.Printf("seed ticket failed: %s", errStr)
			continue
		}
		items := make([]*TicketItem, 0, len(ts.items))
		for _, is := range ts.items {
			items = append(items, &TicketItem{Kind: is.kind, Description: is.description, Quantity: is.qty})
		}
		if ok, errStr := store.ReplaceTicketItems(t.Id, items); !ok {
			log.Printf("seed ticket items failed for %s: %s", t.Number, errStr)
		}
	}
}

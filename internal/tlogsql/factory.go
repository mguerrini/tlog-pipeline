package tlogsql

// AllGenerators devuelve los 10 generators SQL en el orden canónico.
func AllGenerators() []Generator {
	return []Generator{
		ReceptionGenerator{},            // doc 00
		ReturnGenerator{},               // doc 01
		TransferGenerator{},             // doc 02
		AdjustmentVerbrauchGenerator{},  // doc 31
		AdjustmentInventurGenerator{},   // doc 32
		CountVerbrauchGenerator{},       // doc 41
		CountInventurGenerator{},        // doc 42
		FiscalDocFCGenerator{},          // doc 05
		FiscalDocNCGenerator{},          // doc 06
		CierreGenerator{},               // doc 07
	}
}

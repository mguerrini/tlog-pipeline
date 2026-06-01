package tlogsql

// AllGenerators devuelve los 10 generators SQL en el orden canónico.
func AllGenerators() []Generator {
	return []Generator{
		ReceptionGenerator{},            // doc 00
		ReturnGenerator{},               // doc 01
		TransferGenerator{},             // doc 02
		AdjustmentVerbrauchGenerator{},  // doc 03
		AdjustmentInventurGenerator{},   // doc 03 (secuencia compartida con Verbrauch)
		CountVerbrauchGenerator{},       // doc 04
		CountInventurGenerator{},        // doc 04 (secuencia compartida con Verbrauch)
		FiscalDocFCGenerator{},          // doc 05
		FiscalDocNCGenerator{},          // doc 06
		CierreGenerator{},               // doc 07
	}
}

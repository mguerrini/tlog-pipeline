package tlogsql

// AllGenerators devuelve los 8 generators SQL en el orden canónico NNNN.
// El orden coincide con factory.AllGenerators() del flujo in-memory.
func AllGenerators() []Generator {
	return []Generator{
		ReceptionGenerator{},    // 0001
		ReturnGenerator{},       // 0002
		TransferGenerator{},     // 0003
		AdjustmentGenerator{},   // 0004
		CountGenerator{},        // 0005
		FiscalDocFCGenerator{},  // 0006
		FiscalDocNCGenerator{},  // 0007
		CierreGenerator{},       // 0008
	}
}

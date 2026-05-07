package naming

// TLOGType representa el tipo de TLOG OCPRA generado por el pipeline.
type TLOGType string

const (
	TLOGReception   TLOGType = "Reception"
	TLOGReturn      TLOGType = "Return"
	TLOGTransfer    TLOGType = "Transfer"
	TLOGAdjustment  TLOGType = "Adjustment"
	TLOGCount       TLOGType = "Count"
	TLOGFiscalDocFC TLOGType = "FiscalDocFC"
	TLOGFiscalDocNC TLOGType = "FiscalDocNC"
	TLOGCierre      TLOGType = "Cierre"
)

// TLOGOrder define el orden canónico para el sufijo NNNN del nombre del XML.
// La posición en este slice (1-based) determina el NNNN del archivo generado.
var TLOGOrder = []TLOGType{
	TLOGReception,   // 0001
	TLOGReturn,      // 0002
	TLOGTransfer,    // 0003
	TLOGAdjustment,  // 0004
	TLOGCount,       // 0005
	TLOGFiscalDocFC, // 0006
	TLOGFiscalDocNC, // 0007
	TLOGCierre,      // 0008
}

// indexOf devuelve la posición 1-based de t dentro de TLOGOrder. 0 si no está.
func indexOf(t TLOGType) int {
	for i, v := range TLOGOrder {
		if v == t {
			return i + 1
		}
	}
	return 0
}

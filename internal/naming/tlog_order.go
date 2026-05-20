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

// TLOGOrder define el orden canónico de generación de los TLOG.
var TLOGOrder = []TLOGType{
	TLOGReception,
	TLOGReturn,
	TLOGTransfer,
	TLOGAdjustment,
	TLOGCount,
	TLOGFiscalDocFC,
	TLOGFiscalDocNC,
	TLOGCierre,
}

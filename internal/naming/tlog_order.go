package naming

// TLOGType representa el tipo de TLOG OCPRA generado por el pipeline.
type TLOGType string

const (
	TLOGReception           TLOGType = "Reception"
	TLOGReturn              TLOGType = "Return"
	TLOGTransfer            TLOGType = "Transfer"
	TLOGAdjustmentVerbrauch TLOGType = "AdjustmentVerbrauch"
	TLOGAdjustmentInventur  TLOGType = "AdjustmentInventur"
	TLOGCountVerbrauch      TLOGType = "CountVerbrauch"
	TLOGCountInventur       TLOGType = "CountInventur"
	TLOGFiscalDocFC         TLOGType = "FiscalDocFC"
	TLOGFiscalDocNC         TLOGType = "FiscalDocNC"
	TLOGCierre              TLOGType = "Cierre"
)

// TLOGOrder define el orden canónico de generación de los TLOG.
var TLOGOrder = []TLOGType{
	TLOGReception,
	TLOGReturn,
	TLOGTransfer,
	TLOGAdjustmentVerbrauch,
	TLOGAdjustmentInventur,
	TLOGCountVerbrauch,
	TLOGCountInventur,
	TLOGFiscalDocFC,
	TLOGFiscalDocNC,
	TLOGCierre,
}

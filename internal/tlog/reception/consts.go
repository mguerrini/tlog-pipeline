package reception

// Constantes y valores fijos del mapeo InventoryReception.
// Fuente: MAPEO_TLOG_INVENTORY_RECEPTION.md
const (
	WorkstationID                = "0"
	Period                       = "0"
	Subperiod                    = "0"
	DocumentTypeCode             = "InventoryReception"
	InventoryControlDocumentState = "4" // LFS_STATUS=42 -> 4 (cerrado)
	FiscalReceiptFlag            = "true"
	DestinationLocation          = "DEP1_OS"
	SourceLocation               = "DEP1_OS"
	ItemBrand                    = "0"
	UnitSalesAmount              = "0.0000"
	SalesTotalAmount             = "0.0000"
	Stock                        = "0.0000"
	DailyAverageSales            = "0.0000"
	SuggestedPurchaseOrder       = "0.0000"
	LineState                    = "1" // detalle.InventoryControlDocumentState
)

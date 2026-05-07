package cierre

import (
	"fmt"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
	"github.com/opessa/tlog-pipeline/internal/tlog/unknown"
)

const (
	WorkstationID       = "0"
	Period              = "0"
	Subperiod           = "0"
	TypeCode            = "BusinessEOS"
	TypeID              = "63"
	StockSeqNumber      = "1"
	LocationCode        = "DEP1_OS" // [UNKNOWN] - ver MAPEO_TLOG_CIERRE_REAL.md
	RevenueCenter       = "RCD"     // [UNKNOWN] - ver MAPEO_TLOG_CIERRE_REAL.md
	ItemInventoryState  = "OnSale"
)

// Generator implementa tlog.Generator para BusinessEOS (Cierre de día).
type Generator struct{}

func (Generator) Type() naming.TLOGType { return naming.TLOGCierre }

func (Generator) Generate(s *db.Store, h *common.HeaderCtx, kstID string) (*tlog.GenerateResult, error) {
	dtTable := s.Tables["DAILYTOTALS"]
	if dtTable == nil {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	// Filtrar por KST_ID. La tabla no tiene AKTIV ni DAY_DATE en el nombre,
	// el día ya está implícito (la DB se carga con los CSVs de un único día).
	var items []db.Row
	for _, row := range dtTable.Rows {
		if row["KST_ID"] != kstID {
			continue
		}
		items = append(items, row)
	}
	if len(items) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	kst := s.Kostst[kstID]
	retailID := common.FormatRetailStoreID(kst["KST_CODE"])
	seqNum := common.BuildSequenceNumber11(retailID, 1)

	x := common.NewXMLBuilder()
	writeCierreHeader(x, h, retailID, seqNum)

	for i, row := range items {
		writeCierreItem(x, s, row, seqNum, i+1)
	}

	x.Close() // ItemList
	x.Close() // Transaction

	return &tlog.GenerateResult{
		XMLContent: x.String(),
		NumDocs:    1,
		NumLines:   len(items),
	}, nil
}

func writeCierreHeader(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string) {
	x.Open("Transaction")
	x.Element("RETAILSTOREID", retailID)
	x.Element("WORKSTATIONID", WorkstationID)
	x.Element("SEQUENCENUMBER", seqNum)
	x.Element("BUSINESSDAYDATE", h.FormatBusinessDayDate())
	x.Element("BEGINDATETIME", h.FormatBeginDateTime())
	x.Element("ENDDATETIME", h.FormatEndDateTime())
	x.Element("OPERATORID", h.OperatorID)
	x.Element("PERIODO", Period)
	x.Element("SUBPERIOD", Subperiod)
	x.Element("PERIODCODE", "0")
	x.Element("SUBPERIODCODE", "0")
	x.Element("TYPECODE", TypeCode)
	x.Element("TYPEID", TypeID)
	x.Open("ItemList")
}

func writeCierreItem(x *common.XMLBuilder, s *db.Store, row db.Row, seqNum string, itemSeq int) {
	artRow := s.Artikel[row["ART_ID"]]
	artNummer := artRow["ART_NUMMER"]
	if artNummer == "" {
		artNummer = row["ART_ID"] // fallback
	}

	sohBeg, _ := db.AsFloat(row["DAY_SOHBEG"])
	qtySold, _ := db.AsFloat(row["DAY_QTYSOLD"])
	qtyPurch, _ := db.AsFloat(row["DAY_QTYPURCH"])
	qtyTrsfIn, _ := db.AsFloat(row["DAY_QTYTRSFIN"])
	qtyTrsfOut, _ := db.AsFloat(row["DAY_QTYTRSFOUT"])
	sohEnd, _ := db.AsFloat(row["DAY_SOHEND"])

	// [UNKNOWN] campos — marcados con la convención del proyecto
	returnUnitCount := unknown.Emit("0.0000",
		"No hay campo directo en DAILYTOTALS para devoluciones de venta. Validar con negocio")
	returnToVentorCount := unknown.Emit("0.0000",
		"Sin campo directo. Posible SUM(LFP_QTYRTV). Confirmar typo VENTOR vs VENDOR")
	adjustInCount := unknown.Emit("0.0000",
		"DAY_QTYUSAGE positivo o DAY_QTYINV. Separación positivo/negativo a definir con negocio")
	adjustOutCount := unknown.Emit("0.0000",
		"DAY_QTYUSAGE negativo o DAY_QTYEXPENSE. A definir con negocio")

	x.Open("Item")
	x.Element("STOCK_SEQ_NUMBER", StockSeqNumber)
	x.Element("LOCATION_CODE", LocationCode)
	x.Element("REVENUE_CENTER", RevenueCenter)
	x.Element("ITEM_INVENTORY_STATE", ItemInventoryState)
	x.Element("ITEM_SEQ_NUMBER", fmt.Sprintf("%d", itemSeq))
	x.Element("ITEM_CODE", artNummer)
	x.Element("BEGIN_UNIT_COUNT", common.FormatDecimal4(sohBeg))
	x.Element("GROSS_SALE_UNIT_COUNT", common.FormatDecimal4(qtySold))
	x.Element("RETURN_UNIT_COUNT", returnUnitCount)
	x.Element("RECEIVED_UNIT_COUNT", common.FormatDecimal4(qtyPurch))
	x.Element("RETURN_TO_VENTOR_UNIT_COUNT", returnToVentorCount)
	x.Element("TRANSFERIN_UNIT_COUNT", common.FormatDecimal4(qtyTrsfIn))
	x.Element("TRANSFEROUT_UNIT_COUNT", common.FormatDecimal4(qtyTrsfOut))
	x.Element("ADJUSTMENTIN_UNIT_COUNT", adjustInCount)
	x.Element("ADJUSTMENTOUT_UNIT_COUNT", adjustOutCount)
	x.Element("CURRENT_UNIT_COUNT", common.FormatDecimal4(sohEnd))
	x.Close()
}

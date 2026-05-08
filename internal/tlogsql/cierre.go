package tlogsql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
	"github.com/opessa/tlog-pipeline/internal/tlog/unknown"
)

const (
	cierreWorkstationID      = "0"
	cierrePeriod             = "0"
	cierreSubperiod          = "0"
	cierreTypeCode           = "BusinessEOS"
	cierreTypeID             = "63"
	cierreStockSeqNumber     = "1"
	cierreLocationCode       = "DEP1_OS"
	cierreRevenueCenter      = "RCD"
	cierreItemInventoryState = "OnSale"
)

// CierreGenerator implementa TLOG_BUSINESS_EOS (cierre) con SQL.
//
// Lee DAILYTOTALS1 filtrado por KST_ID. La DB se carga con los CSVs de un
// único día, así que no hace falta filtrar por DAY_DATE.
type CierreGenerator struct{}

func (CierreGenerator) Type() naming.TLOGType { return naming.TLOGCierre }

func (CierreGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string) (*tlog.GenerateResult, error) {
	const itemsSQL = `
SELECT dt.KST_ID, dt.ART_ID, dt.DAY_DATE,
       dt.DAY_SOHBEG, dt.DAY_SOHEND,
       dt.DAY_QTYPURCH, dt.DAY_QTYTRSFIN, dt.DAY_QTYTRSFOUT,
       dt.DAY_QTYUSAGE, dt.DAY_QTYSOLD, dt.DAY_QTYINV,
       art.ART_NUMMER, art.ART_NAME
FROM DAILYTOTALS1 dt
LEFT JOIN ARTIKEL art ON art.ART_ID = dt.ART_ID
WHERE dt.KST_ID = ?
ORDER BY dt.ART_ID`
	items, err := queryRows(ctx, conn, itemsSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("cierre items: %w", err)
	}
	if len(items) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	kst, err := fetchKostst(ctx, conn, kstID)
	if err != nil {
		return nil, err
	}
	retailID := common.FormatRetailStoreID(kst["KST_CODE"])
	seqNum := common.BuildSequenceNumber11(retailID, 1)

	x := common.NewXMLBuilder()
	writeCierreHeader(x, h, retailID, seqNum)
	for i, row := range items {
		writeCierreItem(x, row, i+1)
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
	x.Element("WORKSTATIONID", cierreWorkstationID)
	x.Element("SEQUENCENUMBER", seqNum)
	x.Element("BUSINESSDAYDATE", h.FormatBusinessDayDate())
	x.Element("BEGINDATETIME", h.FormatBeginDateTime())
	x.Element("ENDDATETIME", h.FormatEndDateTime())
	x.Element("OPERATORID", h.OperatorID)
	x.Element("PERIODO", cierrePeriod)
	x.Element("SUBPERIOD", cierreSubperiod)
	x.Element("PERIODCODE", "0")
	x.Element("SUBPERIODCODE", "0")
	x.Element("TYPECODE", cierreTypeCode)
	x.Element("TYPEID", cierreTypeID)
	x.Open("ItemList")
}

func writeCierreItem(x *common.XMLBuilder, row map[string]string, itemSeq int) {
	artNummer := row["ART_NUMMER"]
	if artNummer == "" {
		artNummer = row["ART_ID"] // fallback
	}

	sohBeg, _ := db.AsFloat(row["DAY_SOHBEG"])
	qtySold, _ := db.AsFloat(row["DAY_QTYSOLD"])
	qtyPurch, _ := db.AsFloat(row["DAY_QTYPURCH"])
	qtyTrsfIn, _ := db.AsFloat(row["DAY_QTYTRSFIN"])
	qtyTrsfOut, _ := db.AsFloat(row["DAY_QTYTRSFOUT"])
	sohEnd, _ := db.AsFloat(row["DAY_SOHEND"])

	returnUnitCount := unknown.Emit("0.0000",
		"No hay campo directo en DAILYTOTALS para devoluciones de venta. Validar con negocio")
	returnToVentorCount := unknown.Emit("0.0000",
		"Sin campo directo. Posible SUM(LFP_QTYRTV). Confirmar typo VENTOR vs VENDOR")
	adjustInCount := unknown.Emit("0.0000",
		"DAY_QTYUSAGE positivo o DAY_QTYINV. Separación positivo/negativo a definir con negocio")
	adjustOutCount := unknown.Emit("0.0000",
		"DAY_QTYUSAGE negativo o DAY_QTYEXPENSE. A definir con negocio")

	x.Open("Item")
	x.Element("STOCK_SEQ_NUMBER", cierreStockSeqNumber)
	x.Element("LOCATION_CODE", cierreLocationCode)
	x.Element("REVENUE_CENTER", cierreRevenueCenter)
	x.Element("ITEM_INVENTORY_STATE", cierreItemInventoryState)
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

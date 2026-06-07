package tlogsql

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/sequence"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// AdjustmentInventurGenerator implementa TLOG_INVENTORY_ADJUSTMENT con SQL.
//
// Filtro INVENTUR: KST_ID = ? AND INV_STATUS = 8 AND INV_TYP = 4.
// Para cada INV_ID se cargan las líneas de INVPOSART (con join a ARTIKEL).
type AdjustmentInventurGenerator struct{}

func (AdjustmentInventurGenerator) Type() naming.TLOGType { return naming.TLOGAdjustmentInventur }

const adjustmentInventurCandidatesSQL = `
SELECT DISTINCT I.INV_ID, K.KST_CODE, I.INV_NAME, I.CHG_ZEIT
FROM main.INVENTUR I
	INNER JOIN main.KOSTST K ON I.KST_ID = K.KST_ID
WHERE I.KST_ID = ? AND I.INV_STATUS = 8 AND I.INV_TYP = 4
ORDER BY I.INV_ID`

func (AdjustmentInventurGenerator) ListCandidateIDs(ctx context.Context, conn *sql.DB, kstID string) ([]string, error) {
	rows, err := queryRows(ctx, conn, adjustmentInventurCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("adjustment candidatos: %w", err)
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r["INV_ID"])
	}
	return ids, nil
}

func (g AdjustmentInventurGenerator) BuildSeqMap(ctx context.Context, conn *sql.DB, kstID string, businessDay time.Time, startCounter int) (tlog.DocSeqMap, int, error) {
	ids, err := g.ListCandidateIDs(ctx, conn, kstID)
	if err != nil {
		return nil, 0, err
	}
	return buildSeqMapFromIDs(ids, businessDay, sequence.DocAdjustmentInventur, startCounter)
}

func (AdjustmentInventurGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, crossSeqMap tlog.DocSeqMap, _ int) (*tlog.GenerateResult, error) {
	candidates, err := queryRows(ctx, conn, adjustmentInventurCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("adjustment candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	retailID := common.FormatRetailStoreID(candidates[0]["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, inv := range candidates {
		lines, err := adjustmentLines(ctx, conn, inv["INV_ID"])
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		seqNum := seqMap[inv["INV_ID"]]
		if seqNum == "" {
			return nil, fmt.Errorf("adjustment: sin sequence pre-asignado para INV_ID=%s", inv["INV_ID"])
		}
		countSeqNum := crossSeqMap[inv["INV_ID"]]

		x := common.NewXMLBuilder()
		writeAdjustmentDoc(x, h, retailID, seqNum, countSeqNum, inv, lines)
		files = append(files, tlog.GeneratedFile{
			SeqNum:     seqNum,
			XMLContent: x.String(),
			NumLines:   len(lines),
		})
		totalLines += len(lines)
	}

	if len(files) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}
	return &tlog.GenerateResult{
		Files:    files,
		NumDocs:  len(files),
		NumLines: totalLines,
	}, nil
}

// adjustmentLines devuelve las líneas de INVPOSART de un inventario, joineando
// ARTIKEL para arrastrar ART_NUMMER y ART_NAME.
func adjustmentLines(ctx context.Context, conn *sql.DB, invID string) ([]map[string]string, error) {
	const linesSQL = `
		SELECT distinct inv.INV_ID, inv.ART_ID, inv.VPK_ID, inv.INP_IST, inv.INP_SOLL,
			   inv.INP_EKP, inv.INP_VKP, 
			   art.ART_NUMMER, art.ART_NAME, art.ART_NR, art.CHG_ZEIT
		FROM INVPOSART inv
				 LEFT JOIN ARTIKEL art ON art.ART_ID = inv.ART_ID
		WHERE inv.INV_ID = ? AND inv.INP_SOLL - INV.INP_IST <> 0
		ORDER BY inv.ART_ID
		`
	rows, err := queryRows(ctx, conn, linesSQL, invID)
	if err != nil {
		return nil, fmt.Errorf("invposart INV=%s: %w", invID, err)
	}
	return rows, nil
}

func writeAdjustmentDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum, countSeqNum string,
	inv map[string]string, lines []map[string]string) {

	createTimestamp := h.FormatARTimestamp(h.BeginDateTime)
	if t, err := time.Parse("2006-01-02 15:04:05", inv["CHG_ZEIT"]); err == nil {
		createTimestamp = h.FormatARTimestamp(t)
	}

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", "0")
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", "0")
	x.Element("Subperiod", "0")
	x.Element("PeriodCode", "0")
	x.Element("SubPeriodCode", "0")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	x.Element("DocumentTypeCode", "InventoryAdjustment")
	x.Element("InventoryControlDocumentState", "2")
	x.Element("contractReferenceNumber", "Generado desde la Web")
	x.Element("CreateDateTimestamp", createTimestamp)
	x.Element("DestinationRetailStoreID", retailID)
	x.Element("ExpectedDeliveryDate", h.FormatARTimestamp(h.BeginDateTime))
	x.EmptyElement("ICDAmount")
	x.Element("LastUpdateDate", h.FormatARTimestamp(h.BeginDateTime))
	x.EmptyElement("Originator")
	x.Element("SourceRetailStore", retailID)
	x.EmptyElement("Supplier")
	x.EmptyElement("OrderDocumentType")
	x.Element("User", h.OperatorID)
	x.EmptyElement("ICDQuantity")
	x.EmptyElement("ICDTotSalesAmount")
	x.EmptyElement("Frequency")
	x.Element("InventoryAdjustmentType", "CORRECTIVE_ADJUSTMENT")
	x.EmptyElement("ReceiptNumber")
	x.Element("FiscalReceiptFlag", "false")
	x.EmptyElement("ReceiptType")
	x.Element("ReceiptDate", h.FormatARTimestamp(h.BeginDateTime))
	x.EmptyElement("CAINumber")
	x.EmptyElement("CAIDate")
	x.EmptyElement("PagesQuantity")
	x.EmptyElement("NetAmount")
	x.EmptyElement("ExemptAmout")
	x.EmptyElement("TaxAmount")
	x.EmptyElement("VatAmount")
	x.EmptyElement("ServicesVATAmount")
	x.EmptyElement("DifferencialVATAmount")
	x.EmptyElement("IvaTaxAmount")
	x.EmptyElement("IIBBTaxAmount")
	x.EmptyElement("TotalAmount")

	x.Open("InventoryControlDocumentLineItems")
	for i, line := range lines {
		writeAdjustmentLine(x, line, retailID, seqNum, i+1)
	}
	x.Close()
	x.Open("inventoryControlDocumentReferences")
	x.Open("inventoryControlDocumentReference")
	if seqNum == "" || countSeqNum == "" {
		x.EmptyElement("SerialFormID")
		x.EmptyElement("SerialFormIDTo")
	} else {
		x.Element("SerialFormID", seqNum)
		x.Element("SerialFormIDTo", countSeqNum)
	}
	x.Close()
	x.Close()
	x.Close()
	x.Close()
}

func writeAdjustmentLine(x *common.XMLBuilder, line map[string]string, retailID, seqNum string, detSeq int) {
	ist, _ := db.AsFloat(line["INP_IST"])
	soll, _ := db.AsFloat(line["INP_SOLL"])
	variance := ist - soll
	variance = math.Abs(variance)

	ekp, _ := db.AsFloat(line["INP_EKP"])
	costTotal := variance * ekp
	costTotal = math.Abs(costTotal)

	// El generator in-memory original usa artRow["ART_NR"], pero ART_NR no
	// existe en el schema SQLite (ARTIKEL solo tiene ART_ID/ART_NAMEID/
	// ART_NUMMER). Para parity, preservamos line["ART_NR"] que en este flujo
	// devuelve "". Si se quiere poblar el campo, editar la query de
	// invposartLines y/o el schema de ARTIKEL.
	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", "0")
	x.Element("SequenceNumber", seqNum)

	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NUMMER"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID"]))))
	x.EmptyElement("ItemBrand")
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(ekp))
	x.Element("UnitCount", common.FormatDecimal4(variance))
	x.Element("DestinationLocation", "DEP1_OS")
	x.Element("SourceLocation", "DEP1_OS")
	x.Element("CostTotalAmount", common.FormatDecimal4(math.Abs(costTotal)))
	x.Element("UnitSalesAmount", "0.0000")
	x.Element("SalesTotalAmount", "0.0000")
	x.Element("Stock", common.FormatDecimal4(soll))
	x.Element("DailyAverageSales", "0.0000")
	x.Element("SuggestedPurchaseOrder", "0.0000")
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

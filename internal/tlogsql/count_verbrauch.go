package tlogsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// CountVerbrauchGenerator implementa TLOG_INVENTORY_COUNT_REAL con SQL.
//
// Driver: HIS_VERBRAUCH (cabecera, VBR_STATUS = 2) + HIS_VERBRAUCHPOS (detalle).
type CountVerbrauchGenerator struct{}

func (CountVerbrauchGenerator) Type() naming.TLOGType { return naming.TLOGCountVerbrauch }

const countVerbrauchCandidatesSQL = `
SELECT V.VBR_ID, V.VBR_NAME, V.VRT_ID, V.CHG_ZEIT,
       K.KST_CODE
FROM HIS_VERBRAUCH V
    INNER JOIN KOSTST K ON V.KST_ID = K.KST_ID
WHERE V.KST_ID = ? AND V.VBR_STATUS = 2 AND V.VRT_ID IN (2, 3)
ORDER BY V.VBR_ID`

func (CountVerbrauchGenerator) ListCandidateIDs(ctx context.Context, conn *sql.DB, kstID string) ([]string, error) {
	rows, err := queryRows(ctx, conn, countVerbrauchCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("count candidatos: %w", err)
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r["VBR_ID"])
	}
	return ids, nil
}

func (CountVerbrauchGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, crossSeqMap tlog.DocSeqMap, _ int) (*tlog.GenerateResult, error) {
	candidates, err := queryRows(ctx, conn, countVerbrauchCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("count candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	retailID := common.FormatRetailStoreID(candidates[0]["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, vbr := range candidates {
		lines, err := conutHisVerbrauchposLines(ctx, conn, vbr["VBR_ID"])
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		seqNum := seqMap[vbr["VBR_ID"]]
		if seqNum == "" {
			return nil, fmt.Errorf("count: sin sequence pre-asignado para VBR_ID=%s", vbr["VBR_ID"])
		}
		adjSeqNum := crossSeqMap[vbr["VBR_ID"]]

		x := common.NewXMLBuilder()
		writeCountDoc(x, h, retailID, seqNum, adjSeqNum, vbr, lines)

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

// conutHisVerbrauchposLines devuelve las líneas de HIS_VERBRAUCHPOS para un VBR_ID,
// joineando ARTIKEL para obtener ART_NUMMER y ART_NAME.
func conutHisVerbrauchposLines(ctx context.Context, conn *sql.DB, vbrID string) ([]map[string]string, error) {
	const linesSQL = `
SELECT p.VBR_ID, p.VBT_POS, p.ART_NR, p.VBT_MENGE, p.VBT_WES, p.VPK_NR,
       a.ART_NUMMER, a.ART_NAME
FROM HIS_VERBRAUCHPOS p
    LEFT JOIN ARTIKEL a ON a.ART_ID = p.ART_NR
WHERE p.VBR_ID = ?
ORDER BY p.VBT_POS`
	rows, err := queryRows(ctx, conn, linesSQL, vbrID)
	if err != nil {
		return nil, fmt.Errorf("his_verbrauchpos VBR=%s: %w", vbrID, err)
	}
	return rows, nil
}

func writeCountDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum, adjSeqNum string,
	vbr map[string]string, lines []map[string]string) {

	createTimestamp := h.FormatARTimestamp(h.BeginDateTime)
	expectedDate := h.FormatARTimestamp(h.BeginDateTime)
	receiptDate := h.FormatARTimestamp(h.BeginDateTime)
	if t, err := time.Parse("2006-01-02 15:04:05", vbr["CHG_ZEIT"]); err == nil {
		createTimestamp = h.FormatARTimestamp(t)
		// ExpectedDeliveryDate y ReceiptDate usan solo la fecha (hora en cero)
		dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		expectedDate = h.FormatARTimestamp(dayStart)
		receiptDate = h.FormatARTimestamp(dayStart)
	}

	// ICDAmount = SUM(VBT_MENGE × VBT_WES)
	var icdAmount float64
	for _, l := range lines {
		menge, _ := db.AsFloat(l["VBT_MENGE"])
		wes, _ := db.AsFloat(l["VBT_WES"])
		icdAmount += menge * wes
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
	x.Element("DocumentTypeCode", "InventoryCount")
	x.Element("InventoryControlDocumentState", "4")
	x.Element("contractReferenceNumber", vbr["VBR_NAME"])
	x.Element("CreateDateTimestamp", createTimestamp)
	x.Element("DestinationRetailStoreID", retailID)
	x.Element("ExpectedDeliveryDate", expectedDate)
	x.Element("ICDAmount", common.FormatDecimal4(icdAmount))
	x.Element("LastUpdateDate", createTimestamp)
	x.EmptyElement("Originator")
	x.Element("SourceRetailStore", retailID)
	x.EmptyElement("Supplier")
	x.EmptyElement("OrderDocumentType")
	x.Element("User", h.OperatorID)
	x.EmptyElement("ICDQuantity")
	x.EmptyElement("ICDTotSalesAmount")
	x.EmptyElement("Frequency")
	x.Element("InventoryAdjustmentType", mapVrtIDToAdjType(vbr["VRT_ID"])) //todo validar
	x.EmptyElement("ReceiptNumber")
	x.Element("FiscalReceiptFlag", "false")
	x.EmptyElement("ReceiptType")
	x.Element("ReceiptDate", receiptDate)
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
	for _, line := range lines {
		writeCountLine(x, line, retailID, seqNum, createTimestamp)
	}
	x.Close()
	x.Open("inventoryControlDocumentReferences")
	x.Open("inventoryControlDocumentReference")
	if seqNum == "" || adjSeqNum == "" {
		x.EmptyElement("SerialFormID")
		x.EmptyElement("SerialFormIDTo")
	} else {
		x.Element("SerialFormID", adjSeqNum)
		x.Element("SerialFormIDTo", seqNum)
	}
	x.Close()
	x.Close()
	x.Close()
	x.Close()
}

func writeCountLine(x *common.XMLBuilder, line map[string]string, retailID, seqNum, lastUpdateDate string) {
	wes, _ := db.AsFloat(line["VBT_WES"])
	menge, _ := db.AsFloat(line["VBT_MENGE"])

	costTotalAmount := wes * menge

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", "0")
	x.Element("SequenceNumber", seqNum)
	x.Element("DetSequenceNumber", line["VBT_POS"])
	x.Element("Item", line["ART_NUMMER"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_NR"]))))
	x.EmptyElement("ItemBrand")
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(wes))
	x.Element("UnitCount", common.FormatDecimal4(menge))
	x.Element("DestinationLocation", "DEP1_OS")
	x.Element("SourceLocation", "DEP1_OS")
	x.Element("CostTotalAmount", common.FormatDecimal4(costTotalAmount))
	x.Element("UnitSalesAmount", "0.0000")
	x.Element("SalesTotalAmount", "0.0000")
	x.Element("Stock", "0.0000")
	x.Element("DailyAverageSales", "0.0000")
	x.Element("SuggestedPurchaseOrder", "0.0000")
	x.Element("PickupCode", "S1")
	x.Element("LastUpdateDate", lastUpdateDate)
	x.EmptyElement("DifBME_ASNTypeID")
	x.Element("InventoryControlDocumentState", "4")
	x.Close()
}

package tlogsql

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// CountInventurGenerator implementa TLOG_INVENTORY_COUNT_INVENTUR con SQL.
//
// Misma fuente que AdjustmentGenerator: INVENTUR (INV_STATUS=8, INV_TYP=4).
type CountInventurGenerator struct{}

func (CountInventurGenerator) Type() naming.TLOGType { return naming.TLOGCountInventur }

const countInventurCandidatesSQL = `
SELECT DISTINCT I.INV_ID, K.KST_CODE, I.INV_NAME, I.CHG_ZEIT
FROM main.INVENTUR I
	INNER JOIN main.KOSTST K ON I.KST_ID = K.KST_ID
WHERE I.KST_ID = ? AND I.INV_STATUS = 8 AND I.INV_TYP = 4
ORDER BY I.INV_ID`

func (CountInventurGenerator) ListCandidateIDs(ctx context.Context, conn *sql.DB, kstID string) ([]string, error) {
	rows, err := queryRows(ctx, conn, countInventurCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("count_inventur candidatos: %w", err)
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r["INV_ID"])
	}
	return ids, nil
}

func (CountInventurGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, crossSeqMap tlog.DocSeqMap, _ int) (*tlog.GenerateResult, error) {
	candidates, err := queryRows(ctx, conn, adjustmentInventurCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("count_inventur candidatos: %w", err)
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
			return nil, fmt.Errorf("count_inventur: sin sequence pre-asignado para INV_ID=%s", inv["INV_ID"])
		}
		adjSeqNum := crossSeqMap[inv["INV_ID"]]
		x := common.NewXMLBuilder()
		writeCountInventurDoc(x, h, retailID, seqNum, adjSeqNum, inv, lines)
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

func writeCountInventurDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum, adjSeqNum string,
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
	x.Element("DocumentTypeCode", "InventoryCount")
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
	x.Element("InventoryAdjustmentType", "CORRECTIVE_ADJUSTMENT") //todo validar
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
		writeCountInventurLine(x, line, retailID, seqNum, i+1)
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

func writeCountInventurLine(x *common.XMLBuilder, line map[string]string, retailID, seqNum string, detSeq int) {
	ist, _ := db.AsFloat(line["INP_IST"])
	soll, _ := db.AsFloat(line["INP_SOLL"])
	variance := ist - soll
	ekp, _ := db.AsFloat(line["INP_EKP"])
	costTotal := variance * ekp

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
	x.Element("Stock", common.FormatDecimal4(ist))
	x.Element("DailyAverageSales", "0.0000")
	x.Element("SuggestedPurchaseOrder", "0.0000")
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

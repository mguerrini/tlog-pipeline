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

const (
	returnDocumentTypeCode = "InventoryReturn"
	returnWorkstationID    = "0"
	returnPeriod           = "0"
	returnSubperiod        = "0"
	returnItemBrand        = "0"
	returnDestLocation     = "DEP1_OS"
	returnSourceLocation   = "DEP1_OS"
	returnUnitSales        = "0.0000"
	returnSalesTotal       = "0.0000"
	returnStock            = "0.0000"
	returnDailyAvg         = "0.0000"
	returnSuggestedPO      = "0.0000"
)

// ReturnGenerator implementa TLOG_INVENTORY_RETURN usando SQL.
//
// Filtro:
//   - LFS_RTS = 1 (es retorno)
//   - LFS_STATUS in (37, 42)
//   - LFS_BRUTTO < 0
//   - La primera línea de LIEFERPOS tiene KST_ID = ?
type ReturnGenerator struct{}

func (ReturnGenerator) Type() naming.TLOGType { return naming.TLOGReturn }

func (ReturnGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, startCounter int) (*tlog.GenerateResult, error) {
	const candidatesSQL = `
		SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
			l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
			INNER JOIN main.KOSTST K ON lpo.KST_ID1 = K.KST_ID
			INNER JOIN main.LIEFER L2 ON lpo.LF_ID = L2.LF_ID
		WHERE lpo.KST_ID = ? AND l.LFS_STATUS IN (37, 42)
			AND l.LFS_BRUTTO < 0 AND COALESCE(l.LFS_RTS, 0) = 1
		GROUP BY l.LFS_NAME
		ORDER BY l.LFS_NAME
`

	candidates, err := queryRows(ctx, conn, candidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("return candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	retailID := common.FormatRetailStoreID(candidates[0]["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, lfs := range candidates {
		lines, err := receptionLines(ctx, conn, lfs["LFS_ID"]) // mismo SELECT que reception
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}

		seqNum, err := sequence.Build(h.BusinessDay, sequence.DocReturn, startCounter+len(files))
		if err != nil {
			return nil, fmt.Errorf("return sequence: %w", err)
		}
		x := common.NewXMLBuilder()
		writeReturnDoc(x, h, retailID, seqNum, lfs, lines)
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

func writeReturnDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	lfs map[string]string, lines []map[string]string) {

	state := mapLFSStatusReturn(lfs["LFS_STATUS"])
	fiscalFlag := "false"
	if state == "7" {
		fiscalFlag = "true"
	}
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])
	receiptDate := h.FormatARTimestamp(h.BeginDateTime)
	if t, err := time.Parse("2006-01-02 15:04:05", lfs["LFS_DATUM"]); err == nil {
		receiptDate = h.FormatARTimestamp(t)
	}

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", returnWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", returnPeriod)
	x.Element("Subperiod", returnSubperiod)
	x.EmptyElement("PeriodCode")
	x.EmptyElement("SubPeriodCode")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	x.Element("DocumentTypeCode", returnDocumentTypeCode)
	x.Element("InventoryControlDocumentState", state)
	x.EmptyElement("contractReferenceNumber")
	x.Element("CreateDateTimestamp", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("DestinationRetailStoreID", retailID)
	x.Element("ExpectedDeliveryDate", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("ICDAmount", common.FormatDecimal4(math.Abs(brutto)))
	x.Element("LastUpdateDate", h.FormatARTimestamp(h.BeginDateTime))
	x.EmptyElement("Originator")
	x.Element("SourceRetailStore", retailID)
	x.Element("Supplier", lfs["LF_VERT"])
	x.EmptyElement("OrderDocumentType")
	x.Element("User", h.OperatorID)
	x.EmptyElement("ICDQuantity")
	x.EmptyElement("ICDTotSalesAmount")
	x.EmptyElement("Frequency")
	x.EmptyElement("InventoryAdjustmentType")
	x.Element("ReceiptNumber", lfs["LFS_NAME"])
	x.Element("FiscalReceiptFlag", fiscalFlag)
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
	for i, line := range lines {
		writeReturnLine(x, line, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeReturnLine(x *common.XMLBuilder, line map[string]string, detSeq int) {
	menge, _ := db.AsFloat(line["LFP_MENGE"])
	ekp, _ := db.AsFloat(line["LFP_EKP"])
	brutto, _ := db.AsFloat(line["LFP_BRUTTO"])
	var unitCost float64
	if menge != 0 {
		unitCost = math.Abs(ekp / menge)
	}

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NUMMER"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID1"]))))
	x.Element("ItemBrand", returnItemBrand)
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(unitCost))
	x.Element("UnitCount", common.FormatDecimal4(menge)) // viaja con signo original
	x.Element("DestinationLocation", returnDestLocation)
	x.Element("SourceLocation", returnSourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(brutto)) // signo original
	x.Element("UnitSalesAmount", returnUnitSales)
	x.Element("SalesTotalAmount", returnSalesTotal)
	x.Element("Stock", returnStock)
	x.Element("DailyAverageSales", returnDailyAvg)
	x.Element("SuggestedPurchaseOrder", returnSuggestedPO)
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

func mapLFSStatusReturn(s string) string {
	v, _ := db.AsInt(s)
	if v == 42 || v == 37 {
		return "4"
	}
	return "7"
}

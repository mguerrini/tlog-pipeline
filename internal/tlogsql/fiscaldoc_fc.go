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
	fcDocumentTypeCode  = "InventoryFiscalDoc"
	fcReceiptType       = "FC"
	fcInventoryDocState = "4"
	fcFiscalReceiptFlag = "true"
	fcWorkstationID     = "0"
	fcPeriod            = "0"
	fcSubperiod         = "0"
	fcItemBrand         = "0"
	fcDestLocation      = "DEP1_OS"
	fcSourceLocation    = "DEP1_OS"
	fcUnitSales         = "0.0000"
	fcSalesTotal        = "0.0000"
	fcStock             = "0.0000"
	fcDailyAvg          = "0.0000"
	fcSuggestedPO       = "0.0000"
)

// FiscalDocFCGenerator implementa TLOG_INVENTORY_FISCAL_DOC FC con SQL.
//
// Filtro idéntico al de Reception (LFS_STATUS=42, RTS!=1, NETTO/BRUTTO>0).
// Difiere en los elementos del XML.
type FiscalDocFCGenerator struct{}

func (FiscalDocFCGenerator) Type() naming.TLOGType { return naming.TLOGFiscalDocFC }

func (FiscalDocFCGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, startCounter int) (*tlog.GenerateResult, error) {
	const candidatesSQL = `
		SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
			l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
			INNER JOIN main.KOSTST K ON lpo.KST_ID1 = K.KST_ID
			INNER JOIN main.LIEFER L2 ON lpo.LF_ID = L2.LF_ID
		WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 42
			  AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_NETTO > 0 AND l.LFS_BRUTTO > 0
		GROUP BY l.LFS_NAME
		ORDER BY l.LFS_NAME
`
	candidates, err := queryRows(ctx, conn, candidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_fc candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	retailID := common.FormatRetailStoreID(candidates[0]["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, lfs := range candidates {
		lines, err := receptionLines(ctx, conn, lfs["LFS_ID"])
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		seqNum, err := sequence.Build(h.BusinessDay, sequence.DocFiscalDocFC, startCounter+len(files))
		if err != nil {
			return nil, fmt.Errorf("fiscaldoc_fc sequence: %w", err)
		}
		x := common.NewXMLBuilder()
		writeFCDoc(x, h, retailID, seqNum, lfs, lines)
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

func writeFCDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	lfs map[string]string, lines []map[string]string) {

	netto, _ := db.AsFloat(lfs["LFS_NETTO"])
	mwst, _ := db.AsFloat(lfs["LFS_MWST"])
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])
	receiptDate := h.FormatARTimestamp(h.BeginDateTime)
	if t, err := time.Parse("2006-01-02 15:04:05", lfs["LFS_DATUM"]); err == nil {
		receiptDate = h.FormatARTimestamp(t)
	}

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", fcWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", fcPeriod)
	x.Element("Subperiod", fcSubperiod)
	x.EmptyElement("PeriodCode")
	x.EmptyElement("SubPeriodCode")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	x.Element("DocumentTypeCode", fcDocumentTypeCode)
	x.Element("InventoryControlDocumentState", fcInventoryDocState)
	x.EmptyElement("contractReferenceNumber")
	x.Element("CreateDateTimestamp", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("DestinationRetailStoreID", retailID)
	x.Element("ExpectedDeliveryDate", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("ICDAmount", common.FormatDecimal4(brutto))
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
	x.Element("FiscalReceiptFlag", fcFiscalReceiptFlag)
	x.Element("ReceiptType", fcReceiptType)
	x.Element("ReceiptDate", receiptDate)
	x.EmptyElement("CAINumber")
	x.EmptyElement("CAIDate")
	x.EmptyElement("PagesQuantity")
	x.Element("NetAmount", common.FormatDecimal4(netto))
	x.Element("ExemptAmout", "0.0000")
	x.Element("TaxAmount", "0.0000")
	x.Element("VatAmount", common.FormatDecimal4(mwst))
	x.Element("ServicesVATAmount", "0.0000")
	x.Element("DifferencialVATAmount", "0.0000")
	x.Element("IvaTaxAmount", "0.0000")
	x.Element("IIBBTaxAmount", "0.0000")
	x.Element("TotalAmount", common.FormatDecimal4(brutto))

	x.Open("InventoryControlDocumentLineItems")
	for i, line := range lines {
		writeFCLine(x, line, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeFCLine(x *common.XMLBuilder, line map[string]string, detSeq int) {
	menge, _ := db.AsFloat(line["LFP_MENGE"])
	ekp, _ := db.AsFloat(line["LFP_EKP"])
	brutto, _ := db.AsFloat(line["LFP_BRUTTO"])
	var unitCost float64
	if menge != 0 {
		unitCost = ekp / menge
	}

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NR"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID1"]))))
	x.Element("ItemBrand", fcItemBrand)
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(unitCost))
	x.Element("UnitCount", common.FormatDecimal4(menge))
	x.Element("DestinationLocation", fcDestLocation)
	x.Element("SourceLocation", fcSourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(math.Abs(brutto)))
	x.Element("UnitSalesAmount", fcUnitSales)
	x.Element("SalesTotalAmount", fcSalesTotal)
	x.Element("Stock", fcStock)
	x.Element("DailyAverageSales", fcDailyAvg)
	x.Element("SuggestedPurchaseOrder", fcSuggestedPO)
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

package tlogsql

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/sequence"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

const (
	fcDocumentTypeCode    = "InventoryFiscalDoc"
	fcReceiptType         = "FC"
	fcInventoryDocState   = "4"
	fcFiscalReceiptFlag   = "true"
	fcWorkstationID       = "0"
	fcPeriod              = "0"
	fcSubperiod           = "0"
	fcItemBrand           = "0"
	fcDestLocation        = "DEP1_OS"
	fcSourceLocation      = "DEP1_OS"
	fcUnitSales           = "0.0000"
	fcSalesTotal          = "0.0000"
	fcStock               = "0.0000"
	fcDailyAvg            = "0.0000"
	fcSuggestedPO         = "0.0000"
)

// FiscalDocFCGenerator implementa TLOG_INVENTORY_FISCAL_DOC FC con SQL.
//
// Filtro idéntico al de Reception (LFS_STATUS=42, RTS!=1, NETTO/BRUTTO>0).
// Difiere en los elementos del XML.
type FiscalDocFCGenerator struct{}

func (FiscalDocFCGenerator) Type() naming.TLOGType { return naming.TLOGFiscalDocFC }

func (FiscalDocFCGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string) (*tlog.GenerateResult, error) {
	const candidatesSQL = `
SELECT lfs.*
FROM LIEFERSCHEIN lfs
WHERE lfs.LFS_STATUS = 42
  AND COALESCE(lfs.LFS_RTS, 0) <> 1
  AND lfs.LFS_NETTO > 0
  AND lfs.LFS_BRUTTO > 0
  AND (
    SELECT lfp.KST_ID
    FROM LIEFERPOS lfp
    WHERE lfp.LFS_ID = lfs.LFS_ID
    ORDER BY lfp.LFP_POS LIMIT 1
  ) = ?
ORDER BY lfs.LFS_ID`
	candidates, err := queryRows(ctx, conn, candidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_fc candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	kst, err := fetchKostst(ctx, conn, kstID)
	if err != nil {
		return nil, err
	}
	retailID := common.FormatRetailStoreID(kst["KST_CODE"])
	seqNum, err := sequence.Build(retailID, h.BusinessDay, sequence.DocFiscalDocFC)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_fc sequence: %w", err)
	}

	x := common.NewXMLBuilder()
	totalDocs, totalLines := 0, 0

	for _, lfs := range candidates {
		lines, err := receptionLines(ctx, conn, lfs["LFS_ID"])
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		liefer, err := selectOne(ctx, conn, `SELECT * FROM LIEFER WHERE LF_ID = ?`, lfs["LF_ID"])
		if err != nil {
			return nil, err
		}
		if liefer == nil {
			liefer = map[string]string{}
		}
		totalDocs++
		totalLines += len(lines)
		writeFCDoc(x, h, retailID, seqNum, lfs, liefer, lines)
	}

	if totalDocs == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}
	return &tlog.GenerateResult{
		XMLContent: x.String(),
		NumDocs:    totalDocs,
		NumLines:   totalLines,
	}, nil
}

func writeFCDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	lfs, liefer map[string]string, lines []map[string]string) {

	netto, _ := db.AsFloat(lfs["LFS_NETTO"])
	mwst, _ := db.AsFloat(lfs["LFS_MWST"])
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])

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
	x.Element("Supplier", liefer["LF_VERT"])
	x.EmptyElement("OrderDocumentType")
	x.Element("User", h.OperatorID)
	x.EmptyElement("ICDQuantity")
	x.EmptyElement("ICDTotSalesAmount")
	x.EmptyElement("Frequency")
	x.EmptyElement("InventoryAdjustmentType")
	x.Element("ReceiptNumber", lfs["LFS_NAME"])
	x.Element("FiscalReceiptFlag", fcFiscalReceiptFlag)
	x.Element("ReceiptType", fcReceiptType)
	x.Element("ReceiptDate", h.FormatARTimestamp(h.BeginDateTime))
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

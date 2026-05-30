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

const (
	ncDocumentTypeCode  = "InventoryFiscalDoc"
	ncReceiptType       = "NC"
	ncInventoryDocState = "4"
	ncFiscalReceiptFlag = "false"
	ncWorkstationID     = "0"
	ncPeriod            = "0"
	ncSubperiod         = "0"
	ncItemBrand         = "0"
	ncDestLocation      = "DEP1_OS"
	ncSourceLocation    = "DEP1_OS"
	ncUnitSales         = "0.0000"
	ncSalesTotal        = "0.0000"
	ncStock             = "0.0000"
	ncDailyAvg          = "0.0000"
	ncSuggestedPO       = "0.0000"
)

// FiscalDocNCGenerator implementa TLOG_INVENTORY_FISCAL_DOC NC con SQL.
//
// Filtro: LFS_STATUS=42, LFS_RTS=1, LFS_NETTO<0, LFS_BRUTTO<0.
type FiscalDocNCGenerator struct{}

func (FiscalDocNCGenerator) Type() naming.TLOGType { return naming.TLOGFiscalDocNC }

const fiscalDocNCCandidatesSQL = `
	SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
		l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST, L2.LF_SACHB
	FROM LIEFERSCHEIN l
		INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
		INNER JOIN KOSTST K ON lpo.KST_ID1 = K.KST_ID
		INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
	WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 42
		AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_NETTO < 0 AND l.LFS_BRUTTO < 0
	GROUP BY l.LFS_NAME
	ORDER BY l.LFS_NAME
`

func (FiscalDocNCGenerator) ListCandidateIDs(ctx context.Context, conn *sql.DB, kstID string) ([]string, error) {
	rows, err := queryRows(ctx, conn, fiscalDocNCCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_nc candidatos: %w", err)
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r["LFS_ID"])
	}
	return ids, nil
}

func (FiscalDocNCGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, crossSeqMap tlog.DocSeqMap, _ int) (*tlog.GenerateResult, error) {
	candidates, err := queryRows(ctx, conn, fiscalDocNCCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_nc candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	retailID := common.FormatRetailStoreID(candidates[0]["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, lfs := range candidates {
		lines, err := fiscalDocReceptionLines(ctx, conn, lfs["LFS_ID"], h.IsProduction)
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		seqNum := seqMap[lfs["LFS_ID"]]
		if seqNum == "" {
			return nil, fmt.Errorf("fiscaldoc_nc: sin sequence pre-asignado para LFS_ID=%s", lfs["LFS_ID"])
		}
		seqNumTO := crossSeqMap[lfs["LFS_ID"]]
		hdr, err := queryFiscalDocHeaderData(ctx, conn, h, lfs["LFS_ID"])
		if err != nil {
			return nil, fmt.Errorf("fiscaldoc_nc header %s: %w", lfs["LFS_ID"], err)
		}
		x := common.NewXMLBuilder()
		writeNCDoc(x, h, retailID, seqNum, seqNumTO, lfs, lines, hdr)
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

func writeNCDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum, seqNumTO string,
	lfs map[string]string, lines []map[string]string, hdr fiscalDocHeaderData) {

	netto, _ := db.AsFloat(lfs["LFS_NETTO"])
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])
	receiptDate := h.FormatARTimestamp(h.BeginDateTime)
	if t, err := time.Parse("2006-01-02 15:04:05", lfs["LFS_DATUM"]); err == nil {
		receiptDate = h.FormatARTimestamp(t)
	}

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", ncWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", ncPeriod)
	x.Element("Subperiod", ncSubperiod)
	x.Element("PeriodCode", "0")
	x.Element("SubPeriodCode", "0")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	if seqNumTO != "" {
		x.Element("SerialFormIDTO", seqNumTO)
	} else {
		x.EmptyElement("SerialFormIDTO")
	}
	x.Element("DocumentTypeCode", ncDocumentTypeCode)
	x.Element("InventoryControlDocumentState", ncInventoryDocState)
	x.EmptyElement("contractReferenceNumber")
	x.Element("CreateDateTimestamp", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("DestinationRetailStoreID", retailID)
	x.Element("ExpectedDeliveryDate", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("ICDAmount", common.FormatDecimal4(brutto))
	x.Element("LastUpdateDate", h.FormatARTimestamp(h.BeginDateTime))
	x.EmptyElement("Originator")
	x.Element("SourceRetailStore", retailID)
	x.Element("Supplier", lfs["LF_SACHB"])
	x.EmptyElement("OrderDocumentType")
	x.Element("User", h.OperatorID)
	x.EmptyElement("ICDQuantity")
	x.EmptyElement("ICDTotSalesAmount")
	x.EmptyElement("Frequency")
	x.EmptyElement("InventoryAdjustmentType")
	x.Element("ReceiptNumber", lfs["LFS_NAME"])
	x.Element("FiscalReceiptFlag", ncFiscalReceiptFlag)
	x.Element("ReceiptType", ncReceiptType)
	x.Element("ReceiptDate", receiptDate)
	x.Element("CAINumber", hdr.CAINumber)
	x.Element("CAIDate", hdr.CAIDate)
	x.EmptyElement("PagesQuantity")
	x.Element("NetAmount", common.FormatDecimal4(netto))
	x.Element("ExemptAmout", common.FormatDecimal4(hdr.ExemptAmount))
	x.Element("TaxAmount", common.FormatDecimal4(hdr.TaxAmount))
	x.Element("VatAmount", common.FormatDecimal4(hdr.VatAmount))
	x.Element("ServicesVATAmount", "0.0000")
	x.Element("DifferencialVATAmount", "0.0000")
	x.Element("IvaTaxAmount", common.FormatDecimal4(hdr.IvaTaxAmount))
	x.Element("IIBBTaxAmount", common.FormatDecimal4(hdr.IIBBTaxAmount))
	x.Element("TotalAmount", common.FormatDecimal4(brutto))

	x.Open("InventoryControlDocumentLineItems")
	for i, line := range lines {
		writeNCLine(x, line, retailID, seqNum, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeNCLine(x *common.XMLBuilder, line map[string]string, retailID, seqNum string, detSeq int) {
	menge, _ := db.AsFloat(line["LFP_MENGE"])
	ekp, _ := db.AsFloat(line["LFP_EKP"])
	brutto, _ := db.AsFloat(line["LFP_BRUTTO"])
	var unitCost float64
	if menge != 0 {
		unitCost = ekp / menge
	}

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", ncWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NR"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID1"]))))
	x.Element("ItemBrand", ncItemBrand)
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(unitCost))
	x.Element("UnitCount", common.FormatDecimal4(menge)) // negativo
	x.Element("DestinationLocation", ncDestLocation)
	x.Element("SourceLocation", ncSourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(brutto)) // negativo
	x.Element("UnitSalesAmount", ncUnitSales)
	x.Element("SalesTotalAmount", ncSalesTotal)
	x.Element("Stock", ncStock)
	x.Element("DailyAverageSales", ncDailyAvg)
	x.Element("SuggestedPurchaseOrder", ncSuggestedPO)
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

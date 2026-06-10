package tlogsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/sequence"
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
SELECT DISTINCT l.RNG_NAME, l.LFS_ID, k.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,  l.LFS_NAME, l.LFS_DATUM,
                l.LFS_NETTO, l.LFS_MWST, l.LF_SACHB
FROM LIEFERSCHEIN_VIEW l
         INNER JOIN KOSTST k ON l.KST_ID = k.KST_ID
WHERE l.KST_ID = ? AND l.LFS_STATUS = 42
  AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_NETTO < 0 AND l.LFS_BRUTTO < 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME;
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

// BuildSeqMap asigna un seqNum por RNG_NAME; todos los LFS_ID del mismo
// RNG_NAME comparten ese seqNum. Retorna LFS_ID → seqNum y la cantidad de
// RNG_NAMEs distintos (seqNums consumidos).
func (FiscalDocNCGenerator) BuildSeqMap(ctx context.Context, conn *sql.DB, kstID string, businessDay time.Time, startCounter int) (tlog.DocSeqMap, int, error) {
	rows, err := queryRows(ctx, conn, fiscalDocNCCandidatesSQL, kstID)
	if err != nil {
		return nil, 0, fmt.Errorf("fiscaldoc_nc candidatos: %w", err)
	}
	if len(rows) == 0 {
		return nil, 0, nil
	}
	rngSeq := make(map[string]string) // RNG_NAME → seqNum
	sm := make(tlog.DocSeqMap, len(rows))
	for _, r := range rows {
		rng := r["RNG_NAME"]
		seqNum, ok := rngSeq[rng]
		if !ok {
			seqNum, err = sequence.Build(businessDay, sequence.DocFiscalDocNC, startCounter+len(rngSeq))
			if err != nil {
				return nil, 0, err
			}
			rngSeq[rng] = seqNum
		}
		sm[r["LFS_ID"]] = seqNum
	}
	return sm, len(rngSeq), nil
}

func (FiscalDocNCGenerator) Generate(ctx context.Context, genCtx *GeneratorContext, conn *sql.DB, _ int) (*tlog.GenerateResult, error) {
	kstID := genCtx.KstID
	h := genCtx.Header
	seqMap := genCtx.SeqMap
	crossSeqMap := genCtx.CrossSeqMap
	candidates, err := queryRows(ctx, conn, fiscalDocNCCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_nc candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	retailID := common.FormatRetailStoreID(candidates[0]["KST_CODE"])

	// Agrupar candidatos por RNG_NAME preservando el orden de primera aparición.
	type rngGroup struct {
		firstRow map[string]string
		lfsIDs   []string
	}
	var rngOrder []string
	rngGroups := make(map[string]*rngGroup)
	for _, lfs := range candidates {
		rng := lfs["RNG_NAME"]
		if _, ok := rngGroups[rng]; !ok {
			rngOrder = append(rngOrder, rng)
			rngGroups[rng] = &rngGroup{firstRow: lfs}
		}
		rngGroups[rng].lfsIDs = append(rngGroups[rng].lfsIDs, lfs["LFS_ID"])
	}

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, rng := range rngOrder {
		g := rngGroups[rng]
		seqNum := seqMap[g.lfsIDs[0]]
		if seqNum == "" {
			return nil, fmt.Errorf("fiscaldoc_nc: sin sequence pre-asignado para RNG_NAME=%s", rng)
		}

		crossSeqNums := make([]string, len(g.lfsIDs))
		for i, lfsID := range g.lfsIDs {
			crossSeqNums[i] = crossSeqMap[lfsID]
		}

		lines, err := fiscalDocReceptionLines(ctx, conn, rng)
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		hdr, err := queryFiscalDocHeaderData(ctx, conn, h, rng)
		if err != nil {
			return nil, fmt.Errorf("fiscaldoc_nc header %s: %w", rng, err)
		}
		x := common.NewXMLBuilder()
		writeNCDoc(x, h, retailID, seqNum, crossSeqNums, g.firstRow, lines, hdr)
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

func writeNCDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string, crossSeqNums []string,
	lfs map[string]string, lines []map[string]string, hdr fiscalDocHeaderData) {

	netto, _ := db.AsFloat(lfs["LFS_NETTO"])
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])
	receiptDate := h.FormatARTimestamp(h.BeginDateTime)
	if t, ok := parseFiscalDate(lfs["LFS_DATUM"]); ok {
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
	x.Element("ReceiptNumber", lfs["RNG_NAME"])
	x.Element("FiscalReceiptFlag", ncFiscalReceiptFlag)
	x.Element("ReceiptType", ncReceiptType)
	x.Element("ReceiptDate", receiptDate)
	x.Element("CAINumber", hdr.CAINumber)
	x.Element("CAIDate", hdr.CAIDate)
	x.EmptyElement("PagesQuantity")

	//	netto := restarle - TAxAmount - VatAmount - IVATaxAmount - IIBBTAXAMOUT
	netAmount := netto - hdr.TaxAmount - hdr.VatAmount - hdr.IvaTaxAmount - hdr.IIBBTaxAmount - hdr.DifferentialIVAVatAMount
	x.Element("NetAmount", common.FormatDecimal4(netAmount))
	x.Element("ExemptAmout", common.FormatDecimal4(hdr.ExemptAmount))
	x.Element("TaxAmount", common.FormatDecimal4(hdr.TaxAmount))
	x.Element("VatAmount", common.FormatDecimal4(hdr.VatAmount))
	x.Element("ServicesVATAmount", "0.0000")
	x.Element("DifferencialVATAmount", common.FormatDecimal4(hdr.DifferentialIVAVatAMount))
	x.Element("IvaTaxAmount", common.FormatDecimal4(hdr.IvaTaxAmount))
	x.Element("IIBBTaxAmount", common.FormatDecimal4(hdr.IIBBTaxAmount))
	x.Element("TotalAmount", common.FormatDecimal4(brutto))

	x.Open("InventoryControlDocumentLineItems")
	for i, line := range lines {
		writeNCLine(x, line, retailID, seqNum, i+1)
	}
	x.Close()
	x.Open("inventoryControlDocumentReferences")
	for _, crossSeqNum := range crossSeqNums {
		x.Open("inventoryControlDocumentReference")
		if seqNum == "" || crossSeqNum == "" {
			x.EmptyElement("SerialFormID")
			x.EmptyElement("SerialFormIDTo")
		} else {
			x.Element("SerialFormID", crossSeqNum)
			x.Element("SerialFormIDTo", seqNum)
		}
		x.Close()
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeNCLine(x *common.XMLBuilder, line map[string]string, retailID, seqNum string, detSeq int) {
	ekp, _ := db.AsFloat(line["LFP_EKP"])
	brutto, _ := db.AsFloat(line["LFP_BRUTTO"])
	mengege, _ := db.AsFloat(line["LFP_MENGEGE"])

	var unitCost float64
	if mengege != 0 {
		unitCost = ekp / mengege
	}

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", ncWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NUMMER"])
	x.Element("UomUnits", "1")
	x.Element("ItemBrand", ncItemBrand)
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(unitCost))
	x.Element("UnitCount", common.FormatDecimal4(mengege)) // negativo
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

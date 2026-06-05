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
	fcFiscalReceiptFlag = "false"
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

const fiscalDocFCCandidatesSQL = `
SELECT DISTINCT l.RNG_NAME, l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, l.LFS_NAME, l.LFS_DATUM,
                l.LFS_NETTO, l.LFS_MWST, L2.LF_SACHB
FROM LIEFERSCHEIN_VIEW l
         INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
         INNER JOIN KOSTST K ON lpo.KST_ID1 = K.KST_ID
         INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 42
  AND COALESCE(l.LFS_RTS, 0) <> 1 AND l.LFS_NETTO > 0 AND l.LFS_BRUTTO > 0
  AND l.RNG_COD <> 1
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME;
`

func (FiscalDocFCGenerator) ListCandidateIDs(ctx context.Context, conn *sql.DB, kstID string) ([]string, error) {
	rows, err := queryRows(ctx, conn, fiscalDocFCCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_fc candidatos: %w", err)
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
func (FiscalDocFCGenerator) BuildSeqMap(ctx context.Context, conn *sql.DB, kstID string, businessDay time.Time, startCounter int) (tlog.DocSeqMap, int, error) {
	rows, err := queryRows(ctx, conn, fiscalDocFCCandidatesSQL, kstID)
	if err != nil {
		return nil, 0, fmt.Errorf("fiscaldoc_fc candidatos: %w", err)
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
			seqNum, err = sequence.Build(businessDay, sequence.DocFiscalDocFC, startCounter+len(rngSeq))
			if err != nil {
				return nil, 0, err
			}
			rngSeq[rng] = seqNum
		}
		sm[r["LFS_ID"]] = seqNum
	}
	return sm, len(rngSeq), nil
}

func (FiscalDocFCGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, crossSeqMap tlog.DocSeqMap, _ int) (*tlog.GenerateResult, error) {
	candidates, err := queryRows(ctx, conn, fiscalDocFCCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("fiscaldoc_fc candidatos: %w", err)
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
			return nil, fmt.Errorf("fiscaldoc_fc: sin sequence pre-asignado para RNG_NAME=%s", rng)
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
			return nil, fmt.Errorf("fiscaldoc_fc header %s: %w", rng, err)
		}
		x := common.NewXMLBuilder()
		kept := writeFCDoc(x, h, retailID, seqNum, crossSeqNums, g.firstRow, lines, hdr)
		files = append(files, tlog.GeneratedFile{
			SeqNum:     seqNum,
			XMLContent: x.String(),
			NumLines:   kept,
		})
		totalLines += kept
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

func writeFCDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string, crossSeqNums []string,
	lfs map[string]string, lines []map[string]string, hdr fiscalDocHeaderData) int {

	netto, _ := db.AsFloat(lfs["LFS_NETTO"])
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])
	receiptDate := h.FormatARTimestamp(h.BeginDateTime)
	if t, ok := parseFiscalDate(lfs["LFS_DATUM"]); ok {
		receiptDate = h.FormatARTimestamp(t)
	}

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", fcWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", fcPeriod)
	x.Element("Subperiod", fcSubperiod)
	x.Element("PeriodCode", "0")
	x.Element("SubPeriodCode", "0")
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
	x.Element("Supplier", lfs["LF_SACHB"])
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
	x.Element("CAINumber", hdr.CAINumber)
	x.Element("CAIDate", hdr.CAIDate)
	x.EmptyElement("PagesQuantity")
	x.Element("NetAmount", common.FormatDecimal4(netto))
	x.Element("ExemptAmout", common.FormatDecimal4(hdr.ExemptAmount))
	x.Element("TaxAmount", common.FormatDecimal4(hdr.TaxAmount))
	x.Element("VatAmount", common.FormatDecimal4(hdr.VatAmount))
	x.Element("ServicesVATAmount", "0.0000")
	x.Element("DifferencialVATAmount", common.FormatDecimal4(hdr.DifferentialIVAVatAMount))
	x.Element("IvaTaxAmount", common.FormatDecimal4(hdr.IvaTaxAmount))
	x.Element("IIBBTaxAmount", common.FormatDecimal4(hdr.IIBBTaxAmount))
	x.Element("TotalAmount", common.FormatDecimal4(brutto))

	x.Open("InventoryControlDocumentLineItems")
	detSeq := 0
	for _, line := range lines {
		switch line["ART_NR"] {
		case "1120", "1100", "1098", "1096":
			continue
		}
		detSeq++
		writeFCLine(x, line, retailID, seqNum, detSeq)
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
	return detSeq
}

func writeFCLine(x *common.XMLBuilder, line map[string]string, retailID, seqNum string, detSeq int) {
	menge, _ := db.AsFloat(line["LFP_MENGE"])
	ekp, _ := db.AsFloat(line["LFP_EKP"])
	brutto, _ := db.AsFloat(line["LFP_BRUTTO"])
	var unitCost float64
	if menge != 0 {
		unitCost = ekp / menge
	}

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", fcWorkstationID)
	x.Element("SequenceNumber", seqNum)

	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NUMMER"])
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

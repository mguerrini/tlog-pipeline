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
	transferDocumentTypeCode  = "InventoryTransfer"
	transferInventoryDocState = "2"
	transferWorkstationID     = "0"
	transferPeriod            = "0"
	transferSubperiod         = "0"
	transferItemBrand         = "0"
	transferDestLocation      = "DEP1_OS"
	transferSourceLocation    = "DEP1_OS"
	transferFiscalFlag        = "false"
)

// TransferGenerator implementa TLOG_INVENTORY_TRANSFER.
//
// Driver: HIS_LAGERBEW (cabecera, LBW_STATUS = 51) + HIS_LAGBEWPOS (detalle).
// KST_ID es el origen del envío; KST_ID1 es el destino.
type TransferGenerator struct{}

func (TransferGenerator) Type() naming.TLOGType { return naming.TLOGTransfer }

const transferCandidatesSQL = `
	SELECT lbw.LBW_ID, lbw.LBW_STATUS, lbw.CHG_ZEIT,
	       lbw.KST_ID, lbw.KST_ID1,
	       kst.KST_CODE,
	       kst1.KST_CODE AS KST_CODE1
	FROM HIS_LAGERBEW lbw
	    INNER JOIN KOSTST kst  ON kst.KST_ID  = lbw.KST_ID
	    INNER JOIN KOSTST kst1 ON kst1.KST_ID = lbw.KST_ID1
	WHERE lbw.KST_ID = ? AND lbw.LBW_STATUS = 51
	ORDER BY lbw.LBW_ID
`

func (TransferGenerator) ListCandidateIDs(ctx context.Context, conn *sql.DB, kstID string) ([]string, error) {
	rows, err := queryRows(ctx, conn, transferCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("transfer candidatos: %w", err)
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r["LBW_ID"])
	}
	return ids, nil
}

func (g TransferGenerator) BuildSeqMap(ctx context.Context, conn *sql.DB, kstID string, businessDay time.Time, startCounter int) (tlog.DocSeqMap, int, error) {
	ids, err := g.ListCandidateIDs(ctx, conn, kstID)
	if err != nil {
		return nil, 0, err
	}
	return buildSeqMapFromIDs(ids, businessDay, sequence.DocTransfer, startCounter)
}

func (TransferGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, _ tlog.DocSeqMap, _ int) (*tlog.GenerateResult, error) {
	candidates, err := queryRows(ctx, conn, transferCandidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("transfer candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	retailID := common.FormatRetailStoreID(candidates[0]["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, lbw := range candidates {
		lines, err := transferLines(ctx, conn, lbw["LBW_ID"])
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}

		seqNum := seqMap[lbw["LBW_ID"]]
		if seqNum == "" {
			return nil, fmt.Errorf("transfer: sin sequence pre-asignado para LBW_ID=%s", lbw["LBW_ID"])
		}

		x := common.NewXMLBuilder()
		writeTransferDoc(x, h, retailID, seqNum, lbw, lines)
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

func transferLines(ctx context.Context, conn *sql.DB, lbwID string) ([]map[string]string, error) {
	const linesSQL = `
SELECT lbp.LBW_ID, lbp.LBP_POS, lbp.ART_NR, lbp.VPK_ID,
       lbp.LBP_MENGEGE, lbp.LBP_ESP, art.ART_NAME, art.ART_NUMMER, 
       lbw.KST_ID, lbw.KST_ID1, KST.KST_CODE, KST1.KST_CODE AS KST_CODE1
FROM HIS_LAGBEWPOS lbp
    INNER JOIN HIS_LAGERBEW lbw ON lbw.LBW_ID = lbp.LBW_ID
    INNER JOIN KOSTST kst  ON kst.KST_ID  = lbw.KST_ID
    INNER JOIN KOSTST kst1 ON kst1.KST_ID = lbw.KST_ID1
    LEFT JOIN ARTIKEL art ON art.ART_ID = lbp.ART_NR
WHERE lbp.LBW_ID = ?
ORDER BY lbp.LBP_POS`
	rows, err := queryRows(ctx, conn, linesSQL, lbwID)
	if err != nil {
		return nil, fmt.Errorf("transfer lineas LBW=%s: %w", lbwID, err)
	}
	return rows, nil
}

func writeTransferDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	lbw map[string]string, lines []map[string]string) {

	chgZeit := h.FormatARTimestamp(h.BeginDateTime)
	if t, err := time.Parse("2006-01-02 15:04:05", lbw["CHG_ZEIT"]); err == nil {
		chgZeit = h.FormatARTimestamp(t)
	}

	var icdAmount float64
	for _, l := range lines {
		menge, _ := db.AsFloat(l["LBP_MENGEGE"])
		esp, _ := db.AsFloat(l["LBP_ESP"])
		icdAmount += menge * esp
	}

	destRetailID := common.FormatRetailStoreID(lbw["KST_CODE1"])

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", transferWorkstationID)
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
	x.Element("DocumentTypeCode", "InventoryTransfer")
	x.Element("InventoryControlDocumentState", "2")
	x.EmptyElement("ContractReferenceNumber")
	x.Element("CreateDateTimestamp", chgZeit)
	x.Element("DestinationRetailStoreID", destRetailID)
	x.Element("ExpectedDeliveryDate", chgZeit)
	x.Element("ICDAmount", common.FormatDecimal4(math.Abs(icdAmount)))
	x.Element("LastUpdateDate", chgZeit)
	x.EmptyElement("Originator")
	x.Element("SourceRetailStore", retailID)
	x.EmptyElement("Supplier")
	x.EmptyElement("OrderDocumentType")
	x.Element("User", h.OperatorID)
	x.EmptyElement("ICDQuantity")
	x.EmptyElement("ICDTotSalesAmount")
	x.EmptyElement("Frequency")
	x.EmptyElement("InventoryAdjustmentType")
	x.EmptyElement("ReceiptNumber")
	x.Element("FiscalReceiptFlag", "false")
	x.EmptyElement("ReceiptType")
	x.Element("ReceiptDate", chgZeit)
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
		writeTransferLine(x, line, retailID, seqNum, chgZeit, i+1)
	}
	x.Close()
	x.Open("inventoryControlDocumentReferences")
	x.Open("inventoryControlDocumentReference")
	x.EmptyElement("SerialFormID")
	x.EmptyElement("SerialFormIDTo")
	x.Close()
	x.Close()
	x.Close()
	x.Close()
}

func writeTransferLine(x *common.XMLBuilder, line map[string]string, retailID, seqNum, lastUpdateDate string, detSeq int) {
	menge, _ := db.AsFloat(line["LBP_MENGEGE"])
	esp, _ := db.AsFloat(line["LBP_ESP"])
	total := math.Abs(menge * esp)

	destRetailID := common.FormatRetailStoreID(line["KST_CODE1"])

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", "0")
	x.Element("SequenceNumber", seqNum)
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NR"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID"]))))
	x.Element("ItemBrand", "0")
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(esp))
	x.Element("UnitCount", common.FormatDecimal4(menge))
	x.Element("DestinationLocation", destRetailID)
	x.Element("SourceLocation", retailID)
	x.Element("CostTotalAmount", common.FormatDecimal4(total))
	x.Element("UnitSalesAmount", "0.0000")
	x.Element("SalesTotalAmount", "0.0000")
	x.Element("Stock", "0.0000")
	x.Element("DailyAverageSales", "0.0000")
	x.Element("SuggestedPurchaseOrder", "0.0000")
	x.EmptyElement("PickupCode")
	x.Element("LastUpdateDate", lastUpdateDate)
	x.EmptyElement("DifBME_ASNTypeID")
	x.Element("InventoryControlDocumentState", "2")
	x.Close()
}

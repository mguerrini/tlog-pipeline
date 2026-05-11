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
	receptionDocumentTypeCode = "InventoryReception"
	receptionWorkstationID    = "0"
	receptionPeriod           = "0"
	receptionSubperiod        = "0"
	receptionItemBrand        = "0"
	receptionDestLocation     = "DEP1_OS"
	receptionSourceLocation   = "DEP1_OS"
	receptionUnitSales        = "0.0000"
	receptionSalesTotal       = "0.0000"
	receptionStock            = "0.0000"
	receptionDailyAvg         = "0.0000"
	receptionSuggestedPO      = "0.0000"
)

// ReceptionGenerator implementa el TLOG_INVENTORY_RECEPTION usando SQL.
//
// Filtro:
//   - LIEFERSCHEIN.LFS_STATUS = 42
//   - JOIN con LIEFERPOS donde LFP.KST_ID = ? (retail solicitado)
type ReceptionGenerator struct{}

func (ReceptionGenerator) Type() naming.TLOGType { return naming.TLOGReception }

func (ReceptionGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, startCounter int) (*tlog.GenerateResult, error) {
	const candidatesSQL = `
		SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
			INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
			INNER JOIN main.KOSTST K on K.KST_ID = lpo.KST_ID
		WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 37 AND COALESCE(l.LFS_RTS, 0) <> 1
		GROUP BY l.LFS_NAME
		ORDER BY l.LFS_NAME
`

	candidates, err := queryRows(ctx, conn, candidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("reception candidatos: %w", err)
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

		seqNum, err := sequence.Build(h.BusinessDay, sequence.DocReception, startCounter+len(files))
		if err != nil {
			return nil, fmt.Errorf("reception sequence: %w", err)
		}
		x := common.NewXMLBuilder()
		writeReceptionDoc(x, h, retailID, seqNum, lfs, lines)
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

// receptionLines devuelve las líneas de LIEFERPOS de un LFS, con el join a
// ARTIKEL para arrastrar ART_NAME y ART_NUMMER.
//
// NOTA: Si falta algún campo del artículo en el XML, agregarlo al SELECT y
// usarlo en writeReceptionLine.
func receptionLines(ctx context.Context, conn *sql.DB, lfsID string) ([]map[string]string, error) {
	const linesSQL = `
SELECT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       art.ART_NAME, art.ART_NUMMER
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS`
	rows, err := queryRows(ctx, conn, linesSQL, lfsID)
	if err != nil {
		return nil, fmt.Errorf("reception lineas LFS=%s: %w", lfsID, err)
	}
	return rows, nil
}

func writeReceptionDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	lfs map[string]string, lines []map[string]string) {

	state := mapLFSStatusReception(lfs["LFS_STATUS"])
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
	x.Element("WorkstationID", receptionWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", receptionPeriod)
	x.Element("Subperiod", receptionSubperiod)
	x.EmptyElement("PeriodCode")
	x.EmptyElement("SubPeriodCode")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	x.Element("DocumentTypeCode", receptionDocumentTypeCode)
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
		writeReceptionLine(x, line, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeReceptionLine(x *common.XMLBuilder, line map[string]string, detSeq int) {
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
	x.Element("ItemBrand", receptionItemBrand)
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(unitCost))
	x.Element("UnitCount", common.FormatDecimal4(menge))
	x.Element("DestinationLocation", receptionDestLocation)
	x.Element("SourceLocation", receptionSourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(math.Abs(brutto)))
	x.Element("UnitSalesAmount", receptionUnitSales)
	x.Element("SalesTotalAmount", receptionSalesTotal)
	x.Element("Stock", receptionStock)
	x.Element("DailyAverageSales", receptionDailyAvg)
	x.Element("SuggestedPurchaseOrder", receptionSuggestedPO)
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

func mapLFSStatusReception(s string) string {
	v, _ := db.AsInt(s)
	if v == 42 || v == 37 {
		return "4"
	}
	return "7"
}

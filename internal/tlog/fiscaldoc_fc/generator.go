package fiscaldoc_fc

import (
	"fmt"
	"math"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

const (
	WorkstationID    = "0"
	Period           = "0"
	Subperiod        = "0"
	DocumentTypeCode = "InventoryFiscalDoc"
	ReceiptType      = "FC"
	InventoryControlDocumentState = "4"
	FiscalReceiptFlag             = "true"
	ItemBrand                     = "0"
	DestinationLocation           = "DEP1_OS"
	SourceLocation                = "DEP1_OS"
	UnitSalesAmount               = "0.0000"
	SalesTotalAmount              = "0.0000"
	Stock                         = "0.0000"
	DailyAverageSales             = "0.0000"
	SuggestedPurchaseOrder        = "0.0000"
)

// Generator implementa tlog.Generator para InventoryFiscalDoc (FC).
type Generator struct{}

func (Generator) Type() naming.TLOGType { return naming.TLOGFiscalDocFC }

func (Generator) Generate(s *db.Store, h *common.HeaderCtx, kstID string) (*tlog.GenerateResult, error) {
	lfsTable := s.Tables["LIEFERSCHEIN"]
	if lfsTable == nil {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	var candidates []db.Row
	for _, row := range lfsTable.Rows {
		if !filterFC(row) {
			continue
		}
		lfsID := row["LFS_ID"]
		lines := s.LieferposByLFS[lfsID]
		if len(lines) == 0 {
			continue
		}
		if lines[0]["KST_ID"] != kstID {
			continue
		}
		candidates = append(candidates, row)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	x := common.NewXMLBuilder()
	totalDocs, totalLines, seq := 0, 0, 1

	for _, lfs := range candidates {
		lfsID := lfs["LFS_ID"]
		lines := s.LieferposByLFS[lfsID]
		kst := s.Kostst[kstID]
		liefer := s.Liefer[lfs["LF_ID"]]
		retailID := common.FormatRetailStoreID(kst["KST_CODE"])
		seqNum := common.BuildSequenceNumber12(retailID, seq)
		seq++
		totalDocs++
		totalLines += len(lines)
		writeFCDoc(x, h, retailID, seqNum, lfs, liefer, lines, s)
	}

	return &tlog.GenerateResult{XMLContent: x.String(), NumDocs: totalDocs, NumLines: totalLines}, nil
}

func filterFC(row db.Row) bool {
	if row["AKTIV"] != "J" {
		return false
	}
	status, ok := db.AsInt(row["LFS_STATUS"])
	if !ok || status != 42 {
		return false
	}
	rts, _ := db.AsInt(row["LFS_RTS"])
	if rts == 1 {
		return false
	}
	netto, _ := db.AsFloat(row["LFS_NETTO"])
	brutto, _ := db.AsFloat(row["LFS_BRUTTO"])
	return netto > 0 && brutto > 0
}

func writeFCDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	lfs, liefer db.Row, lines []db.Row, s *db.Store) {

	netto, _ := db.AsFloat(lfs["LFS_NETTO"])
	mwst, _ := db.AsFloat(lfs["LFS_MWST"])
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", WorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", Period)
	x.Element("Subperiod", Subperiod)
	x.EmptyElement("PeriodCode")
	x.EmptyElement("SubPeriodCode")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	x.Element("DocumentTypeCode", DocumentTypeCode)
	x.Element("InventoryControlDocumentState", InventoryControlDocumentState)
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
	x.Element("FiscalReceiptFlag", FiscalReceiptFlag)
	x.Element("ReceiptType", ReceiptType)
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
		writeFCLine(x, s, line, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeFCLine(x *common.XMLBuilder, s *db.Store, line db.Row, detSeq int) {
	artRow := s.Artikel[line["ART_NR"]]
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
	x.Element("ItemBrand", ItemBrand)
	x.Element("ItemDescription", artRow["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(unitCost))
	x.Element("UnitCount", common.FormatDecimal4(menge))
	x.Element("DestinationLocation", DestinationLocation)
	x.Element("SourceLocation", SourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(math.Abs(brutto)))
	x.Element("UnitSalesAmount", UnitSalesAmount)
	x.Element("SalesTotalAmount", SalesTotalAmount)
	x.Element("Stock", Stock)
	x.Element("DailyAverageSales", DailyAverageSales)
	x.Element("SuggestedPurchaseOrder", SuggestedPurchaseOrder)
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

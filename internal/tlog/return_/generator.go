package return_

import (
	"fmt"
	"math"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/sequence"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

const (
	WorkstationID    = "0"
	Period           = "0"
	Subperiod        = "0"
	DocumentTypeCode = "InventoryReturn"
	ItemBrand        = "0"
	UnitSalesAmount  = "0.0000"
	SalesTotalAmount = "0.0000"
	Stock            = "0.0000"
	DailyAverageSales      = "0.0000"
	SuggestedPurchaseOrder = "0.0000"
	DestinationLocation    = "DEP1_OS"
	SourceLocation         = "DEP1_OS"
)

// Generator implementa tlog.Generator para InventoryReturn.
type Generator struct{}

func (Generator) Type() naming.TLOGType { return naming.TLOGReturn }

func (Generator) Generate(s *db.Store, h *common.HeaderCtx, kstID string, startCounter int) (*tlog.GenerateResult, error) {
	lfsTable := s.Tables["LIEFERSCHEIN"]
	if lfsTable == nil {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	var candidates []db.Row
	for _, row := range lfsTable.Rows {
		if !filterReturn(row) {
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

	kst := s.Kostst[kstID]
	retailID := common.FormatRetailStoreID(kst["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, lfs := range candidates {
		lfsID := lfs["LFS_ID"]
		lines := s.LieferposByLFS[lfsID]
		liefer := s.Liefer[lfs["LF_ID"]]
		seqNum, err := sequence.Build(h.BusinessDay, sequence.DocReturn, startCounter+len(files))
		if err != nil {
			return nil, fmt.Errorf("return sequence: %w", err)
		}
		x := common.NewXMLBuilder()
		writeReturnDoc(x, h, retailID, seqNum, lfs, liefer, lines, s)
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

func filterReturn(row db.Row) bool {
	if row["AKTIV"] != "J" {
		return false
	}
	rts, _ := db.AsInt(row["LFS_RTS"])
	if rts != 1 {
		return false
	}
	status, ok := db.AsInt(row["LFS_STATUS"])
	if !ok {
		return false
	}
	if status != 37 && status != 42 {
		return false
	}
	brutto, _ := db.AsFloat(row["LFS_BRUTTO"])
	return brutto < 0
}

func writeReturnDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	lfs, liefer db.Row, lines []db.Row, s *db.Store) {

	state := mapLFSStatus(lfs["LFS_STATUS"])
	fiscalFlag := "false"
	if state == "7" {
		fiscalFlag = "true"
	}
	brutto, _ := db.AsFloat(lfs["LFS_BRUTTO"])

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", WorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", Period)
	x.Element("Subperiod", Subperiod)
	x.Element("PeriodCode", "0")
	x.Element("SubPeriodCode", "0")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	x.Element("DocumentTypeCode", DocumentTypeCode)
	x.Element("InventoryControlDocumentState", state)
	x.EmptyElement("contractReferenceNumber")
	x.Element("CreateDateTimestamp", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("DestinationRetailStoreID", retailID)
	x.Element("ExpectedDeliveryDate", h.FormatARTimestamp(h.BeginDateTime))
	x.Element("ICDAmount", common.FormatDecimal4(math.Abs(brutto)))
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
	x.Element("FiscalReceiptFlag", fiscalFlag)
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
		writeReturnLine(x, s, line, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeReturnLine(x *common.XMLBuilder, s *db.Store, line db.Row, detSeq int) {
	artRow := s.Artikel[line["ART_NR"]]
	menge, _ := db.AsFloat(line["LFP_MENGE"])
	ekp, _ := db.AsFloat(line["LFP_EKP"])
	brutto, _ := db.AsFloat(line["LFP_BRUTTO"])
	var unitCost float64
	if menge != 0 {
		unitCost = math.Abs(ekp / menge)
	}

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NR"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID1"]))))
	x.Element("ItemBrand", ItemBrand)
	x.Element("ItemDescription", artRow["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(unitCost))
	x.Element("UnitCount", common.FormatDecimal4(menge)) // viaja con signo original
	x.Element("DestinationLocation", DestinationLocation)
	x.Element("SourceLocation", SourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(brutto)) // viaja con signo original
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

func mapLFSStatus(s string) string {
	v, _ := db.AsInt(s)
	if v == 42 || v == 37 {
		return "4"
	}
	return "7"
}

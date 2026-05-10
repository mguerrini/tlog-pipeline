package count

import (
	"fmt"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/sequence"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

const (
	WorkstationID              = "0"
	Period                     = "0"
	Subperiod                  = "0"
	DocumentTypeCode           = "InventoryCount"
	InventoryAdjustmentType    = "CORRECTIVE_ADJUSTMENT"
	InventoryControlDocumentState = "2"
	ItemBrand                  = "0"
	DestinationLocation        = "DEP1_OS"
	SourceLocation             = "DEP1_OS"
	UnitSalesAmount            = "0.0000"
	SalesTotalAmount           = "0.0000"
	Stock                      = "0.0000"
	DailyAverageSales          = "0.0000"
	SuggestedPurchaseOrder     = "0.0000"
	FiscalReceiptFlag          = "false"
)

// Generator implementa tlog.Generator para InventoryCount.
type Generator struct{}

func (Generator) Type() naming.TLOGType { return naming.TLOGCount }

func (Generator) Generate(s *db.Store, h *common.HeaderCtx, kstID string) (*tlog.GenerateResult, error) {
	invTable := s.Tables["INVENTUR"]
	if invTable == nil {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	var candidates []db.Row
	for _, row := range invTable.Rows {
		if row["KST_ID"] != kstID {
			continue
		}
		status, _ := db.AsInt(row["INV_STATUS"])
		typ, _ := db.AsInt(row["INV_TYP"])
		if status == 8 && typ == 4 {
			candidates = append(candidates, row)
		}
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	kst := s.Kostst[kstID]
	retailID := common.FormatRetailStoreID(kst["KST_CODE"])

	var files []tlog.GeneratedFile
	totalLines := 0

	for _, inv := range candidates {
		invID := inv["INV_ID"]
		lines := s.InvposartByINV[invID]
		if len(lines) == 0 {
			continue
		}
		seqNum, err := sequence.Build(h.BusinessDay, sequence.DocCount, len(files))
		if err != nil {
			return nil, fmt.Errorf("count sequence: %w", err)
		}
		x := common.NewXMLBuilder()
		writeCountDoc(x, h, retailID, seqNum, inv, lines, s)
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
	return &tlog.GenerateResult{Files: files, NumDocs: len(files), NumLines: totalLines}, nil
}

func writeCountDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	inv db.Row, lines []db.Row, s *db.Store) {

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
	x.EmptyElement("ICDAmount")
	x.Element("LastUpdateDate", h.FormatARTimestamp(h.BeginDateTime))
	x.EmptyElement("Originator")
	x.Element("SourceRetailStore", retailID)
	x.EmptyElement("Supplier")
	x.EmptyElement("OrderDocumentType")
	x.Element("User", h.OperatorID)
	x.EmptyElement("ICDQuantity")
	x.EmptyElement("ICDTotSalesAmount")
	x.EmptyElement("Frequency")
	x.Element("InventoryAdjustmentType", InventoryAdjustmentType)
	x.Element("ReceiptNumber", inv["INV_NAME"])
	x.Element("FiscalReceiptFlag", FiscalReceiptFlag)
	x.EmptyElement("ReceiptType")
	x.Element("ReceiptDate", inv["INV_DATUM"])
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
		writeCountLine(x, s, line, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeCountLine(x *common.XMLBuilder, s *db.Store, line db.Row, detSeq int) {
	artRow := s.Artikel[line["ART_ID"]]
	ist, _ := db.AsFloat(line["INP_IST"])
	ekp, _ := db.AsFloat(line["INP_EKP"])
	costTotal := ist * ekp

	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", artRow["ART_NR"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID"]))))
	x.Element("ItemBrand", ItemBrand)
	x.Element("ItemDescription", artRow["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(ekp))
	x.Element("UnitCount", common.FormatDecimal4(ist))
	x.Element("DestinationLocation", DestinationLocation)
	x.Element("SourceLocation", SourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(costTotal))
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

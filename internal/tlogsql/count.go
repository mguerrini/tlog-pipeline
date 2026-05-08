package tlogsql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/sequence"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

const (
	countDocumentTypeCode    = "InventoryCount"
	countInventoryAdjType    = "CORRECTIVE_ADJUSTMENT"
	countInventoryDocState   = "2"
	countFiscalReceiptFlag   = "false"
	countWorkstationID       = "0"
	countPeriod              = "0"
	countSubperiod           = "0"
	countItemBrand           = "0"
	countDestLocation        = "DEP1_OS"
	countSourceLocation      = "DEP1_OS"
	countUnitSales           = "0.0000"
	countSalesTotal          = "0.0000"
	countStock               = "0.0000"
	countDailyAvg            = "0.0000"
	countSuggestedPO         = "0.0000"
)

// CountGenerator implementa TLOG_INVENTORY_COUNT con SQL.
//
// Mismo filtro que Adjustment: KST_ID = ? AND INV_STATUS = 8 AND INV_TYP = 4.
// Difiere solo en el cálculo de costTotal (sin variance) y en algunos
// elementos del XML.
type CountGenerator struct{}

func (CountGenerator) Type() naming.TLOGType { return naming.TLOGCount }

func (CountGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string) (*tlog.GenerateResult, error) {
	const candidatesSQL = `
SELECT *
FROM INVENTUR
WHERE KST_ID = ? AND INV_STATUS = 8 AND INV_TYP = 4
ORDER BY INV_ID`
	candidates, err := queryRows(ctx, conn, candidatesSQL, kstID)
	if err != nil {
		return nil, fmt.Errorf("count candidatos: %w", err)
	}
	if len(candidates) == 0 {
		return &tlog.GenerateResult{Empty: true}, nil
	}

	kst, err := fetchKostst(ctx, conn, kstID)
	if err != nil {
		return nil, err
	}
	retailID := common.FormatRetailStoreID(kst["KST_CODE"])
	seqNum, err := sequence.Build(retailID, h.BusinessDay, sequence.DocCount)
	if err != nil {
		return nil, fmt.Errorf("count sequence: %w", err)
	}

	x := common.NewXMLBuilder()
	totalDocs, totalLines := 0, 0

	for _, inv := range candidates {
		lines, err := invposartLines(ctx, conn, inv["INV_ID"])
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		totalDocs++
		totalLines += len(lines)
		writeCountDoc(x, h, retailID, seqNum, inv, lines)
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

func writeCountDoc(x *common.XMLBuilder, h *common.HeaderCtx, retailID, seqNum string,
	inv map[string]string, lines []map[string]string) {

	x.Open("Transaction")
	x.Element("RetailStoreID", retailID)
	x.Element("WorkstationID", countWorkstationID)
	x.Element("SequenceNumber", seqNum)
	x.Element("BusinessDayDate", h.FormatBusinessDayDate())
	x.Element("Period", countPeriod)
	x.Element("Subperiod", countSubperiod)
	x.EmptyElement("PeriodCode")
	x.EmptyElement("SubPeriodCode")
	x.Element("BeginDateTime", h.FormatBeginDateTime())
	x.Element("EndDateTime", h.FormatEndDateTime())
	x.Element("OperatorID", h.OperatorID)
	x.EmptyElement("OriginalTransaction")

	x.Open("InventoryControlTransaction")
	x.Element("SerialFormID", seqNum)
	x.Element("DocumentTypeCode", countDocumentTypeCode)
	x.Element("InventoryControlDocumentState", countInventoryDocState)
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
	x.Element("InventoryAdjustmentType", countInventoryAdjType)
	x.Element("ReceiptNumber", inv["INV_NAME"])
	x.Element("FiscalReceiptFlag", countFiscalReceiptFlag)
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
		writeCountLine(x, line, i+1)
	}
	x.Close()
	x.Close()
	x.Close()
}

func writeCountLine(x *common.XMLBuilder, line map[string]string, detSeq int) {
	ist, _ := db.AsFloat(line["INP_IST"])
	ekp, _ := db.AsFloat(line["INP_EKP"])
	costTotal := ist * ekp

	// Como en adjustment, el original usa artRow["ART_NR"] (no presente en
	// el schema SQLite). Mismo comportamiento (queda vacío en SQL).
	x.Open("inventoryControlDocumentMerchandiseLineItem")
	x.Element("DetSequenceNumber", fmt.Sprintf("%d", detSeq))
	x.Element("Item", line["ART_NR"])
	x.Element("UomUnits", common.FormatDecimal4(float64(db.MustAsInt(line["VPK_ID"]))))
	x.Element("ItemBrand", countItemBrand)
	x.Element("ItemDescription", line["ART_NAME"])
	x.Element("UnitBaseCostAmount", common.FormatDecimal4(ekp))
	x.Element("UnitCount", common.FormatDecimal4(ist))
	x.Element("DestinationLocation", countDestLocation)
	x.Element("SourceLocation", countSourceLocation)
	x.Element("CostTotalAmount", common.FormatDecimal4(costTotal))
	x.Element("UnitSalesAmount", countUnitSales)
	x.Element("SalesTotalAmount", countSalesTotal)
	x.Element("Stock", countStock)
	x.Element("DailyAverageSales", countDailyAvg)
	x.Element("SuggestedPurchaseOrder", countSuggestedPO)
	x.EmptyElement("PickupCode")
	x.EmptyElement("LastUpdateDate")
	x.EmptyElement("DifBME_ASNTypeID")
	x.Close()
}

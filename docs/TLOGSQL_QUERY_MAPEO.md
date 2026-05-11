# Queries y Mapeo de Campos — tlogsql

Documento generado desde el código fuente de `internal/tlogsql/`.
Muestra el SQL ejecutado y cómo cada campo XML se obtiene.

---

## TLOG_INVENTORY_RECEPTION

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,
       L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM
FROM LIEFERSCHEIN l
    INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
    INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
    INNER JOIN KOSTST K ON K.KST_ID = lpo.KST_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 37 AND COALESCE(l.LFS_RTS, 0) <> 1
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Query Items** (por cada `LFS_ID` del driver)
```sql
SELECT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       art.ART_NAME, art.ART_NUMMER
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDay, DocReception, contador)
<BusinessDayDate>                  Calculado = fecha del nombre del archivo (YYYYMMDD)
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<BeginDateTime>                    Calculado = BusinessDayDate + BEGIN_DATE_OFFSET (config)
<EndDateTime>                      Calculado = BusinessDayDate + END_DATE_OFFSET (config)
<OperatorID>                       Calculado = config.process.operator_id
<SerialFormID>                     Calculado = igual que SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryReception"
<InventoryControlDocumentState>    Calculado = si LFS_STATUS IN (42, 37) → "4", sino → "7"
<CreateDateTimestamp>              Calculado = FormatARTimestamp(BeginDateTime)
<DestinationRetailStoreID>         Query Driver = KST_CODE
<ExpectedDeliveryDate>             Calculado = FormatARTimestamp(BeginDateTime)
<ICDAmount>                        Calculado = ABS(LFS_BRUTTO) con 4 decimales
<LastUpdateDate>                   Calculado = FormatARTimestamp(BeginDateTime)
<SourceRetailStore>                Query Driver = KST_CODE
<Supplier>                         Query Driver = LF_VERT
<User>                             Calculado = config.process.operator_id
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Calculado = si InventoryControlDocumentState == "7" → "true", sino → "false"
<ReceiptDate>                      Query Driver = LFS_DATUM (parse "2006-01-02 15:04:05", FormatARTimestamp)
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NR
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGE (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = ABS(LFP_BRUTTO) con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

---

## TLOG_INVENTORY_RETURN

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,
       L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
       l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST
FROM LIEFERSCHEIN l
    INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
    INNER JOIN KOSTST K ON lpo.KST_ID1 = K.KST_ID
    INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS IN (37, 42)
    AND l.LFS_BRUTTO < 0 AND COALESCE(l.LFS_RTS, 0) = 1
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Query Items** (mismo que Reception, por cada `LFS_ID` del driver)
```sql
SELECT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       art.ART_NAME, art.ART_NUMMER
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDay, DocReturn, contador)
<BusinessDayDate>                  Calculado = fecha del nombre del archivo (YYYYMMDD)
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<BeginDateTime>                    Calculado = BusinessDayDate + BEGIN_DATE_OFFSET (config)
<EndDateTime>                      Calculado = BusinessDayDate + END_DATE_OFFSET (config)
<OperatorID>                       Calculado = config.process.operator_id
<SerialFormID>                     Calculado = igual que SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryReturn"
<InventoryControlDocumentState>    Calculado = si LFS_STATUS IN (42, 37) → "4", sino → "7"
<CreateDateTimestamp>              Calculado = FormatARTimestamp(BeginDateTime)
<DestinationRetailStoreID>         Query Driver = KST_CODE
<ExpectedDeliveryDate>             Calculado = FormatARTimestamp(BeginDateTime)
<ICDAmount>                        Calculado = ABS(LFS_BRUTTO) con 4 decimales
<LastUpdateDate>                   Calculado = FormatARTimestamp(BeginDateTime)
<SourceRetailStore>                Query Driver = KST_CODE
<Supplier>                         Query Driver = LF_VERT
<User>                             Calculado = config.process.operator_id
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Calculado = si InventoryControlDocumentState == "7" → "true", sino → "false"
<ReceiptDate>                      Query Driver = LFS_DATUM (parse "2006-01-02 15:04:05", FormatARTimestamp)
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NR
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = ABS(LFP_EKP / LFP_MENGE) (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE (viaja con signo original, negativo)
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Query Items = LFP_BRUTTO (viaja con signo original, negativo)
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

---

## TLOG_INVENTORY_FISCAL_DOC FC

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,
       L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
       l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST
FROM LIEFERSCHEIN l
    INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
    INNER JOIN KOSTST K ON lpo.KST_ID1 = K.KST_ID
    INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 42
    AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_NETTO > 0 AND l.LFS_BRUTTO > 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Query Items** (mismo que Reception, por cada `LFS_ID` del driver)
```sql
SELECT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       art.ART_NAME, art.ART_NUMMER
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDay, DocFiscalDocFC, contador)
<BusinessDayDate>                  Calculado = fecha del nombre del archivo (YYYYMMDD)
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<BeginDateTime>                    Calculado = BusinessDayDate + BEGIN_DATE_OFFSET (config)
<EndDateTime>                      Calculado = BusinessDayDate + END_DATE_OFFSET (config)
<OperatorID>                       Calculado = config.process.operator_id
<SerialFormID>                     Calculado = igual que SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryFiscalDoc"
<InventoryControlDocumentState>    Valor Fijo = "4"
<CreateDateTimestamp>              Calculado = FormatARTimestamp(BeginDateTime)
<DestinationRetailStoreID>         Query Driver = KST_CODE
<ExpectedDeliveryDate>             Calculado = FormatARTimestamp(BeginDateTime)
<ICDAmount>                        Query Driver = LFS_BRUTTO con 4 decimales
<LastUpdateDate>                   Calculado = FormatARTimestamp(BeginDateTime)
<SourceRetailStore>                Query Driver = KST_CODE
<Supplier>                         Query Driver = LF_VERT
<User>                             Calculado = config.process.operator_id
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Valor Fijo = "true"
<ReceiptType>                      Valor Fijo = "FC"
<ReceiptDate>                      Query Driver = LFS_DATUM (parse "2006-01-02 15:04:05", FormatARTimestamp)
<NetAmount>                        Query Driver = LFS_NETTO con 4 decimales
<ExemptAmout>                      Valor Fijo = "0.0000"
<TaxAmount>                        Valor Fijo = "0.0000"
<VatAmount>                        Query Driver = LFS_MWST con 4 decimales
<ServicesVATAmount>                Valor Fijo = "0.0000"
<DifferencialVATAmount>            Valor Fijo = "0.0000"
<IvaTaxAmount>                     Valor Fijo = "0.0000"
<IIBBTaxAmount>                    Valor Fijo = "0.0000"
<TotalAmount>                      Query Driver = LFS_BRUTTO con 4 decimales
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NR
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGE (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = ABS(LFP_BRUTTO) con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

---

## TLOG_INVENTORY_FISCAL_DOC NC

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,
       L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
       l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST
FROM LIEFERSCHEIN l
    INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
    INNER JOIN KOSTST K ON lpo.KST_ID1 = K.KST_ID
    INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 42
    AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_NETTO < 0 AND l.LFS_BRUTTO < 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Query Items** (mismo que Reception, por cada `LFS_ID` del driver)
```sql
SELECT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       art.ART_NAME, art.ART_NUMMER
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDay, DocFiscalDocNC, contador)
<BusinessDayDate>                  Calculado = fecha del nombre del archivo (YYYYMMDD)
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<BeginDateTime>                    Calculado = BusinessDayDate + BEGIN_DATE_OFFSET (config)
<EndDateTime>                      Calculado = BusinessDayDate + END_DATE_OFFSET (config)
<OperatorID>                       Calculado = config.process.operator_id
<SerialFormID>                     Calculado = igual que SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryFiscalDoc"
<InventoryControlDocumentState>    Valor Fijo = "4"
<CreateDateTimestamp>              Calculado = FormatARTimestamp(BeginDateTime)
<DestinationRetailStoreID>         Query Driver = KST_CODE
<ExpectedDeliveryDate>             Calculado = FormatARTimestamp(BeginDateTime)
<ICDAmount>                        Query Driver = LFS_BRUTTO con 4 decimales (negativo)
<LastUpdateDate>                   Calculado = FormatARTimestamp(BeginDateTime)
<SourceRetailStore>                Query Driver = KST_CODE
<Supplier>                         Query Driver = LF_VERT
<User>                             Calculado = config.process.operator_id
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Valor Fijo = "true"
<ReceiptType>                      Valor Fijo = "NC"
<ReceiptDate>                      Query Driver = LFS_DATUM (parse "2006-01-02 15:04:05", FormatARTimestamp)
<NetAmount>                        Query Driver = LFS_NETTO con 4 decimales (negativo)
<ExemptAmout>                      Valor Fijo = "0.0000"
<TaxAmount>                        Valor Fijo = "0.0000"
<VatAmount>                        Query Driver = LFS_MWST con 4 decimales (negativo)
<ServicesVATAmount>                Valor Fijo = "0.0000"
<DifferencialVATAmount>            Valor Fijo = "0.0000"
<IvaTaxAmount>                     Valor Fijo = "0.0000"
<IIBBTaxAmount>                    Valor Fijo = "0.0000"
<TotalAmount>                      Query Driver = LFS_BRUTTO con 4 decimales (negativo)
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NR
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGE (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE (viaja con signo original, negativo)
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Query Items = LFP_BRUTTO (viaja con signo original, negativo)
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

---

## TLOG_INVENTORY_ADJUSTMENT

**Query Driver**
```sql
SELECT DISTINCT I.INV_ID, K.KST_CODE, I.INV_NAME, I.CHG_ZEIT
FROM INVENTUR I
    INNER JOIN KOSTST K ON I.KST_ID = K.KST_ID
WHERE I.KST_ID = ? AND I.INV_STATUS = 8 AND I.INV_TYP = 4
ORDER BY I.INV_ID
```

**Query Items** (por cada `INV_ID` del driver)
```sql
SELECT inv.INV_ID, inv.ART_ID, inv.VPK_ID, inv.INP_IST, inv.INP_SOLL,
       inv.INP_EKP, inv.INP_VKP,
       art.ART_NUMMER, art.ART_NAME
FROM INVPOSART inv
LEFT JOIN ARTIKEL art ON art.ART_ID = inv.ART_ID
WHERE inv.INV_ID = ?
ORDER BY inv.ART_ID
```

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDay, DocAdjustment, contador)
<BusinessDayDate>                  Calculado = fecha del nombre del archivo (YYYYMMDD)
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<BeginDateTime>                    Calculado = BusinessDayDate + BEGIN_DATE_OFFSET (config)
<EndDateTime>                      Calculado = BusinessDayDate + END_DATE_OFFSET (config)
<OperatorID>                       Calculado = config.process.operator_id
<SerialFormID>                     Calculado = igual que SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryAdjustment"
<InventoryControlDocumentState>    Valor Fijo = "2"
<CreateDateTimestamp>              Query Driver = CHG_ZEIT (parse "2006-01-02 15:04:05", FormatARTimestamp; fallback BeginDateTime)
<DestinationRetailStoreID>         Query Driver = KST_CODE
<ExpectedDeliveryDate>             Calculado = FormatARTimestamp(BeginDateTime)
<LastUpdateDate>                   Calculado = FormatARTimestamp(BeginDateTime)
<SourceRetailStore>                Query Driver = KST_CODE
<User>                             Calculado = config.process.operator_id
<InventoryAdjustmentType>          Valor Fijo = "CORRECTIVE_ADJUSTMENT"
<ReceiptNumber>                    Query Driver = INV_NAME
<FiscalReceiptFlag>                Valor Fijo = "false"
<ReceiptDate>                      Calculado = FormatARTimestamp(BeginDateTime)
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NR (campo inexistente en INVPOSART → siempre vacío)
<UomUnits>                Calculado = CAST(VPK_ID AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Query Items = INP_EKP con 4 decimales
<UnitCount>               Calculado = INP_IST - INP_SOLL (varianza)
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = ABS((INP_IST - INP_SOLL) * INP_EKP) con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

---

## TLOG_INVENTORY_COUNT

**Query Driver**
```sql
SELECT DISTINCT I.INV_ID, K.KST_CODE, I.INV_NAME, I.CHG_ZEIT, I.INV_DATUM
FROM INVENTUR I
    INNER JOIN KOSTST K ON I.KST_ID = K.KST_ID
WHERE I.KST_ID = ? AND I.INV_STATUS = 8 AND I.INV_TYP = 4
ORDER BY I.INV_ID
```

**Query Items** (mismo que Adjustment, por cada `INV_ID` del driver)
```sql
SELECT inv.INV_ID, inv.ART_ID, inv.VPK_ID, inv.INP_IST, inv.INP_SOLL,
       inv.INP_EKP, inv.INP_VKP,
       art.ART_NUMMER, art.ART_NAME
FROM INVPOSART inv
LEFT JOIN ARTIKEL art ON art.ART_ID = inv.ART_ID
WHERE inv.INV_ID = ?
ORDER BY inv.ART_ID
```

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDay, DocCount, contador)
<BusinessDayDate>                  Calculado = fecha del nombre del archivo (YYYYMMDD)
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<BeginDateTime>                    Calculado = BusinessDayDate + BEGIN_DATE_OFFSET (config)
<EndDateTime>                      Calculado = BusinessDayDate + END_DATE_OFFSET (config)
<OperatorID>                       Calculado = config.process.operator_id
<SerialFormID>                     Calculado = igual que SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryCount"
<InventoryControlDocumentState>    Valor Fijo = "2"
<CreateDateTimestamp>              Query Driver = CHG_ZEIT (parse "2006-01-02 15:04:05", FormatARTimestamp; fallback BeginDateTime)
<DestinationRetailStoreID>         Query Driver = KST_CODE
<ExpectedDeliveryDate>             Calculado = FormatARTimestamp(BeginDateTime)
<LastUpdateDate>                   Calculado = FormatARTimestamp(BeginDateTime)
<SourceRetailStore>                Query Driver = KST_CODE
<User>                             Calculado = config.process.operator_id
<InventoryAdjustmentType>          Valor Fijo = "CORRECTIVE_ADJUSTMENT"
<ReceiptNumber>                    Query Driver = INV_NAME
<FiscalReceiptFlag>                Valor Fijo = "false"
<ReceiptDate>                      Query Driver = INV_DATUM (sin parseo, se usa el string directo)
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NR (campo inexistente en INVPOSART → siempre vacío)
<UomUnits>                Calculado = CAST(VPK_ID AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Query Items = INP_EKP con 4 decimales
<UnitCount>               Query Items = INP_IST (stock contado, sin restar SOLL)
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = INP_IST * INP_EKP con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

---

## TLOG_BUSINESS_EOD (Cierre)

El cierre no sigue el patrón driver/items. Genera un único documento por retail con todos los artículos del día.

**Query KOSTST** (para obtener datos del retail)
```sql
SELECT * FROM KOSTST WHERE KST_ID = ?
```
Campos usados: `KST_CODE` (RetailStoreID), `KST_LOCID` (LOCATION_CODE en cada item).

**Query Items**
```sql
SELECT dt.KST_ID, dt.ART_ID, dt.DAY_DATE,
       dt.DAY_SOHBEG, dt.DAY_SOHEND, dt.DAY_SOHINV,
       dt.DAY_QTYPURCH, dt.DAY_QTYTRSFIN, dt.DAY_QTYTRSFOUT,
       dt.DAY_QTYUSAGE, dt.DAY_QTYSOLD, dt.DAY_QTYINV,
       art.ART_NUMMER, art.ART_NAME
FROM DAILYTOTALS dt
LEFT JOIN ARTIKEL art ON art.ART_ID = dt.ART_ID
WHERE dt.KST_ID = ?
ORDER BY dt.ART_ID
```

**Mapeo — Cabecera (`<Transaction>`)**

```
<RETAILSTOREID>     Query KOSTST = KST_CODE
<WORKSTATIONID>     Valor Fijo = "0"
<SEQUENCENUMBER>    Calculado = sequence.Build(BusinessDay, DocCierre, contador)
<BUSINESSDAYDATE>   Calculado = fecha del nombre del archivo (YYYYMMDD)
<BEGINDATETIME>     Calculado = BusinessDayDate + BEGIN_DATE_OFFSET (config)
<ENDDATETIME>       Calculado = BusinessDayDate + END_DATE_OFFSET (config)
<OPERATORID>        Calculado = config.process.operator_id
<PERIODO>           Valor Fijo = "0"
<SUBPERIOD>         Valor Fijo = "0"
<PERIODCODE>        Valor Fijo = ""
<SUBPERIODCODE>     Valor Fijo = ""
<TYPECODE>          Valor Fijo = "BusinessEOD"
<TYPEID>            Valor Fijo = "63"
```

**Mapeo — Item (`<Item>` dentro de `<ItemList>`)**

```
<STOCK_SEQ_NUMBER>           Valor Fijo = "1"
<LOCATION_CODE>              Query KOSTST = KST_LOCID
<REVENUE_CENTER>             Valor Fijo = "RCD"
<ITEM_INVENTORY_STATE>       Valor Fijo = "OnSale"
<ITEM_SEQ_NUMBER>            Calculado = índice 1..N por documento
<ITEM_CODE>                  Query Items = ART_NUMMER (fallback ART_ID si ART_NUMMER vacío)
<BEGIN_UNIT_COUNT>           Query Items = DAY_SOHBEG con 4 decimales
<GROSS_SALE_UNIT_COUNT>      Query Items = DAY_QTYSOLD con 4 decimales
<RETURN_UNIT_COUNT>          Valor Fijo = "0.0000"
<RECEIVED_UNIT_COUNT>        Query Items = DAY_QTYPURCH con 4 decimales
<RETURN_TO_VENTOR_UNIT_COUNT> Valor Fijo = "0.0000"
<TRANSFERIN_UNIT_COUNT>      Query Items = DAY_QTYTRSFIN con 4 decimales
<TRANSFEROUT_UNIT_COUNT>     Query Items = DAY_QTYTRSFOUT con 4 decimales
<ADJUSTMENTIN_UNIT_COUNT>    Query Items = DAY_QTYUSAGE con 4 decimales
<ADJUSTMENTOUT_UNIT_COUNT>   Query Items = DAY_QTYINV con 4 decimales
<CURRENT_UNIT_COUNT>         Query Items = DAY_SOHINV con 4 decimales
```

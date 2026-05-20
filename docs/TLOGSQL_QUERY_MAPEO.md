# Queries y Mapeo de Campos — tlogsql

Documento generado desde el código fuente de `internal/tlogsql/`.
Muestra el SQL ejecutado y cómo cada campo XML se obtiene.

**Convenciones:**
- `Valor Fijo = "X"` — constante hardcodeada o valor de configuración conocido.
- `Calculado = Igual al Tag XXX` — mismo valor que otro tag ya definido en el documento.
- `Calculado = Igual al Tag XXX con formato "..."` — mismo origen pero distinto formato de fecha.
- `Calculado = fórmula` — valor derivado mediante operación aritmética o condicional.
- `Query Driver = CAMPO` — valor tomado directamente del resultado del Query Driver.
- `Query Items = CAMPO` — valor tomado directamente del resultado del Query Items.

---

## TLOG_INVENTORY_RECEPTION

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM
FROM LIEFERSCHEIN l
         INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
         INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
         INNER JOIN main.KOSTST K on K.KST_ID = lpo.KST_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS IN (37, 42) AND COALESCE(l.LFS_RTS, 0) <> 1 AND l.LFS_BRUTTO > 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Cambios Query Driver**

A COMPLETAR

**Query Items** (por cada `LFS_ID` del driver)
```sql
SELECT DISTINCT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       lfp.LFP_HACCPINFO, lfp.LFP_ABLAUFDT,
       art.ART_NAME, art.ART_NUMMER,
       art.ART_NR AS ART_ART_NR, art.ART_MWSTNR
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Cambios Query Items**

A COMPLETAR

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocReception, contador)
<BusinessDayDate>                  Calculado = Fecha del nombre del archivo con formato "YYYY-MM-DD"
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<PeriodCode>                       Valor Fijo = "0"
<SubPeriodCode>                    Valor Fijo = "0"
<BeginDateTime>                    Calculado = (BusinessDayDate - 1 día) + config.process.begin_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<EndDateTime>                      Calculado = BusinessDayDate + config.process.end_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<OperatorID>                       Valor Fijo = config.process.operator_id
<SerialFormID>                     Calculado = Igual al Tag SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryReception"
<InventoryControlDocumentState>    Calculado = si LFS_STATUS = 42 → "4"; si LFS_STATUS = 37 → "7"
<CreateDateTimestamp>              Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<DestinationRetailStoreID>         Calculado = Igual al Tag RetailStoreID
<ExpectedDeliveryDate>             Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<ICDAmount>                        Calculado = ABS(LFS_BRUTTO) con 4 decimales
<LastUpdateDate>                   Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<SourceRetailStore>                Calculado = Igual al Tag RetailStoreID
<Supplier>                         Query Driver = LF_VERT
<User>                             Calculado = Igual al Tag OperatorID
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Calculado = si InventoryControlDocumentState = "7" → "true"; sino → "false"
<ReceiptDate>                      Query Driver = LFS_DATUM con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
```

***NOTA:***
InventoryControlDocumentState es siempre 7 porque el filtro del query es l.LFS_STATUS = 37

**Cambios Mapeo Cabecera**

```
<RetailStoreID>                    
<WorkstationID>                    
<SequenceNumber>                   
<BusinessDayDate>                  
<Period>                           
<Subperiod>                        
<PeriodCode>                       
<SubPeriodCode>                    
<BeginDateTime>                    
<EndDateTime>                      
<OperatorID>                       
<SerialFormID>                     
<DocumentTypeCode>                 
<InventoryControlDocumentState>    
<CreateDateTimestamp>              
<DestinationRetailStoreID>         
<ExpectedDeliveryDate>             
<ICDAmount>                        
<LastUpdateDate>                   
<SourceRetailStore>                
<Supplier>                         
<User>                             
<ReceiptNumber>                    
<FiscalReceiptFlag>                
<ReceiptDate>                      
```


**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<RetailStoreID>           Calculado = Igual al Tag RetailStoreID
<WorkstationID>           Calculado = Igual al Tag WorkstationID
<SequenceNumber>          Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NUMMER (join ARTIKEL)
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGE con 4 decimales (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE con 4 decimales
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = ABS(LFP_BRUTTO) con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

**Cambios Mapeo Detalle**

```
<RetailStoreID>           
<WorkstationID>           
<SequenceNumber>          
<DetSequenceNumber>       
<Item>                    
<UomUnits>                
<ItemBrand>               
<ItemDescription>         
<UnitBaseCostAmount>      
<UnitCount>               
<DestinationLocation>     
<SourceLocation>          
<CostTotalAmount>         
<UnitSalesAmount>         
<SalesTotalAmount>        
<Stock>                   
<DailyAverageSales>       
<SuggestedPurchaseOrder>  
```

---

## TLOG_INVENTORY_RETURN

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
                l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST
FROM LIEFERSCHEIN l
         INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
         INNER JOIN main.KOSTST K ON lpo.KST_ID1 = K.KST_ID
         INNER JOIN main.LIEFER L2 ON lpo.LF_ID = L2.LF_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS IN (37, 42) AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_BRUTTO < 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Cambios Query Driver**

A COMPLETAR

**Query Items** (mismo que Reception, por cada `LFS_ID` del driver)
```sql
SELECT DISTINCT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       lfp.LFP_HACCPINFO, lfp.LFP_ABLAUFDT,
       art.ART_NAME, art.ART_NUMMER,
       art.ART_NR AS ART_ART_NR, art.ART_MWSTNR
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Cambios Query Items**

A COMPLETAR

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocReturn, contador)
<BusinessDayDate>                  Calculado = Fecha del nombre del archivo con formato "YYYY-MM-DD"
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<PeriodCode>                       Valor Fijo = "0"
<SubPeriodCode>                    Valor Fijo = "0"
<BeginDateTime>                    Calculado = (BusinessDayDate - 1 día) + config.process.begin_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<EndDateTime>                      Calculado = BusinessDayDate + config.process.end_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<OperatorID>                       Valor Fijo = config.process.operator_id
<SerialFormID>                     Calculado = Igual al Tag SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryReturn"
<InventoryControlDocumentState>    Calculado = si LFS_STATUS = 42 → "4"; si LFS_STATUS = 37 → "7"
<CreateDateTimestamp>              Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<DestinationRetailStoreID>         Calculado = Igual al Tag RetailStoreID
<ExpectedDeliveryDate>             Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<ICDAmount>                        Calculado = ABS(LFS_BRUTTO) con 4 decimales
<LastUpdateDate>                   Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<SourceRetailStore>                Calculado = Igual al Tag RetailStoreID
<Supplier>                         Query Driver = LF_VERT
<User>                             Calculado = Igual al Tag OperatorID
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Calculado = si InventoryControlDocumentState = "7" → "true"; sino → "false"
<ReceiptDate>                      Query Driver = LFS_DATUM con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
```

**Cambios Mapeo Cabecera**

```
<RetailStoreID>                    
<WorkstationID>                    
<SequenceNumber>                   
<BusinessDayDate>                  
<Period>                           
<Subperiod>                        
<PeriodCode>                       
<SubPeriodCode>                    
<BeginDateTime>                    
<EndDateTime>                      
<OperatorID>                       
<SerialFormID>                     
<DocumentTypeCode>                 
<InventoryControlDocumentState>    
<CreateDateTimestamp>              
<DestinationRetailStoreID>         
<ExpectedDeliveryDate>             
<ICDAmount>                        
<LastUpdateDate>                   
<SourceRetailStore>                
<Supplier>                         
<User>                             
<ReceiptNumber>                    
<FiscalReceiptFlag>                
<ReceiptDate>                      
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<RetailStoreID>           Calculado = Igual al Tag RetailStoreID
<WorkstationID>           Calculado = Igual al Tag WorkstationID
<SequenceNumber>          Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NUMMER (join ARTIKEL)
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = ABS(LFP_EKP / LFP_MENGE) con 4 decimales (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE con 4 decimales (viaja con signo original, negativo)
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Query Items = LFP_BRUTTO con 4 decimales (viaja con signo original, negativo)
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

**Cambios Mapeo Detalle**

```
<RetailStoreID>           
<WorkstationID>           
<SequenceNumber>          
<DetSequenceNumber>       
<Item>                    
<UomUnits>                
<ItemBrand>               
<ItemDescription>         
<UnitBaseCostAmount>      
<UnitCount>               
<DestinationLocation>     
<SourceLocation>          
<CostTotalAmount>         
<UnitSalesAmount>         
<SalesTotalAmount>        
<Stock>                   
<DailyAverageSales>       
<SuggestedPurchaseOrder>  
```

---

## TLOG_INVENTORY_FISCAL_DOC FC

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,
       L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
       l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST, L2.LF_SACHB
FROM LIEFERSCHEIN l
    INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
    INNER JOIN KOSTST K ON lpo.KST_ID1 = K.KST_ID
    INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 42
    AND COALESCE(l.LFS_RTS, 0) <> 1 AND l.LFS_NETTO > 0 AND l.LFS_BRUTTO > 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Cambios Query Driver**

A COMPLETAR

**Query Items** (mismo que Reception, por cada `LFS_ID` del driver)
```sql
SELECT DISTINCT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       lfp.LFP_HACCPINFO, lfp.LFP_ABLAUFDT,
       art.ART_NAME, art.ART_NUMMER,
       art.ART_NR AS ART_ART_NR, art.ART_MWSTNR
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Cambios Query Items**

A COMPLETAR

> Las líneas con `ART_ART_NR` IN (`"1120"`, `"1100"`, `"1098"`, `"1096"`) se usan para calcular
> campos de cabecera (CAI, impuestos) y se **excluyen del detalle**.

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocFiscalDocFC, contador)
<BusinessDayDate>                  Calculado = Fecha del nombre del archivo con formato "YYYY-MM-DD"
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<PeriodCode>                       Valor Fijo = "0"
<SubPeriodCode>                    Valor Fijo = "0"
<BeginDateTime>                    Calculado = (BusinessDayDate - 1 día) + config.process.begin_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<EndDateTime>                      Calculado = BusinessDayDate + config.process.end_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<OperatorID>                       Valor Fijo = config.process.operator_id
<SerialFormID>                     Calculado = Igual al Tag SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryFiscalDoc"
<InventoryControlDocumentState>    Valor Fijo = "4"
<CreateDateTimestamp>              Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<DestinationRetailStoreID>         Calculado = Igual al Tag RetailStoreID
<ExpectedDeliveryDate>             Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<ICDAmount>                        Query Driver = LFS_BRUTTO con 4 decimales
<LastUpdateDate>                   Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<SourceRetailStore>                Calculado = Igual al Tag RetailStoreID
<Supplier>                         Query Driver = LF_SACHB
<User>                             Calculado = Igual al Tag OperatorID
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Valor Fijo = "true"
<ReceiptType>                      Valor Fijo = "FC"
<ReceiptDate>                      Query Driver = LFS_DATUM con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<CAINumber>                        Query Items = LFP_HACCPINFO de línea con ART_ART_NR = "1120"; si no existe → vacío
<CAIDate>                          Query Items = LFP_ABLAUFDT de línea con ART_ART_NR = "1120" con formato "YYYY-MM-DD HH:MM:SS.000"; si no existe → vacío
<NetAmount>                        Query Driver = LFS_NETTO con 4 decimales
<ExemptAmout>                      Calculado = Suma de LFP_EKP de líneas con ART_MWSTNR = 0 con 4 decimales
<TaxAmount>                        Query Items = LFP_EKP de línea con ART_ART_NR = "1100" con 4 decimales
<VatAmount>                        Calculado = Suma de LFP_EKP de líneas con ART_MWSTNR ≠ 0 con 4 decimales
<ServicesVATAmount>                Valor Fijo = "0.0000"
<DifferencialVATAmount>            Valor Fijo = "0.0000"
<IvaTaxAmount>                     Query Items = LFP_EKP de línea con ART_ART_NR = "1098" con 4 decimales
<IIBBTaxAmount>                    Query Items = LFP_EKP de línea con ART_ART_NR = "1096" con 4 decimales
<TotalAmount>                      Query Driver = LFS_BRUTTO con 4 decimales
```

**Cambios Mapeo Cabecera**

```
<RetailStoreID>                    
<WorkstationID>                    
<SequenceNumber>                   
<BusinessDayDate>                  
<Period>                           
<Subperiod>                        
<PeriodCode>                       
<SubPeriodCode>                    
<BeginDateTime>                    
<EndDateTime>                      
<OperatorID>                       
<SerialFormID>                     
<DocumentTypeCode>                 
<InventoryControlDocumentState>    
<CreateDateTimestamp>              
<DestinationRetailStoreID>         
<ExpectedDeliveryDate>             
<ICDAmount>                        
<LastUpdateDate>                   
<SourceRetailStore>                
<Supplier>                         
<User>                             
<ReceiptNumber>                    
<FiscalReceiptFlag>                
<ReceiptType>                      
<ReceiptDate>                      
<CAINumber>                        
<CAIDate>                          
<NetAmount>                        
<ExemptAmout>                      
<TaxAmount>                        
<VatAmount>                        
<ServicesVATAmount>                
<DifferencialVATAmount>            
<IvaTaxAmount>                     
<IIBBTaxAmount>                    
<TotalAmount>                      
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

> Se excluyen líneas con `ART_ART_NR` IN (`"1120"`, `"1100"`, `"1098"`, `"1096"`).

```
<RetailStoreID>           Calculado = Igual al Tag RetailStoreID
<WorkstationID>           Calculado = Igual al Tag WorkstationID
<SequenceNumber>          Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>       Calculado = índice 1..N (excluye líneas especiales de impuestos/CAI)
<Item>                    Query Items = ART_NUMMER (join ARTIKEL)
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGE con 4 decimales (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE con 4 decimales
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = ABS(LFP_BRUTTO) con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

**Cambios Mapeo Detalle**

```
<RetailStoreID>           
<WorkstationID>           
<SequenceNumber>          
<DetSequenceNumber>       
<Item>                    
<UomUnits>                
<ItemBrand>               
<ItemDescription>         
<UnitBaseCostAmount>      
<UnitCount>               
<DestinationLocation>     
<SourceLocation>          
<CostTotalAmount>         
<UnitSalesAmount>         
<SalesTotalAmount>        
<Stock>                   
<DailyAverageSales>       
<SuggestedPurchaseOrder>  
```

---

## TLOG_INVENTORY_FISCAL_DOC NC

**Query Driver**
```sql
SELECT DISTINCT l.LFS_ID, K.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,
       L2.LF_VERT, l.LFS_NAME, l.LFS_DATUM,
       l.LFS_INFO, l.LFS_NETTO, l.LFS_MWST, L2.LF_SACHB
FROM LIEFERSCHEIN l
    INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
    INNER JOIN KOSTST K ON lpo.KST_ID1 = K.KST_ID
    INNER JOIN LIEFER L2 ON lpo.LF_ID = L2.LF_ID
WHERE lpo.KST_ID = ? AND l.LFS_STATUS = 42
    AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_NETTO < 0 AND l.LFS_BRUTTO < 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

**Cambios Query Driver**

A COMPLETAR

**Query Items** (mismo que Reception, por cada `LFS_ID` del driver)
```sql
SELECT DISTINCT lfp.LFS_ID, lfp.LFP_POS, lfp.ART_NR, lfp.LFP_MENGE,
       lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
       lfp.LFP_HACCPINFO, lfp.LFP_ABLAUFDT,
       art.ART_NAME, art.ART_NUMMER,
       art.ART_NR AS ART_ART_NR, art.ART_MWSTNR
FROM LIEFERPOS lfp
LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.LFS_ID = ?
ORDER BY lfp.LFP_POS
```

**Cambios Query Items**

A COMPLETAR

> A diferencia de FC, NC **incluye todas las líneas en el detalle** (sin filtro por ART_ART_NR).
> Los campos de importe (CAI, Tax, etc.) se calculan igualmente desde las líneas especiales.

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocFiscalDocNC, contador)
<BusinessDayDate>                  Calculado = Fecha del nombre del archivo con formato "YYYY-MM-DD"
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<PeriodCode>                       Valor Fijo = "0"
<SubPeriodCode>                    Valor Fijo = "0"
<BeginDateTime>                    Calculado = (BusinessDayDate - 1 día) + config.process.begin_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<EndDateTime>                      Calculado = BusinessDayDate + config.process.end_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<OperatorID>                       Valor Fijo = config.process.operator_id
<SerialFormID>                     Calculado = Igual al Tag SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryFiscalDoc"
<InventoryControlDocumentState>    Valor Fijo = "4"
<CreateDateTimestamp>              Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<DestinationRetailStoreID>         Calculado = Igual al Tag RetailStoreID
<ExpectedDeliveryDate>             Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<ICDAmount>                        Query Driver = LFS_BRUTTO con 4 decimales (negativo)
<LastUpdateDate>                   Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<SourceRetailStore>                Calculado = Igual al Tag RetailStoreID
<Supplier>                         Query Driver = LF_SACHB
<User>                             Calculado = Igual al Tag OperatorID
<ReceiptNumber>                    Query Driver = LFS_NAME
<FiscalReceiptFlag>                Valor Fijo = "true"
<ReceiptType>                      Valor Fijo = "NC"
<ReceiptDate>                      Query Driver = LFS_DATUM con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<CAINumber>                        Query Items = LFP_HACCPINFO de línea con ART_ART_NR = "1120"; si no existe → vacío
<CAIDate>                          Query Items = LFP_ABLAUFDT de línea con ART_ART_NR = "1120" con formato "YYYY-MM-DD HH:MM:SS.000"; si no existe → vacío
<NetAmount>                        Query Driver = LFS_NETTO con 4 decimales (negativo)
<ExemptAmout>                      Calculado = Suma de LFP_EKP de líneas con ART_MWSTNR = 0 con 4 decimales
<TaxAmount>                        Query Items = LFP_EKP de línea con ART_ART_NR = "1100" con 4 decimales
<VatAmount>                        Calculado = Suma de LFP_EKP de líneas con ART_MWSTNR ≠ 0 con 4 decimales
<ServicesVATAmount>                Valor Fijo = "0.0000"
<DifferencialVATAmount>            Valor Fijo = "0.0000"
<IvaTaxAmount>                     Query Items = LFP_EKP de línea con ART_ART_NR = "1098" con 4 decimales
<IIBBTaxAmount>                    Query Items = LFP_EKP de línea con ART_ART_NR = "1096" con 4 decimales
<TotalAmount>                      Query Driver = LFS_BRUTTO con 4 decimales (negativo)
```

**Cambios Mapeo Cabecera**

```
<RetailStoreID>                    
<WorkstationID>                    
<SequenceNumber>                   
<BusinessDayDate>                  
<Period>                           
<Subperiod>                        
<PeriodCode>                       
<SubPeriodCode>                    
<BeginDateTime>                    
<EndDateTime>                      
<OperatorID>                       
<SerialFormID>                     
<DocumentTypeCode>                 
<InventoryControlDocumentState>    
<CreateDateTimestamp>              
<DestinationRetailStoreID>         
<ExpectedDeliveryDate>             
<ICDAmount>                        
<LastUpdateDate>                   
<SourceRetailStore>                
<Supplier>                         
<User>                             
<ReceiptNumber>                    
<FiscalReceiptFlag>                
<ReceiptType>                      
<ReceiptDate>                      
<CAINumber>                        
<CAIDate>                          
<NetAmount>                        
<ExemptAmout>                      
<TaxAmount>                        
<VatAmount>                        
<ServicesVATAmount>                
<DifferencialVATAmount>            
<IvaTaxAmount>                     
<IIBBTaxAmount>                    
<TotalAmount>                      
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

> ⚠️ NC usa `lfp.ART_NR` (clave numérica de LIEFERPOS) para `<Item>`, no `ART_NUMMER` de ARTIKEL.
> Todas las líneas se incluyen en el detalle.

```
<RetailStoreID>           Calculado = Igual al Tag RetailStoreID
<WorkstationID>           Calculado = Igual al Tag WorkstationID
<SequenceNumber>          Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NR (lfp.ART_NR de LIEFERPOS, no join ARTIKEL)
<UomUnits>                Calculado = CAST(VPK_ID1 AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGE con 4 decimales (0 si LFP_MENGE = 0)
<UnitCount>               Query Items = LFP_MENGE con 4 decimales (negativo)
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Query Items = LFP_BRUTTO con 4 decimales (negativo)
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

**Cambios Mapeo Detalle**

```
<RetailStoreID>           
<WorkstationID>           
<SequenceNumber>          
<DetSequenceNumber>       
<Item>                    
<UomUnits>                
<ItemBrand>               
<ItemDescription>         
<UnitBaseCostAmount>      
<UnitCount>               
<DestinationLocation>     
<SourceLocation>          
<CostTotalAmount>         
<UnitSalesAmount>         
<SalesTotalAmount>        
<Stock>                   
<DailyAverageSales>       
<SuggestedPurchaseOrder>  
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

**Cambios Query Driver**

A COMPLETAR

**Query Items** (por cada `INV_ID` del driver)
```sql
SELECT DISTINCT inv.INV_ID, inv.ART_ID, inv.VPK_ID, inv.INP_IST, inv.INP_SOLL,
       inv.INP_EKP, inv.INP_VKP,
       art.ART_NUMMER, art.ART_NAME, art.ART_NR, art.CHG_ZEIT
FROM INVPOSART inv
LEFT JOIN ARTIKEL art ON art.ART_ID = inv.ART_ID
WHERE inv.INV_ID = ?
ORDER BY inv.ART_ID
```

**Cambios Query Items**

A COMPLETAR

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocAdjustment, contador)
<BusinessDayDate>                  Calculado = Fecha del nombre del archivo con formato "YYYY-MM-DD"
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<PeriodCode>                       Valor Fijo = "0"
<SubPeriodCode>                    Valor Fijo = "0"
<BeginDateTime>                    Calculado = (BusinessDayDate - 1 día) + config.process.begin_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<EndDateTime>                      Calculado = BusinessDayDate + config.process.end_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<OperatorID>                       Valor Fijo = config.process.operator_id
<SerialFormID>                     Calculado = Igual al Tag SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryAdjustment"
<InventoryControlDocumentState>    Valor Fijo = "2"
<contractReferenceNumber>          Valor Fijo = "Generado desde la Web"
<CreateDateTimestamp>              Query Driver = CHG_ZEIT con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<DestinationRetailStoreID>         Calculado = Igual al Tag RetailStoreID
<ExpectedDeliveryDate>             Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<LastUpdateDate>                   Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<SourceRetailStore>                Calculado = Igual al Tag RetailStoreID
<User>                             Calculado = Igual al Tag OperatorID
<InventoryAdjustmentType>          Valor Fijo = "CORRECTIVE_ADJUSTMENT"
<FiscalReceiptFlag>                Valor Fijo = "false"
<ReceiptDate>                      Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
```

**Cambios Mapeo Cabecera**

```
<RetailStoreID>                    
<WorkstationID>                    
<SequenceNumber>                   
<BusinessDayDate>                  
<Period>                           
<Subperiod>                        
<PeriodCode>                       
<SubPeriodCode>                    
<BeginDateTime>                    
<EndDateTime>                      
<OperatorID>                       
<SerialFormID>                     
<DocumentTypeCode>                 
<InventoryControlDocumentState>    
<contractReferenceNumber>          
<CreateDateTimestamp>              
<DestinationRetailStoreID>         
<ExpectedDeliveryDate>             
<LastUpdateDate>                   
<SourceRetailStore>                
<User>                             
<InventoryAdjustmentType>          
<FiscalReceiptFlag>                
<ReceiptDate>                      
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<RetailStoreID>           Calculado = Igual al Tag RetailStoreID
<WorkstationID>           Calculado = Igual al Tag WorkstationID
<SequenceNumber>          Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NUMMER (join ARTIKEL)
<UomUnits>                Calculado = CAST(VPK_ID AS INT) con 4 decimales
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Query Items = INP_EKP con 4 decimales
<UnitCount>               Calculado = INP_IST - INP_SOLL (varianza) con 4 decimales
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = ABS((INP_IST - INP_SOLL) * INP_EKP) con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Query Items = INP_IST con 4 decimales
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

**Cambios Mapeo Detalle**

```
<RetailStoreID>           
<WorkstationID>           
<SequenceNumber>          
<DetSequenceNumber>       
<Item>                    
<UomUnits>                
<ItemDescription>         
<UnitBaseCostAmount>      
<UnitCount>               
<DestinationLocation>     
<SourceLocation>          
<CostTotalAmount>         
<UnitSalesAmount>         
<SalesTotalAmount>        
<Stock>                   
<DailyAverageSales>       
<SuggestedPurchaseOrder>  
```

---

## TLOG_INVENTORY_COUNT

**Query Driver**
```sql
SELECT DISTINCT ????
FROM HisVerbrauch h
WHERE h.KST_ID = ? 
ORDER BY h.VBR_NAME
```

**Cambios Query Driver**

A COMPLETAR

**Query Items**
```sql
SELECT DISTINCT ??
FROM HisVerbrauch h
INNER JOIN HisVerbrauchPos p ON h.VBR_ID = p.VBR_ID
WHERE h.KST_ID = ?
ORDER BY h.NEW_ZEIT
```

**Cambios Query Items**

A COMPLETAR

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocCount, contador)
<BusinessDayDate>                  Calculado = Fecha del nombre del archivo con formato "YYYY-MM-DD"
<Period>                           Valor Fijo = "0"
<Subperiod>                        Valor Fijo = "0"
<PeriodCode>                       Valor Fijo = "0"
<SubPeriodCode>                    Valor Fijo = "0"
<BeginDateTime>                    Calculado = (BusinessDayDate - 1 día) + config.process.begin_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<EndDateTime>                      Calculado = BusinessDayDate + config.process.end_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<OperatorID>                       Valor Fijo = config.process.operator_id
<SerialFormID>                     Calculado = Igual al Tag SequenceNumber
<DocumentTypeCode>                 Valor Fijo = "InventoryCount"
<InventoryControlDocumentState>    Valor Fijo = "4"
<CreateDateTimestamp>              Query Driver = HisVerbrauch.CHG_ZEIT con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<DestinationRetailStoreID>         Calculado = Igual al Tag RetailStoreID
<ExpectedDeliveryDate>             Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<LastUpdateDate>                   Calculado = Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<SourceRetailStore>                Calculado = Igual al Tag RetailStoreID
<User>                             Calculado = Igual al Tag OperatorID
<InventoryAdjustmentType>          Valor Fijo = "CORRECTIVE_ADJUSTMENT"
<ReceiptNumber>                    Query Driver = INV_NAME
<FiscalReceiptFlag>                Valor Fijo = "false"
<ReceiptDate>                      Query Driver = INV_DATUM (string sin formato adicional)
```

**Cambios Mapeo Cabecera**

```
<RetailStoreID>                    
<WorkstationID>                    
<SequenceNumber>                   
<BusinessDayDate>                  
<Period>                           
<Subperiod>                        
<PeriodCode>                       
<SubPeriodCode>                    
<BeginDateTime>                    
<EndDateTime>                      
<OperatorID>                       
<SerialFormID>                     
<DocumentTypeCode>                 
<InventoryControlDocumentState>    
<CreateDateTimestamp>              
<DestinationRetailStoreID>         
<ExpectedDeliveryDate>             
<LastUpdateDate>                   
<SourceRetailStore>                
<User>                             
<InventoryAdjustmentType>          
<ReceiptNumber>                    
<FiscalReceiptFlag>                
<ReceiptDate>                      
```

**Mapeo — Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)**

```
<RetailStoreID>           Calculado = Igual al Tag RetailStoreID
<WorkstationID>           Calculado = Igual al Tag WorkstationID
<SequenceNumber>          Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ART_NUMMER (join ARTIKEL)
<UomUnits>                Calculado = CAST(VPK_ID AS INT) con 4 decimales
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (join ARTIKEL)
<UnitBaseCostAmount>      Query Items = INP_EKP con 4 decimales
<UnitCount>               Query Items = INP_IST (stock contado) con 4 decimales
<DestinationLocation>     Valor Fijo = "DEP1_OS"
<SourceLocation>          Valor Fijo = "DEP1_OS"
<CostTotalAmount>         Calculado = INP_IST * INP_EKP con 4 decimales
<UnitSalesAmount>         Valor Fijo = "0.0000"
<SalesTotalAmount>        Valor Fijo = "0.0000"
<Stock>                   Valor Fijo = "0.0000"
<DailyAverageSales>       Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>  Valor Fijo = "0.0000"
```

**Cambios Mapeo Detalle**

```
<RetailStoreID>           
<WorkstationID>           
<SequenceNumber>          
<DetSequenceNumber>       
<Item>                    
<UomUnits>                
<ItemBrand>               
<ItemDescription>         
<UnitBaseCostAmount>      
<UnitCount>               
<DestinationLocation>     
<SourceLocation>          
<CostTotalAmount>         
<UnitSalesAmount>         
<SalesTotalAmount>        
<Stock>                   
<DailyAverageSales>       
<SuggestedPurchaseOrder>  
```

---

## TLOG_BUSINESS_EOD (Cierre)

El cierre no sigue el patrón driver/items. Genera un único documento por retail con todos los artículos del día.

**Query KOSTST** (para obtener datos del retail)
```sql
SELECT * FROM KOSTST WHERE KST_ID = ?
```
Campos usados: `KST_CODE` → RetailStoreID; `KST_LOCID` → LOCATION_CODE en cada item.

**Cambios Query KOSTST**

A COMPLETAR

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

**Cambios Query Items**

A COMPLETAR

**Mapeo — Cabecera (`<Transaction>`)**

```
<RETAILSTOREID>     Query KOSTST = KST_CODE (5 dígitos con ceros a la izquierda)
<WORKSTATIONID>     Valor Fijo = "0"
<SEQUENCENUMBER>    Calculado = sequence.Build(BusinessDayDate, DocCierre, contador)
<BUSINESSDAYDATE>   Calculado = Fecha del nombre del archivo con formato "YYYY-MM-DD"
<BEGINDATETIME>     Calculado = (BusinessDayDate - 1 día) + config.process.begin_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<ENDDATETIME>       Calculado = BusinessDayDate + config.process.end_date_offset con formato "YYYY-MM-DD HH:MM:SS"
<OPERATORID>        Valor Fijo = config.process.operator_id
<PERIODO>           Valor Fijo = "0"
<SUBPERIOD>         Valor Fijo = "0"
<TYPECODE>          Valor Fijo = "BusinessEOD"
<TYPEID>            Valor Fijo = "63"
```

**Cambios Mapeo Cabecera**

```
<RETAILSTOREID>     
<WORKSTATIONID>     
<SEQUENCENUMBER>    
<BUSINESSDAYDATE>   
<BEGINDATETIME>     
<ENDDATETIME>       
<OPERATORID>        
<PERIODO>           
<SUBPERIOD>         
<TYPECODE>          
<TYPEID>            
```

**Mapeo — Item (`<Item>` dentro de `<ItemList>`)**

```
<STOCK_SEQ_NUMBER>            Valor Fijo = "1"
<LOCATION_CODE>               Query KOSTST = KST_LOCID
<REVENUE_CENTER>              Valor Fijo = "RCD"
<ITEM_INVENTORY_STATE>        Valor Fijo = "OnSale"
<ITEM_SEQ_NUMBER>             Calculado = índice 1..N por documento
<ITEM_CODE>                   Query Items = ART_NUMMER; si vacío → ART_ID (fallback)
<BEGIN_UNIT_COUNT>            Query Items = DAY_SOHBEG con 4 decimales
<GROSS_SALE_UNIT_COUNT>       Query Items = DAY_QTYSOLD con 4 decimales
<RETURN_UNIT_COUNT>           Valor Fijo = "0.0000"
<RECEIVED_UNIT_COUNT>         Query Items = DAY_QTYPURCH con 4 decimales
<RETURN_TO_VENTOR_UNIT_COUNT> Valor Fijo = "0.0000"
<TRANSFERIN_UNIT_COUNT>       Query Items = DAY_QTYTRSFIN con 4 decimales
<TRANSFEROUT_UNIT_COUNT>      Query Items = DAY_QTYTRSFOUT con 4 decimales
<ADJUSTMENTIN_UNIT_COUNT>     Query Items = DAY_QTYUSAGE con 4 decimales
<ADJUSTMENTOUT_UNIT_COUNT>    Query Items = DAY_QTYINV con 4 decimales
<CURRENT_UNIT_COUNT>          Query Items = DAY_SOHEND con 4 decimales
```

**Cambios Mapeo Item**

```
<STOCK_SEQ_NUMBER>            
<LOCATION_CODE>               
<REVENUE_CENTER>              
<ITEM_INVENTORY_STATE>        
<ITEM_SEQ_NUMBER>             
<ITEM_CODE>                   
<BEGIN_UNIT_COUNT>            
<GROSS_SALE_UNIT_COUNT>       
<RETURN_UNIT_COUNT>           
<RECEIVED_UNIT_COUNT>         
<RETURN_TO_VENTOR_UNIT_COUNT> 
<TRANSFERIN_UNIT_COUNT>       
<TRANSFEROUT_UNIT_COUNT>      
<ADJUSTMENTIN_UNIT_COUNT>     
<ADJUSTMENTOUT_UNIT_COUNT>    
<CURRENT_UNIT_COUNT>          
```

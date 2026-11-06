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
SELECT DISTINCT l.LFS_ID, k.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, l.LFS_NAME, l.LFS_DATUM, l.LF_SACHB
FROM LIEFERSCHEIN_VIEW l
         INNER JOIN KOSTST k on k.KST_ID = l.KST_ID
WHERE l.KST_ID = ? AND l.LFS_STATUS = 42 AND COALESCE(l.LFS_RTS, 0) <> 1 AND l.LFS_BRUTTO > 0
      AND l.LF_ID <> 90327
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME
```

> `LIEFERSCHEIN` y `LIEFERPOS` fueron consolidadas en la vista única `LIEFERSCHEIN_VIEW`:
> `KST_ID` y el proveedor (`LF_SACHB`) ya viven en la propia vista, sin joins contra `LIEFER`/`LIEFERPOS`.

**Cambios Query Driver**

A COMPLETAR

**Query Items** (por cada `LFS_ID` del driver)
```sql
SELECT DISTINCT lfp.ART_NR, lfp.LFS_ID, lfp.ART_NR, lfp.LFP_MENGE, lfp.LFP_MENGEGE,
                lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
                lfp.LFP_HACCPINFO, lfp.LFP_ABLAUFDT,
                lfp.ART_NAME, lfp.ART_MUMMER,
                lfp.ART_MUSTNR, AIC.ITEM_CODE
FROM LIEFERSCHEIN_VIEW lfp
         INNER JOIN ART_ITEM_CODE AIC on lfp.ART_MUMMER = AIC.ART_NUMMER
WHERE lfp.LFS_ID = ? and lfp.ART_NR not in (2204, 2205,2206, 2207,2255,2256, 5923)
ORDER BY lfp.LFS_NAME
```

> `LIEFERSCHEIN_VIEW` renombró `ART_NUMMER` → `ART_MUMMER` y `ART_MWSTNR` → `ART_MUSTNR`, y agregó `LFP_MENGEGE`
> (cantidad en UoM secundaria; se usa para `UnitCount` y el costo unitario). El `ART_NUMMER` de `ART_ITEM_CODE`
> (tabla aparte, sin cambios) sigue llamándose igual.

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
<Supplier>                         Query Driver = LF_SACHB
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
<Item>                    Query Items = ITEM_CODE (join ART_ITEM_CODE sobre lfp.ART_MUMMER = AIC.ART_NUMMER)
<UomUnits>                Valor Fijo = "1"
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (LIEFERSCHEIN_VIEW)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGEGE con 4 decimales (0 si LFP_MENGEGE = 0)
<UnitCount>               Query Items = LFP_MENGEGE con 4 decimales
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
SELECT DISTINCT l.LFS_ID, k.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, l.LFS_NAME, l.LFS_DATUM,
                l.LFS_NETTO, l.LFS_MWST, l.LF_SACHB, AIC.ITEM_CODE
FROM LIEFERSCHEIN_VIEW l
         INNER JOIN KOSTST K ON l.KST_ID = k.KST_ID
		 INNER JOIN ART_ITEM_CODE AIC on l.ART_MUMMER = AIC.ART_NUMMER
WHERE l.KST_ID = ? AND l.LFS_STATUS = 42 AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_BRUTTO < 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME;
```

> Igual que en Reception: `LIEFERSCHEIN`/`LIEFERPOS` → `LIEFERSCHEIN_VIEW`; `KST_ID` vive en la vista y el
> proveedor sale de `LF_SACHB`. El driver ahora también resuelve `ITEM_CODE` contra `ART_ITEM_CODE` usando
> `l.ART_MUMMER` (antes `l.ART_NUMMER`).

**Cambios Query Driver**

A COMPLETAR

**Query Items** (por cada `LFS_ID` del driver)
```sql
SELECT DISTINCT lfp.ART_NR, lfp.LFS_ID, lfp.ART_NR, lfp.LFP_MENGE, lfp.LFP_MENGEGE,
                lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1,
                lfp.LFP_HACCPINFO, lfp.LFP_ABLAUFDT,
                lfp.ART_NAME, lfp.ART_MUMMER,
                lfp.ART_MUSTNR
FROM LIEFERSCHEIN_VIEW lfp
WHERE lfp.LFS_ID = ? and lfp.ART_NR not in (2204, 2205,2206, 2207,2255,2256, 5923)
ORDER BY lfp.LFS_NAME
```

> `ART_NUMMER` → `ART_MUMMER`, `ART_MWSTNR` → `ART_MUSTNR` (igual que Reception). Este query de detalle no
> vuelve a traer `ITEM_CODE`; ese valor sólo está disponible en el resultado del Query Driver.

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
<Supplier>                         Query Driver = LF_SACHB
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
<Item>                    Query Items = ITEM_CODE ⚠ el SELECT de detalle no trae esta columna (sólo el Query Driver la resuelve vía ART_ITEM_CODE); queda vacío en el XML tal como está el código hoy
<UomUnits>                Valor Fijo = "1"
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (LIEFERSCHEIN_VIEW)
<UnitBaseCostAmount>      Calculado = ABS(LFP_EKP / LFP_MENGEGE) con 4 decimales (0 si LFP_MENGEGE = 0)
<UnitCount>               Query Items = LFP_MENGEGE con 4 decimales (viaja con signo original, negativo)
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
SELECT DISTINCT l.RNG_NAME, l.LFS_ID, k.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO, l.LFS_NAME, l.LFS_DATUM,
                l.LFS_NETTO, l.LFS_MWST, l.LF_SACHB
FROM LIEFERSCHEIN_VIEW l
	INNER JOIN KOSTST k ON l.KST_ID = K.KST_ID
WHERE l.KST_ID = ? AND l.LFS_STATUS = 42
  AND COALESCE(l.LFS_RTS, 0) <> 1 AND l.LFS_NETTO > 0 AND l.LFS_BRUTTO > 0
  AND l.RNG_COD <> 1 AND l.LF_ID <> 90327
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME;
```

> `LIEFERSCHEIN`/`LIEFERPOS` → `LIEFERSCHEIN_VIEW`. El driver ahora agrupa los `LFS_ID` por `RNG_NAME`
> (talonario/rango fiscal): todos los `LFS_ID` del mismo `RNG_NAME` comparten un `SequenceNumber` y van al
> mismo archivo XML — por eso `RNG_NAME` entra al SELECT. El proveedor sale de `LF_SACHB` directo de la vista.

**Cambios Query Driver**

A COMPLETAR

**Query Items** (por cada `RNG_NAME` del driver)
```sql
SELECT distinct lfp.ART_NR, lfp.LFS_ID, lfp.ART_NR, lfp.LFP_MENGE, lfp.LFP_MENGEGE,
				lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1, lfp.ART_NAME,
				lfp.ART_MUMMER, lfp.ART_MUSTNR, AIC.ITEM_CODE
FROM LIEFERSCHEIN_VIEW lfp
	 INNER JOIN ART_ITEM_CODE AIC on lfp.ART_MUMMER = AIC.ART_NUMMER
WHERE lfp.RNG_NAME = ? and lfp.ART_NR not in (2204, 2205,2206, 2207, 2255, 2256, 5923)
ORDER BY lfp.ART_NR, lfp.LFS_ID;
```

> El Query Items filtra por `RNG_NAME` (no por `LFS_ID` individual) y excluye directamente los `ART_NR`
> especiales de impuestos/CAI (`2204, 2205, 2206, 2207, 2255, 2256, 5923`) — esas líneas no viajan en el
> detalle. `ART_NUMMER` → `ART_MUMMER`, `ART_MWSTNR` → `ART_MUSTNR`; se agrega `LFP_MENGEGE`.
> Este query es compartido con FISCAL_DOC NC (`fiscalDocReceptionLines`).

**Cambios Query Items**

A COMPLETAR

> Los montos de cabecera (CAI, impuestos) **no** se calculan sobre estas líneas con códigos de texto
> (`ART_ART_NR`); se resuelven con queries auxiliares aparte, ver abajo.

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocFiscalDocFC, contador) — uno por RNG_NAME
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
<ReceiptNumber>                    Query Driver = RNG_NAME (no LFS_NAME)
<FiscalReceiptFlag>                Valor Fijo = "false"
<ReceiptType>                      Valor Fijo = "FC"
<ReceiptDate>                      Query Driver = LFS_DATUM con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<CAINumber>                        Query auxiliar (LIEFERSCHEIN_VIEW.LFP_HACCPINFO) de la línea con ART_NR = 2207 y RNG_NAME/LFS_STATUS=42; si no existe → vacío
<CAIDate>                          Query auxiliar (LIEFERSCHEIN_VIEW.LFP_ABLAUFDT) de la línea con ART_NR = 2207, con formato "YYYY-MM-DD HH:MM:SS.000"; si no existe → vacío
<NetAmount>                        Calculado = LFS_NETTO - TaxAmount - VatAmount - IvaTaxAmount - IIBBTaxAmount - DifferencialVATAmount, con 4 decimales
<ExemptAmout>                      Query auxiliar = Suma de LFP_EKP de líneas con ART_MUSTNR = 0 y ART_NR fuera de (2204,2205,2206,2207,2255,2256,5923), con 4 decimales
<TaxAmount>                        Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2204 + Suma de LFP_MENGE con ART_NR = 5923, con 4 decimales
<VatAmount>                        Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2255, con 4 decimales
<ServicesVATAmount>                Valor Fijo = "0.0000"
<DifferencialVATAmount>            Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2256, con 4 decimales
<IvaTaxAmount>                     Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2205, con 4 decimales
<IIBBTaxAmount>                    Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2206, con 4 decimales
<TotalAmount>                      Query Driver = LFS_BRUTTO con 4 decimales
```

> ⚠️ Lo anterior reemplaza la descripción previa (basada en `ART_ART_NR` con códigos de texto
> `"1120"/"1100"/"1098"/"1096"` y `LFP_EKP`): el código actual (`queryFiscalDocHeaderData` en
> `fiscaldoc_common.go`) resuelve estos montos con queries auxiliares por `ART_NR` numérico sobre
> `LIEFERSCHEIN_VIEW`, sumando `LFP_MENGE` (no `LFP_EKP`) salvo en `ExemptAmout`.

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
<DetSequenceNumber>       Calculado = índice 1..N (las líneas especiales de impuestos/CAI ya vienen excluidas por el WHERE del Query Items)
<Item>                    Query Items = ITEM_CODE (join ART_ITEM_CODE sobre lfp.ART_MUMMER = AIC.ART_NUMMER)
<UomUnits>                Valor Fijo = "1"
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (LIEFERSCHEIN_VIEW)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGEGE con 4 decimales (0 si LFP_MENGEGE = 0)
<UnitCount>               Query Items = LFP_MENGEGE con 4 decimales
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
SELECT DISTINCT l.RNG_NAME, l.LFS_ID, k.KST_CODE, l.LFS_STATUS, l.LFS_BRUTTO,  l.LFS_NAME, l.LFS_DATUM,
                l.LFS_NETTO, l.LFS_MWST, l.LF_SACHB
FROM LIEFERSCHEIN_VIEW l
         INNER JOIN KOSTST k ON l.KST_ID = k.KST_ID
WHERE l.KST_ID = ? AND l.LFS_STATUS = 42
  AND COALESCE(l.LFS_RTS, 0) = 1 AND l.LFS_NETTO < 0 AND l.LFS_BRUTTO < 0
GROUP BY l.LFS_NAME
ORDER BY l.LFS_NAME;
```

> Igual que FC: `LIEFERSCHEIN`/`LIEFERPOS` → `LIEFERSCHEIN_VIEW`, agrupado por `RNG_NAME`.

**Cambios Query Driver**

A COMPLETAR

**Query Items** (por cada `RNG_NAME` del driver — es el mismo query que FC, `fiscalDocReceptionLines`)
```sql
SELECT distinct lfp.ART_NR, lfp.LFS_ID, lfp.ART_NR, lfp.LFP_MENGE, lfp.LFP_MENGEGE,
				lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1, lfp.ART_NAME,
				lfp.ART_MUMMER, lfp.ART_MUSTNR, AIC.ITEM_CODE
FROM LIEFERSCHEIN_VIEW lfp
	 INNER JOIN ART_ITEM_CODE AIC on lfp.ART_MUMMER = AIC.ART_NUMMER
WHERE lfp.RNG_NAME = ? and lfp.ART_NR not in (2204, 2205,2206, 2207, 2255, 2256, 5923)
ORDER BY lfp.ART_NR, lfp.LFS_ID;
```

> ⚠️ A diferencia de lo documentado antes, el código actual **no** diferencia FC/NC en este query: ambos
> usan `fiscalDocReceptionLines` y excluyen los mismos `ART_NR` especiales por `WHERE`, no por post-filtro.
> `ART_NUMMER` → `ART_MUMMER`, `ART_MWSTNR` → `ART_MUSTNR`; se agrega `LFP_MENGEGE`.

**Cambios Query Items**

A COMPLETAR

**Mapeo — Cabecera (`<Transaction>` / `<InventoryControlTransaction>`)**

```
<RetailStoreID>                    Query Driver = KST_CODE (5 dígitos con ceros a la izquierda)
<WorkstationID>                    Valor Fijo = "0"
<SequenceNumber>                   Calculado = sequence.Build(BusinessDayDate, DocFiscalDocNC, contador) — uno por RNG_NAME
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
<ReceiptNumber>                    Query Driver = RNG_NAME (no LFS_NAME)
<FiscalReceiptFlag>                Valor Fijo = "false"
<ReceiptType>                      Valor Fijo = "NC"
<ReceiptDate>                      Query Driver = LFS_DATUM con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<CAINumber>                        Query auxiliar (LIEFERSCHEIN_VIEW.LFP_HACCPINFO) de la línea con ART_NR = 2207 y LFS_STATUS=42; si no existe → vacío
<CAIDate>                          Query auxiliar (LIEFERSCHEIN_VIEW.LFP_ABLAUFDT) de la línea con ART_NR = 2207, con formato "YYYY-MM-DD HH:MM:SS.000"; si no existe → vacío
<NetAmount>                        Calculado = LFS_NETTO - TaxAmount - VatAmount - IvaTaxAmount - IIBBTaxAmount - DifferencialVATAmount, con 4 decimales (negativo)
<ExemptAmout>                      Query auxiliar = Suma de LFP_EKP de líneas con ART_MUSTNR = 0 y ART_NR fuera de (2204,2205,2206,2207,2255,2256,5923), con 4 decimales
<TaxAmount>                        Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2204 + Suma de LFP_MENGE con ART_NR = 5923, con 4 decimales
<VatAmount>                        Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2255, con 4 decimales
<ServicesVATAmount>                Valor Fijo = "0.0000"
<DifferencialVATAmount>            Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2256, con 4 decimales
<IvaTaxAmount>                     Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2205, con 4 decimales
<IIBBTaxAmount>                    Query auxiliar = Suma de LFP_MENGE de líneas con ART_NR = 2206, con 4 decimales
<TotalAmount>                      Query Driver = LFS_BRUTTO con 4 decimales (negativo)
```

> ⚠️ Las queries auxiliares (CAI, Tax/Vat/Iva/IIBB) son exactamente las mismas que para FC
> (`queryFiscalDocHeaderData`, compartida entre ambos generadores) — ver la nota en esa sección.

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

> ⚠️ El código actual **ya no** usa `lfp.ART_NR` directo para `<Item>`: usa `ITEM_CODE` del join contra
> `ART_ITEM_CODE`, igual que FC (ver nota de Query Items más arriba).

```
<RetailStoreID>           Calculado = Igual al Tag RetailStoreID
<WorkstationID>           Calculado = Igual al Tag WorkstationID
<SequenceNumber>          Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>       Calculado = índice 1..N por documento
<Item>                    Query Items = ITEM_CODE (join ART_ITEM_CODE sobre lfp.ART_MUMMER = AIC.ART_NUMMER)
<UomUnits>                Valor Fijo = "1"
<ItemBrand>               Valor Fijo = "0"
<ItemDescription>         Query Items = ART_NAME (LIEFERSCHEIN_VIEW)
<UnitBaseCostAmount>      Calculado = LFP_EKP / LFP_MENGEGE con 4 decimales (0 si LFP_MENGEGE = 0)
<UnitCount>               Query Items = LFP_MENGEGE con 4 decimales (negativo)
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

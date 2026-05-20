# Queries y Mapeo de Campos — TLOG_INVENTORY_COUNT

Documento generado desde el código fuente de `internal/tlogsql/count.go`.
Muestra el SQL ejecutado y cómo cada campo XML se obtiene.

**Convenciones:**
- `Valor Fijo = "X"` — constante hardcodeada o valor de configuración conocido.
- `Calculado = Igual al Tag XXX` — mismo valor que otro tag ya definido en el documento.
- `Calculado = Igual al Tag XXX con formato "..."` — mismo origen pero distinto formato de fecha.
- `Calculado = fórmula` — valor derivado mediante operación aritmética o condicional.
- `Query Driver = CAMPO` — valor tomado directamente del resultado del Query Driver.
- `Query Items = CAMPO` — valor tomado directamente del resultado del Query Items.
- `Vacío` — elemento XML presente pero sin contenido (`<Tag></Tag>`).

---

## TLOG_INVENTORY_COUNT

**Query Driver**
```sql
SELECT V.VBR_ID, V.VBR_NAME, V.VRT_ID, V.CHG_ZEIT,
       K.KST_CODE
FROM HIS_VERBRAUCH V
    INNER JOIN KOSTST K ON V.KST_ID = K.KST_ID
WHERE V.KST_ID = ? AND V.VBR_STATUS = 2
ORDER BY V.VBR_ID
```

**Cambios Query Driver**

A COMPLETAR

**Query Items** (por cada `VBR_ID` del driver)
```sql
SELECT p.VBR_ID, p.VBT_POS, p.ART_NR, p.VBT_MENGE, p.VBT_WES, p.VPK_NR,
       a.ART_NUMMER, a.ART_NAME
FROM HIS_VERBRAUCHPOS p
    LEFT JOIN ARTIKEL a ON a.ART_ID = p.ART_NR
WHERE p.VBR_ID = ?
ORDER BY p.VBT_POS
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
<contractReferenceNumber>          Query Driver = VBR_NAME
<CreateDateTimestamp>              Query Driver = CHG_ZEIT con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<DestinationRetailStoreID>         Calculado = Igual al Tag RetailStoreID
<ExpectedDeliveryDate>             Calculado = fecha de CHG_ZEIT (solo fecha, hora fija "00:00:00.000") con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<ICDAmount>                        Calculado = SUM(VBT_MENGE × VBT_WES) de todas las líneas del documento con 4 decimales
<LastUpdateDate>                   Calculado = Igual al Tag CreateDateTimestamp
<SourceRetailStore>                Calculado = Igual al Tag RetailStoreID
<Supplier>                         Vacío
<OrderDocumentType>                Vacío
<User>                             Calculado = Igual al Tag OperatorID
<ICDQuantity>                      Vacío
<ICDTotSalesAmount>                Vacío
<Frequency>                        Vacío
<InventoryAdjustmentType>          Calculado = mapVrtIDToAdjType(VRT_ID): VRT_ID="1" → "JUSTIFIED_ADJUSTMENTS"; VRT_ID="2" → "UNJUSTIFIED_ADJUSTMENTS"; cualquier otro → "CORRECTIVE_ADJUSTMENT" — UNKNOWN A DEFINIR tabla de traducción con OCPRA
<ReceiptNumber>                    Vacío
<FiscalReceiptFlag>                Valor Fijo = "false"
<ReceiptType>                      Vacío
<ReceiptDate>                      Calculado = fecha de CHG_ZEIT (solo fecha, hora fija "00:00:00.000") con formato "YYYY-MM-DD HH:MM:SS.000"; si no es fecha válida → Igual al Tag BeginDateTime con formato "YYYY-MM-DD HH:MM:SS.000"
<CAINumber>                        Vacío
<CAIDate>                          Vacío
<PagesQuantity>                    Vacío
<NetAmount>                        Vacío
<ExemptAmout>                      Vacío
<TaxAmount>                        Vacío
<VatAmount>                        Vacío
<ServicesVATAmount>                Vacío
<DifferencialVATAmount>            Vacío
<IvaTaxAmount>                     Vacío
<IIBBTaxAmount>                    Vacío
<TotalAmount>                      Vacío
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
<ICDAmount>                        
<LastUpdateDate>                   
<SourceRetailStore>                
<Supplier>                         
<OrderDocumentType>                
<User>                             
<ICDQuantity>                      
<ICDTotSalesAmount>                
<Frequency>                        
<InventoryAdjustmentType>          
<ReceiptNumber>                    
<FiscalReceiptFlag>                
<ReceiptType>                      
<ReceiptDate>                      
<CAINumber>                        
<CAIDate>                          
<PagesQuantity>                    
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

> `DetSequenceNumber` proviene directamente de `VBT_POS` (posición en HIS_VERBRAUCHPOS), no es un índice calculado.
> `PickupCode` y `Stock` son UNKNOWN A DEFINIR — ver sección de pendientes abajo.

```
<RetailStoreID>                      Calculado = Igual al Tag RetailStoreID
<WorkstationID>                      Calculado = Igual al Tag WorkstationID
<SequenceNumber>                     Calculado = Igual al Tag SequenceNumber
<DetSequenceNumber>                  Query Items = VBT_POS
<Item>                               Query Items = ART_NUMMER (join ARTIKEL: HIS_VERBRAUCHPOS.ART_NR = ARTIKEL.ART_ID)
<UomUnits>                           Calculado = CAST(VPK_NR AS INT) con 4 decimales
<ItemBrand>                          Vacío
<ItemDescription>                    Query Items = ART_NAME (join ARTIKEL: HIS_VERBRAUCHPOS.ART_NR = ARTIKEL.ART_ID)
<UnitBaseCostAmount>                 Query Items = VBT_WES con 4 decimales
<UnitCount>                          Valor Fijo = "0.0000"
<DestinationLocation>                Valor Fijo = "DEP1_OS"
<SourceLocation>                     Valor Fijo = "DEP1_OS"
<CostTotalAmount>                    Query Items = VBT_WES con 4 decimales (igual que UnitBaseCostAmount)
<UnitSalesAmount>                    Valor Fijo = "0.0000"
<SalesTotalAmount>                   Valor Fijo = "0.0000"
<Stock>                              Valor Fijo = "0.0000" — UNKNOWN A DEFINIR (Excel: "STOCK TEÓRICO INICIAL previo al conteo manual"; campo origen no especificado)
<DailyAverageSales>                  Valor Fijo = "0.0000"
<SuggestedPurchaseOrder>             Valor Fijo = "0.0000"
<PickupCode>                         Valor Fijo = "S1" — UNKNOWN A DEFINIR (validar si puede ser "S2" para artículos que no manejan stock)
<LastUpdateDate>                     Calculado = Igual al Tag CreateDateTimestamp (cabecera)
<DifBME_ASNTypeID>                   Vacío
<InventoryControlDocumentState>      Valor Fijo = "4"
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
<PickupCode>                         
<LastUpdateDate>                     
<DifBME_ASNTypeID>                   
<InventoryControlDocumentState>      
```

---

## Pendientes UNKNOWN A DEFINIR

| Campo | Situación |
|---|---|
| `InventoryAdjustmentType` | Se resuelve desde `VRT_ID` de `HIS_VERBRAUCH`. Mapeo actual: `1` → `JUSTIFIED_ADJUSTMENTS`, `2` → `UNJUSTIFIED_ADJUSTMENTS`, default → `CORRECTIVE_ADJUSTMENT`. Validar tabla completa con OCPRA. |
| `Stock` (detalle) | Fijado en `0.0000`. Excel define "STOCK TEÓRICO INICIAL previo al conteo manual" pero no especifica el campo origen en `HIS_VERBRAUCH` / `HIS_VERBRAUCHPOS`. Validar con OCPRA. |
| `PickupCode` (detalle) | Fijado en `S1`. Excel dice fijo `S1` pero no aclara si puede ser `S2` para artículos que no manejan stock. Validar si depende de algún atributo del artículo. |

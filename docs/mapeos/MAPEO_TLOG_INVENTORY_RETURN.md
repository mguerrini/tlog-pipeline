# Mapeo TLOG_INVENTORY_RETURN

## Información General

- **Tipo de documento:** Devolución de Mercadería al Proveedor (`InventoryReturn`)
- **Tabla driver (cabecera):** `LIEFERSCHEIN`
- **Tabla detalle:** `LIEFERPOS` (relacionada por `LFS_ID`)
- **Documento de referencia:** `ANALISIS DE CAMPOS EN TLOG_OCPRA` — hoja `InventoryReturn`
- **Estados del documento:**
  - `4` = Cerrado (cuando la devolución tiene documento NC asociado)
  - `7` = Pendiente de Imputación (hasta que se impute con la NC y pase a `4`)
  - En el dataset provisto, el `LFS_STATUS = 42` se mapea a `4`.
- **Asociación:** Cada `InventoryReturn` se asocia con un `InventoryFiscalDoc` (NC) cuando se imputa la nota de crédito.

---

## 📋 Comunes a todos los TLOG (header de Transaction)

| Campo | Valor / Origen |
|---|---|
| `BusinessDayDate` | Fecha que viene en el nombre del archivo (`AAAAMMDD`) |
| `Period` | `0` (fijo) |
| `Subperiod` | `0` (fijo) |
| `BeginDateTime` | `BusinessDayDate + BEGIN_DATE_OFFSET` (valor de configuración) |
| `EndDateTime` | `BusinessDayDate + END_DATE_OFFSET` (valor de configuración) |
| `OperatorID` | `admin` (fijo) |

### Regla de SEQUENCENUMBER (TLOG OCPRA = SU)

- **SEQUENCENUMBER:** numérico longitud **12**, comenzando por `900000000001` para la APIES.
- **DET_SEQUENCENUMBER:** numérico longitud **3**, de 1 a 999 por documento (tope 999 artículos).

---

## Condición de Registro (cuándo se genera)

Se genera un TLOG `InventoryReturn` por **cada registro de la tabla `LIEFERSCHEIN`** que cumpla:

- `LFS_RTS = 1` → es devolución a proveedor (Return To Supplier).
- `LFS_NETTO` y `LFS_BRUTTO` **negativos** (devolución).
- `LFS_STATUS in (37, 42)` → mapeado a `InventoryControlDocumentState = 4` (cerrado/imputado), o `7` si quedó pendiente.
- `AKTIV = 'J'` (registro activo).

> **Dataset de ejemplo (`Lieferschein_20260505.csv`):** los registros que cumplen esta condición son los `LFS_ID`: **5, 7, 15, 22** (todos con `LFS_RTS = 1` y montos negativos).
> Los registros con `LFS_RTS` nulo y montos positivos (`LFS_ID`: 4, 6, 8, 19, 20, 21, 23, 24) corresponden a **recepciones** y se procesan en `InventoryReception`.

---

## Mapeo de Cabecera

`<Transaction>` y `<InventoryControlTransaction>` (un nodo por cabecera de `LIEFERSCHEIN` que cumple la condición).

| Campo XML (TLOG) | Valor en XML ejemplo | Tabla / Origen | Campo origen | Observaciones |
|---|---|---|---|---|
| `<RetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS. Se obtiene haciendo `KOSTST.KST_ID = LIEFERPOS.KST_ID` (el `KST_ID` está en el detalle, no en la cabecera de `LIEFERSCHEIN`). |
| `<WorkstationID>` | `0` | fijo | fijo | `0` = BME (Back Office). |
| `<SequenceNumber>` | `000190000002` | Construir | — | Numérico longitud 12, comenzando por `900000000001` para la APIES (SU). Se construye en Bridge/SU. |
| `<BusinessDayDate>` | `2026-05-05` | Construir | — | Fecha que viene en el nombre del archivo (`AAAAMMDD`). Se obtiene del WS de Periodos y Turnos en Bridge. |
| `<Period>` | `0` | fijo | fijo | Valor fijo `0`. |
| `<Subperiod>` | `0` | fijo | fijo | Valor fijo `0`. |
| `<PeriodCode>` | `0` | vacío | — | Debe obtenerse del WS de Periodos y Turnos en Bridge. |
| `<SubPeriodCode>` | `0` | vacío | — | Debe obtenerse del WS de Periodos y Turnos en Bridge (turno). |
| `<BeginDateTime>` | `2026-05-05 00:00:00` | Construir | — | `BusinessDayDate + BEGIN_DATE_OFFSET` (configuración). Inicia 22:00:01 según planilla. |
| `<EndDateTime>` | `2026-05-05 23:59:59` | Construir | — | `BusinessDayDate + END_DATE_OFFSET` (configuración). Termina 22:00:00 según planilla. |
| `<OperatorID>` | `admin` | fijo | fijo | Valor fijo `admin` (regla del proyecto). El XML revisado trae `1`. |
| `<OriginalTransaction>` | (vacío) | — | — | No se utiliza. |
| `<SerialFormID>` | `000190000002` | — | — | Mismo contenido que `SequenceNumber` de cabecera. |
| `<DocumentTypeCode>` | `InventoryReturn` | fijo | fijo | Valor fijo `InventoryReturn`. Condición `LFS_RTS = 1`. |
| `<InventoryControlDocumentState>` | `4` | `LIEFERSCHEIN` | `LFS_STATUS` | `LFS_STATUS in (37, 42)` → `4` (Cerrado). Cuando queda pendiente de imputación → `7`. |
| `<contractReferenceNumber>` | (vacío) | — | — | No se utiliza. |
| `<CreateDateTimestamp>` | `2026-05-05 10:00:00.000 ART` | Construir | — | Mismo dato y formato que `BeginDateTime`. Validar con Emma. |
| `<DestinationRetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | Mismo dato que `RetailStoreID`. Validar con Emma. |
| `<ExpectedDeliveryDate>` | `2026-05-05 00:00:00.0 ART` | Construir | — | Mismo dato y formato que `BeginDateTime`. Validar con Emma. |
| `<ICDAmount>` | `1500.0000` | `LIEFERSCHEIN` | `ABS(LFS_BRUTTO)` | Costo total de la devolución. Se valoriza con el costo del Maestro de Artículos. **El XML revisado indica que NO debe ir con signo negativo**: usar `ABS(LFS_BRUTTO)`. |
| `<LastUpdateDate>` | `2026-05-05 10:01:00.000 ART` | Construir | — | Mismo dato y formato que `BeginDateTime`. Validar con Emma. |
| `<Originator>` | (vacío) | — | — | No se utiliza. |
| `<SourceRetailStore>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS. Mismo valor que `RetailStoreID`. |
| `<Supplier>` | `30-0000000000-5` | `LIEFER` | `LF_VERT` | Código del proveedor (CUIT). Formato: `30-0000000000-5`. `[UNKNOWN] - {LF_VERT vacío en dataset} - {Aunque el Excel confirma que el campo es LF_VERT, los datos de prueba traen LF_VERT vacío para LF_ID=17. Validar dónde se carga el CUIT del proveedor (¿podría ser LF_EU_IDNR?)}` |
| `<OrderDocumentType>` | (vacío) | — | — | No se utiliza. |
| `<User>` | `admin` | fijo | fijo | Valor fijo `admin` (regla del proyecto). Mismo valor que `OperatorID`. |
| `<ICDQuantity>` | (vacío) | — | — | No se utiliza. |
| `<ICDTotSalesAmount>` | (vacío) | — | — | No se utiliza. |
| `<Frequency>` | (vacío) | — | — | No se utiliza. |
| `<InventoryAdjustmentType>` | (vacío) | — | — | No se utiliza. |
| `<ReceiptNumber>` | `REC202110-0000018` | `LIEFERSCHEIN` | `LFS_NAME` | Número de comprobante con formato ARCA: `A/X/C-XXXXX-XXXXXXXX` (Remito X, Factura A o C, Nota de Crédito A). |
| `<FiscalReceiptFlag>` | `false` | Construir | — | **Excel: regla condicional** — `true` para estado `7` (Pendiente de imputación), `false` para estado `4` (Cerrado). Se deriva de `InventoryControlDocumentState`. |
| `<ReceiptType>` | (vacío) | fijo | fijo | Viaja `null`. Validar con Emma. |
| `<ReceiptDate>` | `2026-05-05 00:00:00.0 ART` | `LIEFERSCHEIN` | `LFS_DATUM` | Fecha del comprobante. Validar con el Negocio. |
| `<CAINumber>` | (vacío) | — | — | No se utiliza en `InventoryReturn`. |
| `<CAIDate>` | (vacío) | — | — | No se utiliza en `InventoryReturn`. |
| `<PagesQuantity>` | *(vacío)* | — | — | **Viaja en null para los inventory return** (según comentarios del XML revisado). |
| `<NetAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return** (según comentarios del XML revisado). |
| `<ExemptAmout>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |
| `<TaxAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |
| `<VatAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |
| `<ServicesVATAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |
| `<DifferencialVATAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |
| `<IvaTaxAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |
| `<IIBBTaxAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |
| `<TotalAmount>` | *(vacío)* | — | — | **Viaja en null para los inventory return**. |

---

## Mapeo de Detalle

`<inventoryControlDocumentMerchandiseLineItem>` (un nodo por cada línea de `LIEFERPOS` con `LFS_ID = LFS_ID` de la cabecera).

| Campo XML (TLOG) | Valor en XML ejemplo | Tabla / Origen | Campo origen | Observaciones |
|---|---|---|---|---|
| `<DetSequenceNumber>` | `1` | Construir | — | Numérico longitud 3. A partir de `1` para cada documento (1..N por línea). El XML revisado lo trae como `<SequenceNumber>` pero **debe llamarse `<DetSequenceNumber>`**. |
| `<Item>` | `608` | `LIEFERPOS` | `ART_NR` | SKU / código de artículo. Relaciona con `ARTIKEL.ART_ID`. |
| `<UomUnits>` | `1.0000` | `LIEFERPOS` | `VPK_ID1` | Código de unidad de medida. Se relaciona con `VPCKEINH.VPK_ID`. (Excel confirma: es `VPK_ID`.) |
| `<ItemBrand>` | `0` | fijo | fijo | Valor fijo `0` (Excel: "MARCA" → fijo `0`). |
| `<ItemDescription>` | `Pechuga pollo entera` | `ARTIKEL` | `ART_NAME` | Nombre del artículo. Relación: `ARTIKEL.ART_ID = LIEFERPOS.ART_NR`. |
| `<UnitBaseCostAmount>` | `-1500.0000` o `1500.0000` | `LIEFERPOS` | `LFP_EKP / LFP_MENGE` | Costo unitario del artículo. Se obtiene del Maestro de Artículos. Calculado como `LFP_EKP / LFP_MENGE`. `[UNKNOWN] - {-1500.0000} - {El XML trae con signo negativo. La cabecera ICDAmount va en ABS(). ¿Coherencia entre cabecera y detalle? Validar si todo va ABS o solo cabecera}` |
| `<UnitCount>` | `-150.0000` o `150.0000` | `LIEFERPOS` | `LFP_MENGE` | Cantidad devuelta. `[UNKNOWN] - {-150.0000} - {El XML trae negativo. Validar coherencia con cabecera (ABS)}` |
| `<DestinationLocation>` | `DEP1_OS` | fijo | fijo | **Valor fijo `"DEP1_OS"`** (Excel: 'Valor: "DEP1_OS"' — Depósito OPESSA). |
| `<SourceLocation>` | `DEP1_OS` | fijo | fijo | **Valor fijo `"DEP1_OS"`** (Excel). |
| `<CostTotalAmount>` | `-150.0000` o `150.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Costo total del artículo en la devolución. `[UNKNOWN] - {-150.0000} - {El XML trae negativo. Validar coherencia con cabecera (ABS)}` |
| `<UnitSalesAmount>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<SalesTotalAmount>` | `-0.0000` | fijo | fijo | Valor fijo `0`. |
| `<Stock>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<DailyAverageSales>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<SuggestedPurchaseOrder>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<PickupCode>` | (vacío) | — | — | No se utiliza en `InventoryReturn`. |
| `<LastUpdateDate>` | (vacío) | — | — | No se utiliza en `InventoryReturn`. |
| `<DifBME_ASNTypeID>` | (vacío) | — | — | No se utiliza en `InventoryReturn`. |
| `<InventoryControlDocumentState>` (línea) | — | `LIEFERSCHEIN` | `LFS_STATUS` | `7` = Pendiente de Imputación / `4` = Cerrado. Cuando la devolución tiene NC asociada → `4`. Mientras esté pendiente → `7`. |

---

## Pseudocódigo

```
═══════════════════════════════════════════════════════════════
  PROCESO: Generar XML InventoryReturn
═══════════════════════════════════════════════════════════════

INICIO

  1. Leer cabeceras candidatas:
       SELECT * FROM LIEFERSCHEIN ls
       WHERE ls.LFS_RTS = 1
         AND ls.LFS_STATUS IN (37, 42)
         AND ls.AKTIV = 'J'
         AND ls.LFS_BRUTTO < 0

  2. PARA CADA cabecera ls:

     2.1. Resolver datos comunes (lookup):
          - APIES (RetailStoreID) = (KOSTST.KST_CODE WHERE KST_ID = LIEFERPOS.KST_ID)
              -- KST_ID viene del detalle, no de la cabecera
          - Proveedor = (LIEFER.LF_VERT WHERE LF_ID = ls.LF_ID)
          - Operador / User = 'admin' (fijo)
          - SequenceNumber = construir (Bridge: 12 dígitos comenzando con 9)
          - SerialFormID = SequenceNumber
          - BusinessDayDate = nombre del archivo (AAAAMMDD)
          - BeginDateTime = BusinessDayDate + BEGIN_DATE_OFFSET (config)
          - EndDateTime   = BusinessDayDate + END_DATE_OFFSET (config)

     2.2. Mapear cabecera del XML:
          - DocumentTypeCode = 'InventoryReturn'   (fijo)
          - InventoryControlDocumentState:
                IF ls.LFS_STATUS IN (37, 42) THEN 4
                ELSE 7
          - FiscalReceiptFlag:
                IF InventoryControlDocumentState = 4 THEN false
                ELSE true
          - ICDAmount         = ABS(ls.LFS_BRUTTO)        -- sin signo negativo
          - ReceiptNumber     = ls.LFS_NAME
          - ReceiptDate       = ls.LFS_DATUM
          - ReceiptType       = null
          - SourceRetailStore = APIES (mismo que RetailStoreID)
          - DestinationRetailStoreID = APIES (mismo que RetailStoreID)
          - Supplier          = LIEFER.LF_VERT

          -- Campos que viajan en null para los Inventory Return:
          - PagesQuantity, NetAmount, ExemptAmout, TaxAmount, VatAmount,
            ServicesVATAmount, DifferencialVATAmount, IvaTaxAmount,
            IIBBTaxAmount, TotalAmount → null

          -- Campos vacíos:
          - contractReferenceNumber, Originator, OrderDocumentType,
            ICDQuantity, ICDTotSalesAmount, Frequency,
            InventoryAdjustmentType, CAINumber, CAIDate → vacíos

     2.3. PARA CADA línea lp en LIEFERPOS WHERE LFS_ID = ls.LFS_ID
          ORDER BY LFP_POS:

          - DetSequenceNumber  = 1..N (correlativo por documento)
          - Item               = lp.ART_NR
          - UomUnits           = lp.VPK_ID1   (lookup en VPCKEINH si fuera necesario)
          - ItemBrand          = 0     (fijo)
          - ItemDescription    = (ARTIKEL.ART_NAME WHERE ART_ID = lp.ART_NR)
          - UnitBaseCostAmount = lp.LFP_EKP / lp.LFP_MENGE
          - UnitCount          = lp.LFP_MENGE       -- [UNKNOWN] signo
          - DestinationLocation = 'DEP1_OS'         -- fijo
          - SourceLocation     = 'DEP1_OS'          -- fijo
          - CostTotalAmount    = lp.LFP_BRUTTO       -- [UNKNOWN] signo
          - UnitSalesAmount    = 0  (fijo)
          - SalesTotalAmount   = 0  (fijo)
          - Stock              = 0  (fijo)
          - DailyAverageSales  = 0  (fijo)
          - SuggestedPurchaseOrder = 0  (fijo)
          - PickupCode, LastUpdateDate, DifBME_ASNTypeID → vacíos
          - InventoryControlDocumentState (línea):
                IF ls.LFS_STATUS IN (37, 42) THEN 4 ELSE 7

     2.4. Escribir XML.

  3. Marcar cabeceras como exportadas (EXP_NR / flag).

FIN
```

---

## 📌 Resumen de [UNKNOWN] (los que persisten)

| # | Campo | Razón |
|---|---|---|
| 1 | `<Supplier>` | `LF_VERT` está vacío en dataset (`LF_ID=17`). Validar campo de origen del CUIT (¿`LF_EU_IDNR`?). |
| 2 | `<UnitCount>`, `<UnitBaseCostAmount>`, `<CostTotalAmount>` (detalle) | El XML los trae con signo negativo, pero la cabecera `<ICDAmount>` va en `ABS()`. Validar con Negocio si en el TLOG real el detalle también debe ir en valor absoluto. |

# Mapeo TLOG_INVENTORY_RECEPTION

## Información General

- **Tipo de documento:** Recepción de Mercadería (`InventoryReception`)
- **Tabla driver (cabecera):** `LIEFERSCHEIN`
- **Tabla detalle:** `LIEFERPOS` (relacionada por `LFS_ID`)
- **Documento de referencia:** `ANALISIS DE CAMPOS EN TLOG_OCPRA` — hoja `InventoryReception`
- **Estados del documento:** `4` (cerrado, asociado a FC) o `7` (pendiente de imputación). En el set de datos provisto, el `LFS_STATUS = 42` se convierte a `4`.

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
- Composición de Tx única: `APIES‖PosID‖NroDeSecuencia` (longitud máxima conjunta = 15).

---

## Condición de Registro (cuándo se genera)

Se genera un TLOG `InventoryReception` por **cada registro de la tabla `LIEFERSCHEIN`** que cumpla:

- `LFS_RTS` distinto de `1` (no es devolución / Nota de Crédito — esos van por `InventoryReturn`).
- `LFS_NETTO` y `LFS_BRUTTO` **positivos** (recepción real, no NC).
- `LFS_STATUS = 42` (recepción cerrada / imputada). → Se mapea a `InventoryControlDocumentState = 4`.
- Si la recepción quedara **pendiente de imputación** (sin documento fiscal asociado), `InventoryControlDocumentState = 7`.
- `AKTIV = 'J'` (registro activo).

> **Nota:** En el dataset de ejemplo (`Lieferschein_20260505.csv`) los registros que cumplen esta condición son los `LFS_ID`: `4, 6, 8, 19, 20, 21, 23, 24` (todos con `LFS_RTS` nulo y montos positivos).
> Los registros con `LFS_RTS = 1` y montos negativos (`LFS_ID`: `5, 7, 15, 22`) corresponden a **devoluciones (Notas de Crédito)** y se procesan en `InventoryReturn` / `InventoryFiscalDoc (NC)`.

---

## Mapeo de Cabecera

`<Transaction>` y `<InventoryControlTransaction>` (un nodo por cabecera de `LIEFERSCHEIN`).

| Campo XML (TLOG) | Valor en XML ejemplo | Tabla / Origen | Campo origen | Observaciones |
|---|---|---|---|---|
| `<RetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS. Se obtiene haciendo `KOSTST.KST_ID = LIEFERPOS.KST_ID` (el `KST_ID` está en el detalle, no en la cabecera de `LIEFERSCHEIN`). |
| `<WorkstationID>` | `0` | fijo | fijo | `0` = BME (Back Office). |
| `<SequenceNumber>` | `000190000001` | Construir | — | Numérico longitud 12, comenzando por `900000000001` para la APIES (SU). Se construye en Bridge/SU. |
| `<BusinessDayDate>` | `2026-05-05` | Construir | — | Fecha que viene en el nombre del archivo (`AAAAMMDD`). Se obtiene del WS de Periodos y Turnos en Bridge. |
| `<Period>` | `0` | fijo | fijo | Valor fijo `0`. |
| `<Subperiod>` | `0` | fijo | fijo | Valor fijo `0`. |
| `<PeriodCode>` | `0` | vacío / WS | — | Se obtiene del WS de Periodos y Turnos en Bridge. En el documento de mapeo figura como vacío → enviar `0`. |
| `<SubPeriodCode>` | `0` | vacío / WS | — | Se obtiene del WS de Periodos y Turnos en Bridge. En el documento de mapeo figura como vacío → enviar `0`. |
| `<BeginDateTime>` | `2026-05-05 00:00:00` | Construir | — | `BusinessDayDate + BEGIN_DATE_OFFSET` (config). Inicio de día (típicamente `22:00:01` del día anterior). |
| `<EndDateTime>` | `2026-05-05 23:59:59` | Construir | — | `BusinessDayDate + END_DATE_OFFSET` (config). Cierre de día (típicamente `22:00:00`). |
| `<OperatorID>` | `admin` | fijo | fijo | `admin` (regla del proyecto). El XML revisado trae `1`, debe enviarse `admin`. |
| `<OriginalTransaction>` | *(vacío)* | — | — | No se usa. |
| `<SerialFormID>` | `000190000001` | = `<SequenceNumber>` | — | Mismo contenido que `SequenceNumber` de cabecera. |
| `<DocumentTypeCode>` | `InventoryReception` | fijo | fijo | Valor fijo. |
| `<InventoryControlDocumentState>` | `4` | `LIEFERSCHEIN` | `LFS_STATUS` | `LFS_STATUS = 42` → mapear a `4` (cerrado). Si la recepción está pendiente de imputación → `7`. |
| `<contractReferenceNumber>` | *(vacío)* | — | — | No se usa en este documento. |
| `<CreateDateTimestamp>` | `2026-05-05 10:00:00.000 ART` | = `<BeginDateTime>` | — | Mismo dato que `BeginDateTime` con formato milisegundos + zona ART. |
| `<DestinationRetailStoreID>` | *(vacío)* | — | — | No se usa en este documento (según mapeo Excel). El XML revisado lo trae con APIES, pero la planilla indica "NO SE USA" → enviar vacío. |
| `<ExpectedDeliveryDate>` | `2026-05-05 00:00:00.0 ART` | = `<BeginDateTime>` | — | Mismo dato que `BeginDateTime` con formato milisegundos + zona ART. |
| `<ICDAmount>` | `14869.0000` | `LIEFERSCHEIN` | `LFS_BRUTTO` | Costo total de la recepción (con impuestos). 4 decimales. |
| `<LastUpdateDate>` | `2026-05-05 10:01:00.000 ART` | = `<BeginDateTime>` | — | Mismo dato que `BeginDateTime` con formato milisegundos + zona ART. |
| `<Originator>` | *(vacío)* | — | — | No se usa. |
| `<SourceRetailStore>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS (mismo que `RetailStoreID`). |
| `<Supplier>` | `30-0000000000-5` | `LIEFER` | `LF_VERT` | Código del proveedor (CUIT). Lookup: `LIEFER.LF_ID = LIEFERSCHEIN.LF_ID` y se toma `LF_VERT`. `[UNKNOWN] - {LF_VERT vacío en dataset} - {Aunque el Excel confirma que el campo es LF_VERT, los datos de prueba traen LF_VERT vacío. Validar dónde se carga el CUIT del proveedor}` |
| `<OrderDocumentType>` | *(vacío)* | — | — | No se usa en este documento. |
| `<User>` | `admin` | fijo | fijo | `admin` (regla del proyecto). Mismo valor que `OperatorID`. El XML revisado trae `1`. |
| `<ICDQuantity>` | *(vacío)* | — | — | No se usa en este documento. |
| `<ICDTotSalesAmount>` | *(vacío)* | — | — | No se usa en este documento. |
| `<Frequency>` | *(vacío)* | — | — | No se usa en este documento. |
| `<InventoryAdjustmentType>` | *(vacío)* | — | — | No se usa en este documento. |
| `<ReceiptNumber>` | `00015` | `LIEFERSCHEIN` | `LFS_NAME` | Número de remito/factura. Formato esperado ARCA: `A/X/C-00000-00000000`. <br>En el dataset hay valores con formato correcto (`A-12345-12345678`, `F-2222-22222222`) y otros simplemente numéricos (`1`, `2`, `3`...). |
| `<FiscalReceiptFlag>` | `true` | fijo | fijo | Valor fijo `true` para `InventoryReception`. |
| `<ReceiptType>` | *(vacío / null)* | fijo | fijo | Valor fijo `null` (vacío) para `InventoryReception`. |
| `<ReceiptDate>` | `2026-05-05 00:00:00.0 ART` | `LIEFERSCHEIN` | `LFS_DATUM` | Fecha del comprobante (carga manual). Formato: `AAAA-MM-DD HH:MM:SS.0 ART`. |
| `<CAINumber>` | *(vacío)* | — | — | No se usa en este documento. |
| `<CAIDate>` | *(vacío)* | — | — | No se usa en este documento. |
| `<PagesQuantity>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`** (según mapeo). El XML revisado lo trae con `1`, pero la planilla indica que no se usa → enviar vacío. |
| `<NetAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<ExemptAmout>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<TaxAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<VatAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<ServicesVATAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<DifferencialVATAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<IvaTaxAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<IIBBTaxAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| `<TotalAmount>` | *(vacío)* | — | — | **NO SE UTILIZA EN `InventoryReception`**. → enviar vacío. |
| *(Estado)* | `REPLICADO` | fijo | fijo | Estado interno del registro: `REPLICADO`. (No es un nodo XML del archivo, sino un flag interno del proceso.) |

---

## Mapeo de Detalle

`<inventoryControlDocumentMerchandiseLineItem>` (un nodo por cada `LIEFERPOS` del `LFS_ID`).

| Campo XML (TLOG) | Valor en XML ejemplo | Tabla / Origen | Campo origen | Observaciones |
|---|---|---|---|---|
| `<RetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | APIES (idem cabecera). |
| `<WorkstationID>` | `0` | fijo | fijo | `0` = BME. |
| `<SequenceNumber>` | `000190000001` | — | — | Idem cabecera (mismo `SequenceNumber`). |
| `<DetSequenceNumber>` | `1` | construir | — | Numerador secuencial 1..N por documento. Numérico longitud 3 (1 a 999). **El campo en el XML revisado se llama `<SequenceNumber>` dentro del detalle, pero debe llamarse `<DetSequenceNumber>`** (corrección indicada en la planilla y en los comentarios del XML). |
| `<Item>` | `567` | `LIEFERPOS` | `ART_NR` | SKU / código de artículo. Se relaciona `ARTIKEL.ART_ID = LIEFERPOS.ART_NR`. |
| `<UomUnits>` | `1.0000` | `LIEFERPOS` | `VPK_ID1` | Código de unidad de medida. Se relaciona `VPCKEINH.VPK_ID = LIEFERPOS.VPK_ID1`. (Excel confirma: es `VPK_ID`, no factor.) |
| `<ItemBrand>` | `0` | fijo | fijo | Valor fijo `0` (Excel: "MARCA" → fijo `0`, no se maneja marca). |
| `<ItemDescription>` | `Agua mineral con gas` | `ARTIKEL` | `ART_NAME` | Descripción del artículo. Lookup: `ARTIKEL.ART_ID = LIEFERPOS.ART_NR`. |
| `<UnitBaseCostAmount>` | `50.0000` | `LIEFERPOS` | `LFP_EKP / LFP_MENGE` | Costo unitario del artículo. Se calcula dividiendo el neto recibido (`LFP_EKP`) por la cantidad recibida (`LFP_MENGE`). |
| `<UnitCount>` | `2.0000` | `LIEFERPOS` | `LFP_MENGE` | Cantidad recibida. |
| `<DestinationLocation>` | `DEP1_OS` | fijo | fijo | **Valor fijo `"DEP1_OS"`** (Excel: 'Valor: "DEP1_OS"' — Depósito OPESSA). No se resuelve desde KST_ID. |
| `<SourceLocation>` | `DEP1_OS` | fijo | fijo | **Valor fijo `"DEP1_OS"`** (Excel). |
| `<CostTotalAmount>` | `100.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Costo total del artículo en la recepción. |
| `<UnitSalesAmount>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<SalesTotalAmount>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<Stock>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<DailyAverageSales>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<SuggestedPurchaseOrder>` | `0.0000` | fijo | fijo | Valor fijo `0`. |
| `<PickupCode>` | *(vacío)* | — | — | No se usa en este documento. |
| `<LastUpdateDate>` | *(vacío)* | — | — | No se usa en este documento. |
| `<DifBME_ASNTypeID>` | *(vacío)* | — | — | No se usa en este documento. |
| `<InventoryControlDocumentState>` *(detalle)* | `1` | fijo | fijo | Valor fijo `1`. (No está presente en el XML revisado pero figura en la planilla.) |

---

## Lookups / joins necesarios

| Origen | Destino | Clave | Uso |
|---|---|---|---|
| `LIEFERSCHEIN.LF_ID` | `LIEFER.LF_ID` | N:1 | Datos de proveedor (`LF_VERT` para `<Supplier>`). |
| `LIEFERPOS.LFS_ID` | `LIEFERSCHEIN.LFS_ID` | N:1 | Asociar líneas a la cabecera. |
| `LIEFERPOS.KST_ID` | `KOSTST.KST_ID` | N:1 | APIES (`KST_CODE`) de la EESS. |
| `LIEFERPOS.ART_NR` | `ARTIKEL.ART_ID` | N:1 | Datos de artículo (`ART_NAME`). |
| `LIEFERPOS.VPK_ID1` | `VPCKEINH.VPK_ID` | N:1 | Unidad de medida. |

> **Importante:** la cabecera `LIEFERSCHEIN` **no contiene `KST_ID`** directamente. La APIES se obtiene desde el `KST_ID` de las líneas (`LIEFERPOS.KST_ID`), que en el dataset es consistente para todas las líneas de un mismo `LFS_ID` (por ejemplo, `KST_ID = 15` para `LFS_ID = 6`).

---

## Pseudocódigo de generación

```
═══════════════════════════════════════════════════════════════
  PROCESO: Generar XML InventoryReception
═══════════════════════════════════════════════════════════════

INICIO

  1. Seleccionar cabeceras de LIEFERSCHEIN imputadas y NO devolución:
       SELECT *
       FROM LIEFERSCHEIN ls
       WHERE ls.LFS_STATUS = 42                       -- imputado/cerrado
         AND (ls.LFS_RTS IS NULL OR ls.LFS_RTS <> 1)  -- no es NC
         AND ls.LFS_NETTO  > 0
         AND ls.LFS_BRUTTO > 0
         AND ls.AKTIV = 'J'
         AND ls.EXP_NR IS NULL                        -- no exportado todavía (si aplica)

  2. PARA CADA cabecera ls:

     2.1. Obtener líneas: LIEFERPOS WHERE LFS_ID = ls.LFS_ID

     2.2. Resolver lookups:
          - KST  = KOSTST WHERE KST_ID = (LIEFERPOS.KST_ID de la primera línea)
          - LF   = LIEFER WHERE LF_ID = ls.LF_ID
          - PER  = WS Bridge → BusinessDayDate, PeriodCode, SubPeriodCode
          - SEQ  = Construir SequenceNumber (12 dígitos, comienza con 9)

     2.3. Construir <Transaction>:
          - RetailStoreID       = KST.KST_CODE
          - WorkstationID       = 0
          - SequenceNumber      = SEQ
          - BusinessDayDate     = nombreArchivo[AAAAMMDD]
          - Period / Subperiod  = 0 / 0
          - PeriodCode / SubPeriodCode = WS Bridge (o 0)
          - BeginDateTime       = BusinessDayDate + BEGIN_DATE_OFFSET
          - EndDateTime         = BusinessDayDate + END_DATE_OFFSET
          - OperatorID          = 'admin'

     2.4. Construir <InventoryControlTransaction>:
          - SerialFormID        = SEQ
          - DocumentTypeCode    = 'InventoryReception'
          - InventoryControlDocumentState = (ls.LFS_STATUS = 42 → 4; pendiente → 7)
          - CreateDateTimestamp = BeginDateTime + ' ART'  (con ms)
          - ExpectedDeliveryDate= BeginDateTime + ' ART'
          - ICDAmount           = ls.LFS_BRUTTO
          - LastUpdateDate      = BeginDateTime + ' ART'
          - SourceRetailStore   = KST.KST_CODE
          - Supplier            = LF.LF_VERT
          - User                = 'admin'
          - ReceiptNumber       = ls.LFS_NAME
          - FiscalReceiptFlag   = true
          - ReceiptType         = null
          - ReceiptDate         = ls.LFS_DATUM
          - (Resto de campos NetAmount/TaxAmount/etc. → vacíos)

     2.5. PARA CADA línea lp en LIEFERPOS WHERE LFS_ID = ls.LFS_ID:

          - DetSequenceNumber   = numerar 1..N
          - Item                = lp.ART_NR
          - UomUnits            = lp.VPK_ID1   -- código de unidad de medida (VPCKEINH.VPK_ID)
          - ItemBrand           = 0            -- fijo
          - ItemDescription     = (ARTIKEL.ART_NAME WHERE ART_ID = lp.ART_NR)
          - UnitBaseCostAmount  = lp.LFP_EKP / lp.LFP_MENGE
          - UnitCount           = lp.LFP_MENGE
          - DestinationLocation = 'DEP1_OS'    -- fijo
          - SourceLocation      = 'DEP1_OS'    -- fijo
          - CostTotalAmount     = lp.LFP_BRUTTO
          - UnitSalesAmount     = 0
          - SalesTotalAmount    = 0
          - Stock               = 0
          - DailyAverageSales   = 0
          - SuggestedPurchaseOrder = 0
          - InventoryControlDocumentState (detalle) = 1

     2.6. Escribir XML.

  3. Marcar cabeceras como exportadas (EXP_NR / flag).

FIN
```

---

## 📌 Resumen de [UNKNOWN] (los que persisten)

| # | Campo / Concepto | Razón |
|---|---|---|
| 1 | `<Supplier>` | `LF_VERT` está vacío en el dataset de ejemplo. Validar carga de CUIT del proveedor. |

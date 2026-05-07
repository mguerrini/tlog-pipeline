# Mapeo TLOG_INVENTORY_FISCAL_DOC_FC

## Información General

- **Tipo de documento:** Documento Fiscal — Factura (FC)
- **Tabla driver:** `LIEFERSCHEIN`
- **Tabla detalle:** `LIEFERPOS`
- **Documento de referencia:** `ANALISIS DE CAMPOS EN TLOG_OCPRA` — hoja `InventoryFiscalDoc`

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

Se genera un TLOG `InventoryFiscalDoc` con `ReceiptType = FC` por cada registro de la tabla `LIEFERSCHEIN` que cumpla:

- `LFS_STATUS = 42` (factura imputada/cerrada)
- `LFS_RTS` distinto de `1` (no es devolución/Nota de Crédito)
- Montos `LFS_NETTO` y `LFS_BRUTTO` **positivos** (factura, no NC)
- `AKTIV = 'J'` (registro activo)

> Nota: En el set de datos de ejemplo (`Lieferschein_20260505.csv`) los registros con `LFS_RTS=1` y montos negativos corresponden a **Notas de Crédito (NC)**. Los registros con `LFS_RTS` nulo y montos positivos corresponden a **Facturas (FC)**.

---

## Mapeo de Cabecera (`<Transaction>` y `<InventoryControlTransaction>`)

| Campo XML (TLOG) | Valor en XML ejemplo | Origen / Tabla | Campo origen | Observaciones |
|---|---|---|---|---|
| `<RetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS. Tomar el `KST_CODE` del centro de costo asociado (a través de `LIEFERPOS.KST_ID`). |
| `<WorkstationID>` | `0` | Fijo | — | Valor fijo `0` (BME). |
| `<SequenceNumber>` (cabecera) | `000190000002` | Construir | — | Longitud 12, debe iniciar en `900000000001` para la APIES (SU). Construido por Bridge. |
| `<BusinessDayDate>` | `2026-05-05` | Nombre del archivo | — | Fecha que viene en el nombre del archivo (`AAAAMMDD`). |
| `<Period>` | `0` | Fijo | — | Valor fijo `0`. |
| `<Subperiod>` | `0` | Fijo | — | Valor fijo `0`. |
| `<PeriodCode>` | `0` (en XML) → vacío | vacío | — | Mapeo dice "vacío". Aunque el XML trae `0`, **prevalece el documento de mapeo**: va vacío. |
| `<SubPeriodCode>` | `0` (en XML) → vacío | vacío | — | Mapeo dice "vacío". Aunque el XML trae `0`, **prevalece el documento de mapeo**: va vacío. |
| `<BeginDateTime>` | `2026-05-05 00:00:00` | Construir | — | `BusinessDayDate + BEGIN_DATE_OFFSET` (valor desde configuración). Mapeo indica "Inicia 22:00:01". |
| `<EndDateTime>` | `2026-05-05 23:59:59` | Construir | — | `BusinessDayDate + END_DATE_OFFSET` (valor desde configuración). Mapeo indica "Termina 22:00:00". |
| `<OperatorID>` | `admin` | Fijo | — | Valor fijo `admin` (regla del proyecto). El XML trae `1`, debe enviarse `admin`. |
| `<OriginalTransaction>` | `<!--NO SE UTILIZA-->` | — | — | NO SE UTILIZA. Vacío. |
| `<SerialFormID>` | `000190000002` | Igual a `SequenceNumber` | — | Mismo contenido que `SequenceNumber` de cabecera. |
| `<DocumentTypeCode>` | `InventoryFiscalDoc` | Fijo | — | Valor fijo `InventoryFiscalDoc`. |
| `<InventoryControlDocumentState>` | `4` | `LIEFERSCHEIN` | `LFS_STATUS` | Para `INVENTORYFISCALDOC` debe ser **`4`**. En LIEFERSCHEIN viene `LFS_STATUS=42`. Mapeo `LFS_STATUS=42 → 4`. (XML trae `37`, prevalece mapeo Excel = `4`.) |
| `<contractReferenceNumber>` | vacío | — | — | Vacío. |
| `<CreateDateTimestamp>` | `2026-05-05 10:00:00.000 ART` | Construir | — | Mismo dato que `BeginDateTime` con formato milisegundos + ` ART`. |
| `<DestinationRetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | Igual a `RetailStoreID` (APIES de la EESS, a través de `LIEFERPOS.KST_ID`). |
| `<ExpectedDeliveryDate>` | `2026-05-05 00:00:00.0 ART` | Construir | — | Mismo dato que `BeginDateTime` con formato milisegundos + ` ART`. |
| `<ICDAmount>` | `81470.0000` | `LIEFERSCHEIN` | `LFS_BRUTTO` | Costo total de la factura (importe bruto). |
| `<LastUpdateDate>` | `2026-05-05 10:01:00.000 ART` | Construir | — | Mismo dato que `BeginDateTime` con formato milisegundos + ` ART`. |
| `<Originator>` | vacío | — | — | Vacío. |
| `<SourceRetailStore>` | `00019` | `KOSTST` | `KST_CODE` | APIES origen. **Debe coincidir con `RetailStoreID`** (Excel: "kst_code de kostst" — APIES EESS, mismo centro de costo). El XML ejemplo con `00009` ≠ `RetailStoreID` (`00019`) es **error del ejemplo**. |
| `<Supplier>` | `30-0000000000-5` | `LIEFER` | `LF_VERT` | Código de proveedor (CUIT). `[UNKNOWN] - {LF_VERT vacío en dataset} - {Aunque el Excel confirma que el campo es LF_VERT, los datos de prueba traen LF_VERT vacío. Validar dónde se carga el CUIT del proveedor}` |
| `<OrderDocumentType>` | vacío | — | — | Vacío. |
| `<User>` | `admin` | Fijo | — | Valor fijo `admin` (regla del proyecto). |
| `<ICDQuantity>` | vacío | — | — | Vacío. |
| `<ICDTotSalesAmount>` | vacío | — | — | Vacío. |
| `<Frequency>` | vacío | — | — | Vacío. |
| `<InventoryAdjustmentType>` | vacío | — | — | Vacío. |
| `<ReceiptNumber>` | `00003` | `LIEFERSCHEIN` | `LFS_NAME` | Número de comprobante con formato `A-XXXXX-XXXXXXXX` (ARCA: A/X/C-00000-00000000). En los datos viene como `A-12345-12345678`, `F-2222-22222222`, etc. |
| `<FiscalReceiptFlag>` | `false` | Fijo | — | Valor fijo `false` para `InventoryFiscalDoc`. |
| `<ReceiptType>` | `FC` | Fijo (por condición) | — | Valor fijo `FC` cuando se cumple la condición de registro de Factura (ver sección "Condición de Registro"). |
| `<ReceiptDate>` | `2026-05-05 00:00:00.0 ART` | `LIEFERSCHEIN` | `LFS_DATUM` | Fecha del comprobante (carga manual). Formato `YYYY-MM-DD HH:MM:SS.0 ART`. |
| `<CAINumber>` | vacío en XML | `LIEFERSCHEIN` | `LFS_INFO` | Número de CAI. En `LFS_INFO` viene en formato `CAINUMBER\|FECHAVENCIMIENTO` (ej: `86151155096498\|30/04/2026`). Tomar parte previa al `\|`. |
| `<CAIDate>` | vacío en XML | `LIEFERSCHEIN` | `LFS_INFO` | Fecha de vencimiento del CAI. En `LFS_INFO` viene en formato `CAINUMBER\|FECHAVENCIMIENTO`. Tomar parte posterior al `\|`. |
| `<PagesQuantity>` | `1` | Fijo | — | Valor fijo `1`. |
| `<NetAmount>` | `81470.0000` | `LIEFERSCHEIN` | `LFS_NETTO` | Valor neto. |
| `<ExemptAmout>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<TaxAmount>` | `0.0000` | `LIEFERSCHEIN` | `LFS_???` | `[UNKNOWN] - {0.0000} - {Excel indica origen "LFS_" sin completar (impuesto interno). ¿Hay un campo en LIEFERSCHEIN para este valor o se calcula? Validar con Emma}` |
| `<VatAmount>` | `0.0000` | `LIEFERSCHEIN` | `LFS_MWST` | Importe de IVA base. |
| `<ServicesVATAmount>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<DifferencialVATAmount>` | `0.0000` | Fijo | — | Valor fijo `0`. (Validar con Napse: `TLOG.DIFFERENCIALVATAMOUNT * 10000 AS IMPIVACZ_CFAC`). |
| `<IvaTaxAmount>` | `0.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Percepción de IVA. `[UNKNOWN] - {0.0000} - {Excel dice "Valor bruto del art_nr identificado como [vacío]" — frase incompleta. ¿Cuál es el criterio para identificar el artículo de percepción IVA? Validar con Emma}` |
| `<IIBBTaxAmount>` | `0.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Percepciones IIBB FC/NC. `[UNKNOWN] - {0.0000} - {Excel dice "Valor bruto del art_nr identificado como [vacío]" — frase incompleta. ¿Cuál es el criterio para identificar el artículo de percepción IIBB? Validar con Emma}` |
| `<TotalAmount>` | `81470.0000` | `LIEFERSCHEIN` | `LFS_BRUTTO` | Total de la FC. |

---

## Mapeo de Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)

Por cada registro de `LIEFERPOS` asociado al `LIEFERSCHEIN` (vía `LFS_ID`), se genera un `inventoryControlDocumentMerchandiseLineItem`.

| Campo XML (TLOG) | Valor en XML ejemplo | Origen / Tabla | Campo origen | Observaciones |
|---|---|---|---|---|
| `<RetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS (heredado de cabecera, a través de `LIEFERPOS.KST_ID` → `KOSTST`). |
| `<WorkstationID>` | `0` | Fijo | — | Valor fijo `0` (BME). |
| `<SequenceNumber>` (detalle) | `000190000002` | Construir | — | Igual al `SequenceNumber` de cabecera. Longitud 12 (SU). |
| `<DetSequenceNumber>` (debe llamarse así, en XML viene como `<SequenceNumber>`) | `1`, `2`, `3`, ... | construir | — | **El campo debe llamarse `DetSequenceNumber`**. Numeración secuencial 1..N de los ítems dentro de la factura. Longitud 3 (tope 999 artículos por factura). |
| `<Item>` | `567` | `LIEFERPOS` | `ART_NR` | SKU / Código de artículo. |
| `<UomUnits>` | `1.0000` | `LIEFERPOS` | `VPK_ID1` | Código de unidad de medida (referencia a `VPCKEINH.VPK_ID`). |
| `<ItemBrand>` | `0` | Fijo | — | Valor fijo `0` (Excel: "MARCA" → fijo `0`). |
| `<ItemDescription>` | `Agua mineral con gas` | `ARTIKEL` | `ART_NAME` | Descripción del artículo (JOIN `LIEFERPOS.ART_NR` = `ARTIKEL.ART_ID`). |
| `<UnitBaseCostAmount>` | `250.0000` | `LIEFERPOS` | `LFP_EKP / LFP_MENGE` | Costo unitario del artículo. Calcular como `LFP_EKP / LFP_MENGE`. NO se carga; se obtiene del Maestro de Artículos. |
| `<UnitCount>` | `10.0000` | `LIEFERPOS` | `LFP_MENGE` | Cantidad recibida. |
| `<DestinationLocation>` | `DEP1_OS` | Fijo | — | **Valor fijo `"DEP1_OS"`** (Excel: 'Valor: "DEP1_OS"' — Depósito OPESSA). |
| `<SourceLocation>` | `DEP1_OS` | Fijo | — | **Valor fijo `"DEP1_OS"`** (Excel). |
| `<CostTotalAmount>` | `2500.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Costo total del artículo en la recepción. |
| `<UnitSalesAmount>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<SalesTotalAmount>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<Stock>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<DailyAverageSales>` | `0.0000` | — | — | `[UNKNOWN] - {0.0000} - {No figura en el documento de mapeo (hoja InventoryFiscalDoc detalle). ¿Se asume fijo 0?}` |
| `<SuggestedPurchaseOrder>` | `0.0000` | — | — | `[UNKNOWN] - {0.0000} - {No figura en el documento de mapeo (hoja InventoryFiscalDoc detalle). ¿Se asume fijo 0?}` |
| `<PickupCode>` | vacío | — | — | `[UNKNOWN] - {vacío} - {No figura en el documento de mapeo para InventoryFiscalDoc. En otros TLOG (Reception/Count) debe ser "S1"/"S2" desde PRISMA}` |
| `<LastUpdateDate>` | vacío | — | — | `[UNKNOWN] - {vacío} - {No figura en el mapeo para InventoryFiscalDoc}` |
| `<DifBME_ASNTypeID>` | vacío | — | — | `[UNKNOWN] - {vacío} - {No figura en el mapeo. Vacío en el XML}` |
| `<InventoryControlDocumentState>` (en detalle) | `4` | Fijo | — | Valor fijo `4` (Excel row 64 detalle FiscalDoc). |

---

## Resumen de Joins / Lookup necesarios

1. `LIEFERSCHEIN` → driver principal.
2. `LIEFERSCHEIN.LF_ID` → `LIEFER.LF_ID` para obtener `LF_VERT` (CUIT proveedor) y `LF_NAME`.
3. `LIEFERSCHEIN.LFS_ID` → `LIEFERPOS.LFS_ID` para el detalle.
4. `LIEFERPOS.KST_ID` → `KOSTST.KST_ID` para obtener `KST_CODE` (APIES).
5. `LIEFERPOS.ART_NR` → `ARTIKEL.ART_ID` para obtener `ART_NAME` (descripción).
6. `LIEFERPOS.VPK_ID1` → `VPCKEINH.VPK_ID` para validar la unidad de medida.
7. `LFS_INFO` (split por `|`) → `CAINumber` y `CAIDate`.

---

## 📌 Resumen de [UNKNOWN] (los que persisten)

| # | Campo / Concepto | Razón |
|---|---|---|
| 1 | `<Supplier>` | `LF_VERT` está vacío en el dataset de ejemplo. Validar carga del CUIT. |
| 2 | `<TaxAmount>` | Excel dice "LFS_" sin completar. Falta campo origen del impuesto interno. |
| 3 | `<IvaTaxAmount>` | Excel dice "Valor bruto del art_nr identificado como [vacío]". Falta criterio para identificar el artículo de percepción IVA. |
| 4 | `<IIBBTaxAmount>` | Excel dice "Valor bruto del art_nr identificado como [vacío]". Falta criterio para identificar el artículo de percepción IIBB. |
| 5 | `<DailyAverageSales>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 6 | `<SuggestedPurchaseOrder>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 7 | `<PickupCode>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 8 | `<LastUpdateDate>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 9 | `<DifBME_ASNTypeID>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |

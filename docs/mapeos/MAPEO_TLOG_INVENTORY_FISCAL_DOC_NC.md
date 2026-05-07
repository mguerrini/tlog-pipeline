# Mapeo TLOG_INVENTORY_FISCAL_DOC_NC

## Información General

- **Tipo de documento:** Documento Fiscal — Nota de Crédito (NC)
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

Se genera un TLOG `InventoryFiscalDoc` con `ReceiptType = NC` por cada registro de la tabla `LIEFERSCHEIN` que cumpla:

- `LFS_STATUS = 42` (documento imputado/cerrado)
- `LFS_RTS = 1` (return / devolución → indica Nota de Crédito)
- Montos `LFS_NETTO` y `LFS_BRUTTO` **negativos** (NC)
- `AKTIV = 'J'` (registro activo)

> Nota: En el set de datos de ejemplo (`Lieferschein_20260505.csv`) los registros con `LFS_RTS=1` y montos negativos (ej: `N-33333-33333333`, `A-NNNNN-NNNNNNNN`, `A-12345-12345679`) corresponden a **Notas de Crédito (NC)**.

> Diferencia clave con FC: las NC son devoluciones de mercadería, por lo cual los importes y cantidades viajan en negativo (`UnitCount`, `UnitBaseCostAmount`, `NetAmount`, `TotalAmount`, etc.).

---

## Mapeo de Cabecera (`<Transaction>` y `<InventoryControlTransaction>`)

| Campo XML (TLOG) | Valor en XML ejemplo | Origen / Tabla | Campo origen | Observaciones |
|---|---|---|---|---|
| `<RetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS. Tomar el `KST_CODE` del centro de costo asociado (a través de `LIEFERPOS.KST_ID`). |
| `<WorkstationID>` | `0` | Fijo | — | Valor fijo `0` (BME). |
| `<SequenceNumber>` (cabecera) | `000190000001` | Construir | — | Longitud 12, debe iniciar en `900000000001` para la APIES (SU). Construido por Bridge. |
| `<BusinessDayDate>` | `2026-05-05` | Nombre del archivo | — | Fecha que viene en el nombre del archivo (`AAAAMMDD`). |
| `<Period>` | `0` | Fijo | — | Valor fijo `0`. |
| `<Subperiod>` | `0` | Fijo | — | Valor fijo `0`. |
| `<PeriodCode>` | `0` (en XML) → vacío | vacío | — | Mapeo dice "vacío". Aunque el XML trae `0`, **prevalece el documento de mapeo**: va vacío. |
| `<SubPeriodCode>` | `0` (en XML) → vacío | vacío | — | Mapeo dice "vacío". Aunque el XML trae `0`, **prevalece el documento de mapeo**: va vacío. |
| `<BeginDateTime>` | (no presente en XML NC) | Construir | — | `BusinessDayDate + BEGIN_DATE_OFFSET` (valor desde configuración). Mapeo indica "Inicia 22:00:01". (En el XML del ejemplo de NC el tag está omitido — error del ejemplo, **debe enviarse igual** según las instrucciones comunes del proyecto). |
| `<EndDateTime>` | `2026-05-05 23:59:59` | Construir | — | `BusinessDayDate + END_DATE_OFFSET` (valor desde configuración). Mapeo indica "Termina 22:00:00". |
| `<OperatorID>` | `admin` | Fijo | — | Valor fijo `admin` (regla del proyecto). El XML trae `1`, debe enviarse `admin`. |
| `<OriginalTransaction>` | vacío | — | — | NO SE UTILIZA. Vacío. |
| `<SerialFormID>` | `000190000001` | Igual a `SequenceNumber` | — | Mismo contenido que `SequenceNumber` de cabecera. |
| `<DocumentTypeCode>` | `InventoryFiscalDoc` | Fijo | — | Valor fijo `InventoryFiscalDoc`. |
| `<InventoryControlDocumentState>` | `4` | `LIEFERSCHEIN` | `LFS_STATUS` | Para `INVENTORYFISCALDOC` debe ser **`4`**. En LIEFERSCHEIN viene `LFS_STATUS=42`. Mapeo `LFS_STATUS=42 → 4`. (XML trae `37`, prevalece mapeo Excel = `4`.) |
| `<contractReferenceNumber>` | vacío | — | — | Vacío. |
| `<CreateDateTimestamp>` | `2026-05-05 10:00:00.000 ART` | Construir | — | Mismo dato que `BeginDateTime` con formato milisegundos + ` ART`. |
| `<DestinationRetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | Igual a `RetailStoreID` (APIES de la EESS, a través de `LIEFERPOS.KST_ID`). |
| `<ExpectedDeliveryDate>` | `2026-05-05 00:00:00.0 ART` | Construir | — | Mismo dato que `BeginDateTime` con formato milisegundos + ` ART`. |
| `<ICDAmount>` | `-4000.0000` | `LIEFERSCHEIN` | `LFS_BRUTTO` | Costo total de la NC (importe bruto, **negativo** para NC). |
| `<LastUpdateDate>` | `2026-05-05 10:01:00.000 ART` | Construir | — | Mismo dato que `BeginDateTime` con formato milisegundos + ` ART`. |
| `<Originator>` | vacío | — | — | Vacío. |
| `<SourceRetailStore>` | `00019` | `KOSTST` | `KST_CODE` | APIES origen. **Debe coincidir con `RetailStoreID`** (Excel). Para NC en el XML coincide con `DestinationRetailStoreID` (mismo APIES). |
| `<Supplier>` | `30-0000000000-5` | `LIEFER` | `LF_VERT` | Código de proveedor (CUIT). `[UNKNOWN] - {LF_VERT vacío en dataset} - {Aunque el Excel confirma que el campo es LF_VERT, los datos de prueba traen LF_VERT vacío. Validar dónde se carga el CUIT del proveedor}` |
| `<OrderDocumentType>` | vacío | — | — | Vacío. |
| `<User>` | `admin` | Fijo | — | Valor fijo `admin` (regla del proyecto). |
| `<ICDQuantity>` | vacío | — | — | Vacío. |
| `<ICDTotSalesAmount>` | vacío | — | — | Vacío. |
| `<Frequency>` | vacío | — | — | Vacío. |
| `<InventoryAdjustmentType>` | vacío | — | — | Vacío. |
| `<ReceiptNumber>` | `REC202110-0000014` | `LIEFERSCHEIN` | `LFS_NAME` | Número de comprobante. Formato esperado ARCA: `A-XXXXX-XXXXXXXX` (`A/X/C-00000-00000000`). En los datos de ejemplo viene como `N-33333-33333333`, `A-NNNNN-NNNNNNNN`, `A-12345-12345679`. `[UNKNOWN] - {REC202110-0000014} - {En el XML el formato no respeta la máscara. ¿Se debe transformar LFS_NAME al formato A-XXXXX-XXXXXXXX antes de enviar? ¿O se envía tal cual viene de Bridge?}` |
| `<FiscalReceiptFlag>` | `false` | Fijo | — | Valor fijo `false` para `InventoryFiscalDoc`. |
| `<ReceiptType>` | `NC` | Fijo (por condición) | — | Valor fijo `NC` cuando se cumple la condición de registro de Nota de Crédito (ver sección "Condición de Registro"). |
| `<ReceiptDate>` | `2026-05-05 00:00:00.0 ART` | `LIEFERSCHEIN` | `LFS_DATUM` | Fecha del comprobante (carga manual). Formato `YYYY-MM-DD HH:MM:SS.0 ART`. |
| `<CAINumber>` | vacío en XML | `LIEFERSCHEIN` | `LFS_INFO` | Número de CAI. En `LFS_INFO` viene en formato `CAINUMBER\|FECHAVENCIMIENTO` (ej: `12345678911\|30/04/2026`). Tomar parte previa al `\|`. |
| `<CAIDate>` | vacío en XML | `LIEFERSCHEIN` | `LFS_INFO` | Fecha de vencimiento del CAI. En `LFS_INFO` viene en formato `CAINUMBER\|FECHAVENCIMIENTO`. Tomar parte posterior al `\|`. |
| `<PagesQuantity>` | `1` | Fijo | — | Valor fijo `1`. |
| `<NetAmount>` | `-4000.0000` | `LIEFERSCHEIN` | `LFS_NETTO` | Valor neto. **Negativo** para NC. |
| `<ExemptAmout>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<TaxAmount>` | `0.0000` | `LIEFERSCHEIN` | `LFS_???` | `[UNKNOWN] - {0.0000} - {Excel indica origen "LFS_" sin completar (impuesto interno). ¿Hay un campo en LIEFERSCHEIN para este valor o se calcula? Validar con Emma}` |
| `<VatAmount>` | `0.0000` | `LIEFERSCHEIN` | `LFS_MWST` | Importe de IVA base. **Negativo** para NC. |
| `<ServicesVATAmount>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<DifferencialVATAmount>` | `0.0000` | Fijo | — | Valor fijo `0`. (Validar con Napse). |
| `<IvaTaxAmount>` | `0.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Percepción de IVA. `[UNKNOWN] - {0.0000} - {Excel dice "Valor bruto del art_nr identificado como [vacío]" — frase incompleta. ¿Cuál es el criterio para identificar el artículo de percepción IVA? Validar con Emma}` |
| `<IIBBTaxAmount>` | `0.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Percepciones IIBB FC/NC. `[UNKNOWN] - {0.0000} - {Excel dice "Valor bruto del art_nr identificado como [vacío]" — frase incompleta. ¿Cuál es el criterio para identificar el artículo de percepción IIBB? Validar con Emma}` |
| `<TotalAmount>` | `-4000.0000` | `LIEFERSCHEIN` | `LFS_BRUTTO` | Total de la NC. **Negativo** para NC. |

---

## Mapeo de Detalle (`<inventoryControlDocumentMerchandiseLineItem>`)

Por cada registro de `LIEFERPOS` asociado al `LIEFERSCHEIN` (vía `LFS_ID`), se genera un `inventoryControlDocumentMerchandiseLineItem`. Las cantidades y costos viajan **en negativo** para NC.

| Campo XML (TLOG) | Valor en XML ejemplo | Origen / Tabla | Campo origen | Observaciones |
|---|---|---|---|---|
| `<RetailStoreID>` | `00019` | `KOSTST` | `KST_CODE` | APIES de la EESS (heredado de cabecera, a través de `LIEFERPOS.KST_ID` → `KOSTST`). |
| `<WorkstationID>` | `0` | Fijo | — | Valor fijo `0` (BME). |
| `<SequenceNumber>` (detalle) | `000190000001` | Construir | — | Igual al `SequenceNumber` de cabecera. Longitud 12 (SU). |
| `<DetSequenceNumber>` (debe llamarse así, en XML viene como `<SequenceNumber>`) | `1` | construir | — | **El campo debe llamarse `DetSequenceNumber`**. Numeración secuencial 1..N de los ítems dentro de la NC. Longitud 3 (tope 999 artículos por documento). |
| `<Item>` | `608` | `LIEFERPOS` | `ART_NR` | SKU / Código de artículo. |
| `<UomUnits>` | `1.0000` | `LIEFERPOS` | `VPK_ID1` | Código de unidad de medida (referencia a `VPCKEINH.VPK_ID`). |
| `<ItemBrand>` | `0` | Fijo | — | Valor fijo `0` (Excel: "MARCA" → fijo `0`). |
| `<ItemDescription>` | `Pechuga pollo entera` | `ARTIKEL` | `ART_NAME` | Descripción del artículo (JOIN `LIEFERPOS.ART_NR` = `ARTIKEL.ART_ID`). |
| `<UnitBaseCostAmount>` | `-4000.0000` | `LIEFERPOS` | `LFP_EKP / LFP_MENGE` | Costo unitario del artículo. Calcular como `LFP_EKP / LFP_MENGE`. **Negativo** en NC. NO se carga; se obtiene del Maestro de Artículos. |
| `<UnitCount>` | `-400.0000` | `LIEFERPOS` | `LFP_MENGE` | Cantidad devuelta. **Negativa** en NC. |
| `<DestinationLocation>` | `DEP1_OS` | Fijo | — | **Valor fijo `"DEP1_OS"`** (Excel: 'Valor: "DEP1_OS"' — Depósito OPESSA). |
| `<SourceLocation>` | `DEP1_OS` | Fijo | — | **Valor fijo `"DEP1_OS"`** (Excel). |
| `<CostTotalAmount>` | `1600000.0000` | `LIEFERPOS` | `LFP_BRUTTO` | Costo total del artículo en la NC. `[UNKNOWN] - {1600000.0000} - {En el XML viene positivo (1600000), pero UnitBaseCostAmount × UnitCount daría positivo (-4000 × -400 = 1600000). ¿Es correcto que CostTotalAmount viaje positivo en NC? Confirmar el signo}` |
| `<UnitSalesAmount>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<SalesTotalAmount>` | `-0.0000` | Fijo | — | Valor fijo `0`. |
| `<Stock>` | `0.0000` | Fijo | — | Valor fijo `0`. |
| `<DailyAverageSales>` | `0.0000` | — | — | `[UNKNOWN] - {0.0000} - {No figura en el documento de mapeo (hoja InventoryFiscalDoc detalle). ¿Se asume fijo 0?}` |
| `<SuggestedPurchaseOrder>` | `0.0000` | — | — | `[UNKNOWN] - {0.0000} - {No figura en el documento de mapeo (hoja InventoryFiscalDoc detalle). ¿Se asume fijo 0?}` |
| `<PickupCode>` | vacío | — | — | `[UNKNOWN] - {vacío} - {No figura en el documento de mapeo para InventoryFiscalDoc}` |
| `<LastUpdateDate>` | vacío | — | — | `[UNKNOWN] - {vacío} - {No figura en el mapeo para InventoryFiscalDoc}` |
| `<DifBME_ASNTypeID>` | vacío | — | — | `[UNKNOWN] - {vacío} - {No figura en el mapeo. Vacío en el XML}` |
| `<InventoryControlDocumentState>` (en detalle) | `4` | Fijo | — | Valor fijo `4` (Excel row 64 detalle FiscalDoc). |

---

## Resumen de Joins / Lookup necesarios

1. `LIEFERSCHEIN` → driver principal (filtrado por `LFS_RTS=1` y montos negativos para NC).
2. `LIEFERSCHEIN.LF_ID` → `LIEFER.LF_ID` para obtener `LF_VERT` (CUIT proveedor) y `LF_NAME`.
3. `LIEFERSCHEIN.LFS_ID` → `LIEFERPOS.LFS_ID` para el detalle.
4. `LIEFERPOS.KST_ID` → `KOSTST.KST_ID` para obtener `KST_CODE` (APIES).
5. `LIEFERPOS.ART_NR` → `ARTIKEL.ART_ID` para obtener `ART_NAME` (descripción).
6. `LIEFERPOS.VPK_ID1` → `VPCKEINH.VPK_ID` para validar la unidad de medida.
7. `LFS_INFO` (split por `|`) → `CAINumber` y `CAIDate`.

---

## Diferencias clave FC vs NC

| Aspecto | FC (Factura) | NC (Nota de Crédito) |
|---|---|---|
| `LFS_RTS` | distinto de `1` (NULL) | `1` |
| `LFS_NETTO` | positivo | negativo |
| `LFS_BRUTTO` | positivo | negativo |
| `<ReceiptType>` | `FC` | `NC` |
| `<ICDAmount>` | positivo | negativo |
| `<NetAmount>` | positivo | negativo |
| `<TotalAmount>` | positivo | negativo |
| `<UnitCount>` (detalle) | positivo | negativo |
| `<UnitBaseCostAmount>` (detalle) | positivo | negativo |
| `<CostTotalAmount>` (detalle) | positivo | `[UNKNOWN]` (en XML viene positivo aunque ambos factores son negativos) |

---

## 📌 Resumen de [UNKNOWN] (los que persisten)

| # | Campo / Concepto | Razón |
|---|---|---|
| 1 | `<Supplier>` | `LF_VERT` está vacío en el dataset de ejemplo. Validar carga del CUIT. |
| 2 | `<ReceiptNumber>` formato | En el XML de NC viene `REC202110-0000014` (formato no estándar). ¿Se debe transformar `LFS_NAME` al formato ARCA `A-XXXXX-XXXXXXXX`? |
| 3 | `<TaxAmount>` | Excel dice "LFS_" sin completar. Falta campo origen del impuesto interno. |
| 4 | `<IvaTaxAmount>` | Excel dice "Valor bruto del art_nr identificado como [vacío]". Falta criterio. |
| 5 | `<IIBBTaxAmount>` | Excel dice "Valor bruto del art_nr identificado como [vacío]". Falta criterio. |
| 6 | `<CostTotalAmount>` (detalle) | En el XML viene positivo aunque ambos factores son negativos. Confirmar signo. |
| 7 | `<DailyAverageSales>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 8 | `<SuggestedPurchaseOrder>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 9 | `<PickupCode>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 10 | `<LastUpdateDate>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |
| 11 | `<DifBME_ASNTypeID>` (detalle) | No figura en hoja FiscalDoc detalle del Excel. |

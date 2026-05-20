# Mapeo TLOG_INVENTORY_COUNT_REAL

> **Documento:** Conteo de inventario (relacionado con `InventoryAdjustment`)
> **Archivo XML de salida de referencia:** `TLOG_INVENTORY_COUNT_REAL_2020_07_21.xml`
> **Archivo XML revisado (proyecto):** `TLOG_INVENTORY_COUNT_REAL__revisado_.xml`
> **Hoja de referencia en Excel:** `InventoryCount` — `ANALISIS_DE_CAMPOS_EN_TLOG_OCPRA__Mapeo_a_completarv2.xlsx`
> **Última actualización:** 2026-05-20

---

## Nota general

> Validar con negocio si el `InventoryCount` se genera siempre que se realiza un ajuste, o si puede generarse de forma independiente.

---

## 🎯 Tabla driver y condición de registro

| Concepto | Valor |
|---|---|
| **Tabla driver de cabecera** | `HisVerbrauch` |
| **Tabla driver de detalle** | `HisVerbrauchpos` (JOIN por `VBR_ID`) |
| **Condición de registro (cabecera)** | `vbr_status = 2` (Excel: `DOCUMENTTYPECODE` campo `vbr_status=2`) |
| **Condición de registro (estado documento)** | `vbr_status = 3` (Excel: `INVENTORYCONTROLDOCUMENTSTATE` campo `vbr_status=3`) |

> **Cambio respecto a versión anterior:** La tabla driver es `HisVerbrauch` / `HisVerbrauchpos`, **no** `INVENTUR` / `INVPOSART`. El Excel lo define explícitamente en ambas secciones (cabecera y detalle).

---

## 📋 Comunes a todos los TLOG (header de Transaction)

| Campo             | Valor / Origen                                                 |
|-------------------|----------------------------------------------------------------|
| `BusinessDayDate` | Fecha que viene en el nombre del archivo (`AAAAMMDD`)          |
| `Period`          | `0` (fijo)                                                     |
| `Subperiod`       | `0` (fijo)                                                     |
| `PeriodCode`      | `0` (fijo)                                                     |
| `SubperiodCode`   | `0` (fijo)                                                     |
| `BeginDateTime`   | `BusinessDayDate + BEGIN_DATE_OFFSET` (valor de configuración) |
| `EndDateTime`     | `BusinessDayDate + END_DATE_OFFSET` (valor de configuración)   |
| `OperatorID`      | `por configuracion` (fijo)                                     |

### Regla de SEQUENCENUMBER (TLOG OCPRA = SU)

- **SEQUENCENUMBER:** Misma forma que los otros archivos
- **DET_SEQUENCENUMBER:** Misma forma que los otros archivos

---

## 🔹 CABECERA — `<Transaction>` + `<InventoryControlTransaction>`

> **Driver:** `HisVerbrauch`
 
Los siguientes campos se calculan igual que en los documentos actuales
RetailStoreID
WorkstationID
SequenceNumber
BusinessDayDate
Period
Subperiod
PeriodCode
SubPeriodCode
BeginDateTime
EndDateTime
OperatorID
OriginalTransaction
SerialFormID
> 
> 
| Campo TLOG | Valor en XML salida        | Campo origen                          | Tabla | Observación                                                                                                                                   |
|---|----------------------------|---------------------------------------|---|-----------------------------------------------------------------------------------------------------------------------------------------------|
| `InventoryControlDocumentState` | `4`                        | `vbr_status = 3`                      | `HisVerbrauch` | Estado del documento. Para conteo es código `4`. El campo también aparece en el BMC en formato `"SEQUENCE - ESTADO"` — validar con negocio    |
| `contractReferenceNumber` | `WST2005-00005`            | `vbr_name`                            | `HisVerbrauch` | Descripción del conteo. Viene del nombre del registro en `HisVerbrauch`                                                                       |
| `CreateDateTimestamp` | `2020-07-20 10:00:00.000`  | `chg_zeit`                            | `HisVerbrauch` | Fecha/hora con formato milisegundos (`yyyy-MM-dd HH:mm:ss.SSS`). Validar con Emma                                                             |
| `DestinationRetailStoreID` | `= RetailStoreID`          | `kst_code`                            | `KOSTST` | Mismo valor que `RetailStoreID` (Excel: "MISMO VALOR QUE RETAILSTOREID")                                                                      |
| `ExpectedDeliveryDate` | `2020-07-20 00:00:00.000`  | `chg_zeit`                            | `HisVerbrauch` | Mismo dato que `CreateDateTimestamp`, formato milisegundos. Validar con Emma                                                                  |
| `ICDAmount` | `203200.0000`              | `SUM(vbt_menge × vbt_wes)`            | `HisVerbrauchpos` | Costo total del conteo. Suma de todas las líneas del `VBR_ID`. JOIN con `HisVerbrauch` por `VBR_ID`                                           |
| `LastUpdateDate` | `2020-07-20 10:01:00.0000` | `chg_zeit`                            | `HisVerbrauch` | Mismo dato que `CreateDateTimestamp`, formato milisegundos. Validar con Emma                                                                  |
| `SourceRetailStore` | `= RetailStoreID`          | `kst_code`                            | `KOSTST` | Mismo APIES que `RetailStoreID`                                                                                                               |
| `Supplier` | vacío                      | fijo                                  | fijo | Null para ajustes/mermas (cuando tabla driver = `HisVerbrauch`)                                                                               |
| `OrderDocumentType` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `User` | = OperatorID               | fijo                                  | fijo | Constante `admin` (mismo que `OperatorID`)                                                                                                    |
| `ICDQuantity` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `ICDTotSalesAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `Frequency` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `InventoryAdjustmentType` | valor                      | `vrt_id`                              | `HisVerbrauch` | Tipo de ajuste. Valores válidos: `JUSTIFIED_ADJUSTMENTS`, `UNJUSTIFIED_ADJUSTMENTS`, `CORRECTIVE_ADJUSTMENT`. El valor se resuelve desde `vrt_id` |
| `ReceiptNumber` | vacío                      | fijo                                  | fijo | Null para conteo (cuando tabla driver = `HisVerbrauch`)                                                                                       |
| `FiscalReceiptFlag` | `false`                    | fijo                                  | fijo | Siempre `false`                                                                                                                               |
| `ReceiptType` | vacío                      | fijo                                  | fijo | Null. Validar con Emma                                                                                                                        |
| `ReceiptDate` | `2020-07-20 00:00:00.000`  | `chg_zeit`                            | `HisVerbrauch` | Fecha de creación (carga manual), con formato `yyyy-MM-dd HH:mm:ss.SSS`                                                                       |
| `CAINumber` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `CAIDate` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `PagesQuantity` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `NetAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `ExemptAmout` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `TaxAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `VatAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `ServicesVATAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `DifferencialVATAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `IvaTaxAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `IIBBTaxAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| `TotalAmount` | vacío                      | fijo                                  | fijo | No se usa en este documento                                                                                                                   |
| Estado interno | `REPLICADO`                | fijo                                  | fijo | Flag interno del proceso. No es un nodo XML del archivo                                                                                       |

---

## 🔸 DETALLE — `<inventoryControlDocumentMerchandiseLineItem>`

> **Driver:** `HisVerbrauchpos` (JOIN `HisVerbrauch.VBR_ID = HisVerbrauchpos.VBR_ID`)

| Campo TLOG | Valor en XML salida | Campo origen | Tabla | Observación |
|---|---|---|---|---|
| `RetailStoreID` | `= cabecera` | `kst_code` | `KOSTST` | Mismo que cabecera |
| `WorkstationID` | `0` | fijo | fijo | PDV (0 = BME) |
| `SequenceNumber` | `= cabecera` | construir | Construir | Idem cabecera. Idem `InventoryReception` |
| `DetSequenceNumber` | `1, 2, 3...` | `vbt_pos` | `HisVerbrauchpos` | Enumeración de ítems dentro del conteo. JOIN con `HisVerbrauch` por `VBR_ID` |
| `Item` | `566` | `art_nr` (join) | `HisVerbrauchpos` | SKU / código de artículo. JOIN `HisVerbrauchpos.art_id = ARTIKEL.art_id → art_nr` |
| `UomUnits` | `1.0000` | `vpk_id` | `HisVerbrauchpos` | Código de unidad de medida. JOIN `HisVerbrauchpos.vpk_id = VPCKEINH.vpk_id → vpk_name` |
| `ItemBrand` | vacío | fijo | fijo | **Valor fijo vacío** (Excel: "MARCA → vacío → fijo") |
| `ItemDescription` | `Agua mineral sin gas` | `art_nr` (join) | `HisVerbrauchpos` | Descripción del artículo. JOIN `HisVerbrauchpos.art_id = ARTIKEL.art_id → art_name` |
| `UnitBaseCostAmount` | `10.0000` | `vbt_wes` | `HisVerbrauchpos` | Costo unitario del artículo. Excel: "NO se carga, se obtiene del Maestro de Artículos" |
| `UnitCount` | `150.0000` | fijo | fijo | **Valor fijo `0`** (Excel: "STOCK REAL → 0 → fijo"). El XML de salida trae valores reales — prevalece el documento de mapeo: va `0` |
| `DestinationLocation` | `DEP1_OS` | `kst_id` | `HisVerbrauch` | Valor: `"DEP1_OS"` (Depósito OPESSA). Resuelto desde `kst_id` en `HisVerbrauch` |
| `SourceLocation` | `DEP1_OS` | `kst_id` | `HisVerbrauch` | Valor: `"DEP1_OS"` (Depósito OPESSA) |
| `CostTotalAmount` | `1500.0000` | `vbt_wes` | `HisVerbrauchpos` | Costo total artículo conteo. Excel: "VIAJA EL MISMO QUE ARTÍCULO UNITARIO" → mismo valor que `UnitBaseCostAmount` (`vbt_wes`) |
| `UnitSalesAmount` | `0.0000` | fijo | fijo | **Valor fijo `0`** (Excel: "PRECIO VENTA ARTÍCULO UNITARIO CONTEO → 0 → fijo") |
| `SalesTotalAmount` | `0.0000` | fijo | fijo | No se usa en este documento. Fijo `0.0000` |
| `Stock` | `0.0000` | `[UNKNOWN]` | `[UNKNOWN]` | `[UNKNOWN] - {0.0000} - {Excel: "STOCK TEÓRICO INICIAL — Previo al Conteo Manual" pero NO especifica campo origen en HisVerbrauch/HisVerbrauchpos. ¿Campo equivalente a INP_SOLL de INVPOSART? Validar con OCPRA}` |
| `DailyAverageSales` | `0.0000` | fijo | fijo | No se usa en este documento. Fijo `0.0000` |
| `SuggestedPurchaseOrder` | `0.0000` | fijo | fijo | Siempre `0.0000` |
| `PickupCode` | vacío | fijo | fijo | **Valor fijo `S1`** según Excel ("S1 maneja stock — S2 no maneja stock → fijo"). El XML de salida lo trae vacío — **prevalece el documento de mapeo**: va `S1` fijo. Validar con OCPRA si puede ser `S2` para algún artículo |
| `LastUpdateDate` | `2020-07-20 10:01:00.0000` | `chg_zeit` | `HisVerbrauch` | Fecha de fin de registro de la operación, formato `yyyy-MM-dd HH:mm:ss.SSSS` |
| `DifBME_ASNTypeID` | vacío | fijo | fijo | No se usa en este documento |
| `InventoryControlDocumentState` | `4` | fijo | fijo | Estado del documento. Para conteo es código `4` |

---


## 📌 [UNKNOWN] pendientes de validación

| Campo | Razón                                                                                                                                                                                    |
|---|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Stock` (detalle) | Excel define "STOCK TEÓRICO INICIAL — Previo al Conteo Manual" pero no especifica el campo origen en `HisVerbrauch` / `HisVerbrauchpos`. Poner UNKNOWN A DEFINIR                                  |
| `PickupCode` (detalle) | Excel dice fijo `S1`, pero no aclara si puede ser `S2` para artículos que no manejan stock. Validar si es siempre `S1` o depende de algún atributo del artículo. Poner UNKNOWN A DEFINIR |
| `InventoryAdjustmentType` | Se resuelve desde `vrt_id` de `HisVerbrauch` pero el Excel no define la tabla de traducción. Poner  UNKNOWN A DEFINIR                                                                              |

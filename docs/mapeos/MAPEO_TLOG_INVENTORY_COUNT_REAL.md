# Mapeo TLOG_INVENTORY_COUNT_REAL

> **Documento:** Conteo de inventario (relacionado con `InventoryAdjustment`)
> **Archivo XML:** `TLOG_INVENTORY_COUNT_REAL__revisado_.xml`
> **Hoja de referencia en Excel:** `InventoryCount`

---

## 🎯 Tabla driver y condición de registro

| Concepto | Valor |
|---|---|
| **Tabla driver de cabecera** | `INVENTUR` |
| **Tabla driver de detalle** | `INVPOSART` (relacionada por `INV_ID`) |
| **Condición de registro** | `INV_STATUS = 8 AND INV_TYP = 4` (resuelto desde Excel hoja `InventoryAdjusment (2)` row 19; el conteo se relaciona con el ajuste y comparte la misma condición de cierre) |

> **Nota:** En el dataset de ejemplo (`Inventur_20260505.csv`) el registro que cumple esta condición es `INV_ID = 6` (con `INV_STATUS=8`, `INV_TYP=4`).

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

## 🔹 CABECERA — `TLOG_OCPRA_CABECERA`

> **Driver:** `INVENTUR`

| Campo TLOG | Valor | Campo origen | Tabla | Observación de mapeo |
|---|---|---|---|---|
| RETAILSTOREID | `0000015` | `KST_CODE` | `KOSTST` | Mapea por `INVENTUR.KST_ID = KOSTST.KST_ID` |
| WORKSTATIONID | `0` | fijo | fijo | PDV (0 es BME) |
| SEQUENCENUMBER | `900000000001` | construir (SU) | Construir | Numérico longitud **12**, comenzando por `9` (regla SU del Excel hoja `SEQUENCENUMBER`) |
| BUSINESSDAYDATE | `2026-05-05` | nombre del archivo | Construir | Fecha que viene en el nombre del archivo (`AAAAMMDD`). Se obtiene del WS de Periodos y Turnos en Bridge |
| PERIODO | `0` | fijo | fijo | Validar con Emma |
| SUBPERIOD | `0` | fijo | fijo | Validar con Emma |
| PERIODCODE | vacío | vacío | — | Debe obtenerse del WS de Periodos y Turnos en Bridge |
| SUBPERIODCODE | vacío | vacío | — | Debe obtenerse del WS de Periodos y Turnos en Bridge |
| BEGINDATETIME | `2026-05-05 22:00:01` | `BusinessDayDate + BEGIN_DATE_OFFSET` | Construir | Valor de inicio de día, según configuración |
| ENDDATETIME | `2026-05-06 22:00:00` | `BusinessDayDate + END_DATE_OFFSET` | Construir | Valor de cierre de día, según configuración |
| OPERATORID | `admin` | fijo | fijo | Constante `admin` (regla del proyecto). El XML de muestra trae `1`, debe enviarse `admin` |
| SERIALFORMID | `=SEQUENCENUMBER` | `=SEQUENCENUMBER` | — | Mismo valor que SEQUENCENUMBER |
| DOCUMENTTYPECODE | `InventoryCount` | fijo | fijo | Tipo de documento |
| INVENTORYCONTROLDOCUMENTSTATE | `4` | fijo | fijo | Para conteo es código `4` (Excel) |
| CONTRACREFERENCENUMBER | `Generado desde Web` | `INV_INFO` o fijo | `INVENTUR` | `[UNKNOWN] - {Generado desde Web} - {El XML trae literal "Generado desde Web". El Excel dice "DESCRIPCIÓN DEL CONTEO" sin definir campo origen. ¿Se usa literal "Generado desde Web" o sale de INV_INFO? En el dataset INV_INFO trae el nombre del inventario (INV2605-0006), no descripción}` |
| CREATEDATETIMESTAMP | `=BENGINDATETIME` | `=BENGINDATETIME` | Construir | Mismo dato que inicio de registro con formato milisegundo |
| DESTINATIONRETAILSTOREID | `=RETAILSTOREID` | `=RETAILSTOREID` | — | **Mismo valor que RetailStoreID** (Excel: "MISMO VALOR QUE RETAILSTOREID"). El XML con `00009` es **error del ejemplo** |
| EXPECTEDDELIVERYDATE | `=BENGINDATETIME` | `=BENGINDATETIME` | Construir | Mismo dato que inicio de registro con formato milisegundo |
| ICDAMOUNT | `203200.0000` | `SUM(INP_IST × INP_EKP)` | `INVPOSART` | Costo total del conteo (suma del detalle) |
| LASTUPDATEDATE | `=BENGINDATETIME` | `=BENGINDATETIME` | Construir | Mismo dato que inicio de registro con formato milisegundo |
| SOURCERETAILSTORE | `=RETAILSTOREID` | `=RETAILSTOREID` | — | Mismo APIES que RetailStoreID |
| SUPPLIER | null | fijo | fijo | En null para conteos |
| ORDERDOCUMENTTYPE | null | — | — | No se usa en este documento |
| USUARIO | `admin` | `=OPERATORID` | fijo | Usuario que generó el registro |
| ICDQUANTITY | null | — | — | No se usa en este documento |
| ICDTOTSALESAMOUNT | null | — | — | No se usa en este documento |
| FREQUENCY | null | — | — | No se usa en este documento |
| INVENTORYADJUSTMENTTYPE | `CORRECTIVE_ADJUSTMENT` | fijo | fijo | Tipo de ajuste para conteo. Excel da las opciones válidas: `JUSTIFIED_ADJUSTMENTS`, `UNJUSTIFIED_ADJUSTMENTS`, `CORRECTIVE_ADJUSTMENT`. Por defecto se usa `CORRECTIVE_ADJUSTMENT` (es el valor que trae el XML) |
| RECEIPTNUMBER | null | fijo | fijo | En null para conteo |
| FISCALRECEIPTFLAG | `false` | fijo | fijo | Siempre false |
| RECEIPTTYPE | null | fijo | fijo | NULL |
| RECEIPTDATE | `2026-05-05` | `INV_DATUM` | `INVENTUR` | Fecha del inventario |
| CAINUMBER | null | — | — | No se usa en este documento |
| CAIDATE | null | — | — | No se usa en este documento |
| PAGESQUANTITY | null | — | — | No se usa en este documento |
| NETAMOUNT | null | — | — | No se usa en este documento |
| EXEMPTAMOUNT | null | — | — | No se usa en este documento |
| TAXAMOUNT | null | — | — | No se usa en este documento |
| VATAMOUNT | null | — | — | No se usa en este documento |
| SERVICESVATAMOUNT | null | — | — | No se usa en este documento |
| DIFFERENCIALVATAMOUNT | null | — | — | No se usa en este documento |
| IVATAXAMOUNT | null | — | — | No se usa en este documento |
| IIBBTAXAMOUNT | null | — | — | No se usa en este documento |
| TOTALAMOUNT | null | — | — | No se usa en este documento |
| ESTADO | `REPLICADO` | fijo | fijo | "REPLICADO" |

---

## 🔸 DETALLE — `TLOG_OCPRA_DETALLE`

> **Driver:** `INVPOSART` (filtrado por `INV_ID` de la cabecera)

| Campo TLOG | Valor | Campo origen | Tabla | Observación de mapeo |
|---|---|---|---|---|
| RETAILSTOREID | =cab | =cab | — | Mismo que cabecera |
| WORKSTATIONID | `0` | fijo | fijo | PDV (0 es BME) |
| SEQUENCENUMBER | =cab | — | Construir | Idem cabecera |
| DET_SEQUENCENUMBER | `1, 2, 3...` | secuencial | Construir | Numérico (3), de 1 a 999 por documento |
| ITEM | `566` | `ART_NR` | `ARTIKEL` | Relaciona `INVPOSART.ART_ID = ARTIKEL.ART_ID` → `ART_NR` |
| UOMUNITS | `1.0000` | `VPK_ID` | `INVPOSART` | Relaciona con `VPCKEINH.VPK_ID` |
| ITEMBRAND | `0` | fijo | fijo | Valor fijo `0` (Excel: "MARCA" → no se maneja marca, fijo `0`) |
| ITEMDESCRIPTION | `Agua mineral sin gas` | `ART_NAME` | `ARTIKEL` | Vía `INVPOSART.ART_ID = ARTIKEL.ART_ID` |
| UNITBASECOSTAMOUNT | `10.0000` | `INP_EKP` | `INVPOSART` | Costo unitario al momento del conteo. NO se carga, se obtiene del Maestro de Artículos |
| UNITCOUNT | `150.0000` | `INP_IST` | `INVPOSART` | Stock real contado |
| DESTINATIONLOCATION | `DEP1_OS` | fijo | fijo | **Valor fijo `"DEP1_OS"`** (Excel: 'Valor: "DEP1_OS"' — Depósito OPESSA) |
| SOURCELOCATION | `DEP1_OS` | fijo | fijo | **Valor fijo `"DEP1_OS"`** (Excel) |
| COSTTOTALAMOUNT | `1500.0000` | `INP_IST × INP_EKP` | `INVPOSART` | `[UNKNOWN] - {1500.0000} - {El Excel dice "VIAJA EL MISMO QUE ARTICULO UNITARIO" (= UnitBaseCostAmount), pero la lógica natural es UnitCount × UnitBaseCostAmount. ¿Cuál es el cálculo correcto? Validar con Emma}` |
| UNITSALESAMOUNT | `0.0000` | `INP_VKP` | `INVPOSART` | Precio venta unitario del conteo |
| SALESTOTALAMOUNT | `0.0000` | fijo | fijo | No se usa en este documento |
| STOCK | `0.0000` | `INP_SOLL` | `INVPOSART` | **Stock teórico inicial** (previo al conteo manual). Excel row 70: "STOCK TEORICO INICIAL - Previo al Conteo Manual". El XML de muestra trae `0.0000` por ser ejemplo poco preciso; el valor real debe ser `INP_SOLL` |
| DAILYAVERAGESALES | `0.0000` | fijo | fijo | No se usa en este documento |
| SUGGESTEDPURCHASEORDER | `0.0000` | fijo | fijo | Siempre se carga en cero |
| PICKUPCODE | vacío | `[UNKNOWN]` | `[UNKNOWN]` | `[UNKNOWN] - {vacío} - {Marca de stock del artículo a ajustar (S1 maneja stock, S2 no maneja stock). El Excel define qué representa pero NO especifica el campo origen en el maestro de artículos de PRISMA}` |
| LASTUPDATEDATE | `=cab.ENDDATETIME` | `=cab.ENDDATETIME` | — | Fecha de fin de registro de la operación |
| DIFBME_ASNTYPEID | null | — | — | No se usa en este documento |
| INVENTORYCONTROLDOCUMENTSTATE | `4` | fijo | fijo | Para conteo es código `4` |

---

## 📌 Resumen de [UNKNOWN] (los que persisten)

| Campo / Concepto | Razón |
|---|---|
| **CONTRACREFERENCENUMBER** | El XML trae literal "Generado desde Web". Excel dice "DESCRIPCIÓN DEL CONTEO" sin definir campo origen. ¿Literal o `INV_INFO`? |
| **COSTTOTALAMOUNT** | Excel dice "viaja el mismo que articulo unitario" pero la lógica natural es `cantidad × unitario`. Confirmar |
| **PICKUPCODE** | Excel define qué representa (S1/S2 marca de stock) pero no especifica el campo origen en el maestro de artículos de PRISMA |

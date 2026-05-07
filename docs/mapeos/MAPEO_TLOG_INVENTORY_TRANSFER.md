# Mapeo TLOG_INVENTORY_TRANSFER_REAL

> **Documento:** Transferencia de mercadería entre EESS (origen → destino)
> **Archivo XML:** `TLOG_INVENTORY_TRANSFER_REAL__revisado_.xml`
> **Hoja de referencia en Excel:** `InventoryTransfer`

---

## 🎯 Tabla driver y condición de registro

| Concepto | Valor |
|---|---|
| **Tabla driver de cabecera** | `LAGERBEW` |
| **Tabla driver de detalle** | `[UNKNOWN] - {Tabla detalle de LAGERBEW} - {No está provisto el CSV de LAGERBEW ni su tabla de detalle. ¿Existe LAGERBEWPOS o equivalente? Validar con OCPRA / Emma}` |
| **Condición de registro** | `[UNKNOWN] - {Cuándo se emite el TLOG_INVENTORY_TRANSFER} - {El documento de mapeo no especifica la condición. Validar con negocio: ¿se emite por cada cabecera de LAGERBEW que pase a un estado determinado? ¿Se filtra por estado, AKTIV, EXP_NR u otro flag?}` |

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

## 🔸 CABECERA — `<Transaction>` + `<InventoryControlTransaction>` (driver: `LAGERBEW`)

| Campo TLOG | Valor en XML | Campo origen | Tabla | Obs mapeo |
|---|---|---|---|---|
| `RETAILSTOREID` | `00019` | `KST_CODE` | `KOSTST` | APIES de la EESS origen. `[UNKNOWN] - {00019} - {¿Cómo se relaciona LAGERBEW con KOSTST para obtener la EESS origen? ¿LAGERBEW.KST_ID? Validar campo en LAGERBEW}` |
| `WORKSTATIONID` | `0` | fijo | fijo | PDV (0 = BME) |
| `SEQUENCENUMBER` | `000190000003` | construido | Construir | Numérico longitud 12, comenzando por `9` para la APIES (SU). **Observación XML:** "Tiene que iniciar en 9 con un formato de 12 dígitos" |
| `BUSINESSDAYDATE` | `2026-05-05` | nombre del archivo | Construir | Fecha que viene en el nombre del archivo (`AAAAMMDD`). **El Excel dice:** "DEBE SER OBTENIDO DEL WS DE PERIODOS Y TURNOS EN BRIDGE" |
| `PERIOD` | `0` | fijo | fijo | Valor fijo `0` (regla común) |
| `SUBPERIOD` | `0` | fijo | fijo | Valor fijo `0` (regla común) |
| `PERIODCODE` | `0` | vacío | — | El Excel dice: "PERIODO - DEBE SER OBTENIDO DEL WS DE PERIODOS Y TURNOS EN BRIDGE" |
| `SUBPERIODCODE` | `0` | vacío | — | El Excel dice: "TURNO - DEBE SER OBTENIDO DEL WS DE PERIODOS Y TURNOS EN BRIDGE" |
| `BEGINDATETIME` | `2026-05-05 22:00:01` | calculado | Construir | `BusinessDayDate + BEGIN_DATE_OFFSET` (valor de configuración) |
| `ENDDATETIME` | `2026-05-06 22:00:00` | calculado | Construir | `BusinessDayDate + END_DATE_OFFSET` (valor de configuración) |
| `OPERATORID` | `admin` | fijo | fijo | Valor fijo: `admin` (regla del proyecto). El XML real trae `1`, debe enviarse `admin` |
| `ORIGINALTRANSACTION` | vacío | — | — | No se usa |
| `SERIALFORMID` | `000190000003` | `=SEQUENCENUMBER` | — | Mismo valor que `SEQUENCENUMBER`. **Observación XML:** "Tiene que iniciar en 9 con un formato de 12 dígitos" |
| `DOCUMENTTYPECODE` | `InventoryTransfer` | fijo | fijo | Literal `"InventoryTransfer"` |
| `INVENTORYCONTROLDOCUMENTSTATE` | `2` | fijo | fijo | Excel: "ESTADO DEL DOCUMENTO (EN CASO LAS TRANSFERENCIAS ES EL CODIGO 2)". **El XML real trae 4 pero el Excel dice 2 → priorizar Excel = `2`** |
| `CONTRACREFERENCENUMBER` | vacío | fijo | fijo | Excel: "VIENE EN NULL EN TRANSFERENCIAS" |
| `CREATEDATETIMESTAMP` | `2026-05-05 10:00:00.000 ART` | `=BeginDateTime` (con ms + ART) | Construir | Excel: "TIENE EL MISMO DATO QUE INICIO DE REGISTRO CON FORMATO MILISEGUNDO" |
| `DESTINATIONRETAILSTOREID` | `99010` | `KST_CODE` | `KOSTST` | APIES de la EESS destino. `[UNKNOWN] - {99010} - {¿Qué campo de LAGERBEW indica la EESS destino? ¿LAGERBEW.KST_ID_DEST o similar? Validar con OCPRA / Emma}` |
| `EXPECTEDDELIVERYDATE` | `2026-05-05 00:00:00.0 ART` | `=BeginDateTime` (con ms + ART) | Construir | Excel: "TIENE EL MISMO DATO QUE INICIO DE REGISTRO CON FORMATO MILISEGUNDO" |
| `ICDAMOUNT` | `0` | calculado | Detalle | "MONTO TOTAL DE COSTO TRANSFERIDO". `[UNKNOWN] - {0} - {¿Se calcula como SUM(CostTotalAmount) del detalle? El XML real trae 0 pero la lógica esperada sería sumar el detalle. Validar}` |
| `LASTUPDATEDATE` | `2026-05-05 10:01:00.000 ART` | `=BeginDateTime` (con ms + ART) | Construir | Excel: "TIENE EL MISMO DATO QUE INICIO DE REGISTRO CON FORMATO MILISEGUNDO" |
| `ORIGINATOR` | vacío | — | — | No se usa |
| `SOURCERETAILSTORE` | `00019` | `=RETAILSTOREID` | — | APIES origen, igual a `RETAILSTOREID` |
| `SUPPLIER` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `ORDERDOCUMENTTYPE` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `USER` | `admin` | fijo | fijo | Valor fijo: `admin` (regla del proyecto). Excel: "USUARIO GENERO ORIGEN TRANSFERENCIA". El XML real trae `1`, debe enviarse `admin` |
| `ICDQUANTITY` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `ICDTOTSALESAMOUNT` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `FREQUENCY` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `INVENTORYADJUSTMENTTYPE` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `RECEIPTNUMBER` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "viaja en null para los inventory transfer". El XML real trae `0000001` pero **prevalece Excel = vacío/null** |
| `FISCALRECEIPTFLAG` | `false` | fijo | fijo | Excel: "SIEMPRE EN FALSE" |
| `RECEIPTTYPE` | vacío | vacío | — | Excel: "NULL". `[UNKNOWN] - {vacío} - {Excel marca "VALIDAR CON EMMA"}` |
| `RECEIPTDATE` | `2026-05-05 00:00:00.0 ART` | fecha creación | Construir | Excel: "FECHA CREACION (CARGA MANUAL)". `[UNKNOWN] - {2026-05-05 00:00:00.0 ART} - {¿De qué campo de LAGERBEW sale la fecha de creación de la transferencia? ¿LAGERBEW.NEW_ZEIT? Validar}` |
| `CAINUMBER` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `CAIDATE` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `PAGESQUANTITY` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer". El XML real trae `1` pero **prevalece Excel = vacío/null** |
| `NETAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `EXEMPTAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `TAXAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `VATAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `SERVICESVATAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `DIFFERENCIALVATAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `IVATAXAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `IIBBTAXAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |
| `TOTALAMOUNT` | vacío | vacío | — | Excel: "NO SE USA EN ESTE DOCUMENTO" + obs XML: "Viaja en null para los inventory transfer" |

---

## 🔹 DETALLE — `<inventoryControlDocumentMerchandiseLineItem>` (driver: `[UNKNOWN]` — detalle de LAGERBEW)

> **Aclaración previa:** El XML actual nombra el campo de detalle como `<SequenceNumber>`, pero el Excel y la observación del XML indican que el campo correcto es `<DetSequenceNumber>`. **Esto es un cambio a aplicar en el contrato.**

| Campo TLOG | Valor en XML | Campo origen | Tabla | Obs mapeo |
|---|---|---|---|---|
| `RETAILSTOREID` | =cab | `=RETAILSTOREID` cab | — | APIES origen, hereda de la cabecera |
| `WORKSTATIONID` | =cab | fijo | fijo | `0` |
| `SEQUENCENUMBER` | =cab | =cab | Construir | Mismo valor que el `SEQUENCENUMBER` de cabecera |
| `DET_SEQUENCENUMBER` | `1, 2, 3, …` | secuencial | Construir | "SEQUENCE ENUMARACIÓN DE ITEMS DENTRO DE LA RECEPCIÓN". Numérico longitud 3, del 1 al 999 por cada documento. **El XML actual lo llama `SequenceNumber`, hay que renombrar a `DetSequenceNumber`.** |
| `ITEM` | `636` | `ART_NR` (o `ART_NUMMER`) | `ARTIKEL` | SKU / Código de artículo. `[UNKNOWN] - {636} - {¿Qué campo del detalle de LAGERBEW se relaciona con ARTIKEL? Validar}` |
| `UOMUNITS` | `1.0000` | `VPK_ID` | `VPCKEINH` | Código unidad de medida. `[UNKNOWN] - {1.0000} - {¿Qué campo del detalle de LAGERBEW indica la unidad de medida (VPK)? Validar}` |
| `ITEMBRAND` | `0` | fijo | fijo | **Valor fijo `0`** (Excel: "MARCA" → fijo `0`, no se maneja marca, alineado con resto de TLOG). El XML real trae nombre de marca (`Aramark`) pero según el Excel y consistencia con otros TLOG (Reception, Return, FC, NC, Count, Adjustment) va `0` |
| `ITEMDESCRIPTION` | `-BOMBON COSTA VIZZIO 432GR` | `ART_NAME` | `ARTIKEL` | Descripción del artículo. JOIN `<detalle_LAGERBEW>.ART_ID = ARTIKEL.ART_ID` |
| `UNITBASECOSTAMOUNT` | `3.0000` | `ART_OEKP` | `ARTIKEL` | Costo unitario del artículo. **Excel: "NO se carga, se obtiene del Maestro de Artículos"** → `ARTIKEL.ART_OEKP`. JOIN `<detalle_LAGERBEW>.ART_ID = ARTIKEL.ART_ID` |
| `UNITCOUNT` | `3.0000` | cantidad transferida | detalle de LAGERBEW | Cantidad a transferir. `[UNKNOWN] - {3.0000} - {¿Qué campo del detalle de LAGERBEW indica la cantidad transferida? Validar}` |
| `DESTINATIONLOCATION` | `DEP1_OS` | fijo | fijo | Excel: 'Valor: "DEP1_OS"' — "Depósito OPESSA" |
| `SOURCELOCATION` | `DEP1_OS` | fijo | fijo | Excel: 'Valor: "DEP1_OS"' — "Depósito OPESSA" |
| `COSTTOTALAMOUNT` | `9.0000` | calculado | calculado | "COSTO TOTAL ARTICULOS A TRANSFERIR". Calculado como `UnitBaseCostAmount × UnitCount` |
| `UNITSALESAMOUNT` | `0.0000` | `ART_VKP` | `ARTIKEL` | "PRECIO VENTA ARTICULO UNITARIO". Sale de `ARTIKEL.ART_VKP` |
| `SALESTOTALAMOUNT` | `0.0000` | calculado | calculado | "PRECIO VENTA TOTAL ARTICULOS A TRANSFERIR". Calculado como `UnitSalesAmount × UnitCount` |
| `STOCK` | `0.0000` | fijo | fijo | Excel: "PARA LAS TRANSFERENCIAS ES CERO" |
| `DAILYAVERAGESALES` | `0.0000` | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO". Va en null/vacío |
| `SUGGESTEDPURCHASEORDER` | `0.0000` | fijo | fijo | Excel: "SIEMPRE SE CARGA EN CERO" |
| `PICKUPCODE` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `LASTUPDATEDATE` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `DIFBME_ASNTYPEID` | vacío | — | — | Excel: "NO SE USA EN ESTE DOCUMENTO" |
| `INVENTORYCONTROLDOCUMENTSTATE` (en detalle) | `2` | fijo | fijo | Excel: "ESTADO DEL DOCUMENTO / PARA AJUSTES ES VALOR 2". Para Transfer = `2` (alineado con cabecera). El XML actual no lo trae a nivel detalle, hay que agregarlo |

---

## ✏️ Cambios a aplicar al XML actual

1. **Renombrar `<SequenceNumber>` (en líneas de detalle) → `<DetSequenceNumber>`** (observación en el propio XML).
2. **`INVENTORYCONTROLDOCUMENTSTATE` cabecera = `2`** (no `4`) — según Excel para Transfer.
3. **Campos de cabecera que van en null** (no `0.0000` ni `0000001` ni `1`):
   - `RECEIPTNUMBER`
   - `PAGESQUANTITY`
   - `NETAMOUNT`, `EXEMPTAMOUNT`, `TAXAMOUNT`, `VATAMOUNT`, `SERVICESVATAMOUNT`, `DIFFERENCIALVATAMOUNT`, `IVATAXAMOUNT`, `IIBBTAXAMOUNT`, `TOTALAMOUNT`
4. **`OPERATORID` y `USER`** = `admin` (no `1`).
5. **`ITEMBRAND`** = `0` fijo (no nombre de marca como `Aramark`).

---

## 📌 Pendientes de validación (listado consolidado de `[UNKNOWN]`)

| # | Campo / Concepto | Duda |
|---|---|---|
| 1 | **Tabla detalle** | ¿Cuál es la tabla detalle de LAGERBEW? ¿`LAGERBEWPOS` o equivalente? |
| 2 | **Condición de registro** | ¿Bajo qué filtro/estado de LAGERBEW se emite el TLOG? |
| 3 | `RETAILSTOREID` (origen) | ¿Qué campo de LAGERBEW relaciona con la EESS origen? (¿`KST_ID`?) |
| 4 | `DESTINATIONRETAILSTOREID` | ¿Qué campo de LAGERBEW indica la EESS destino? (¿`KST_ID_DEST`?) |
| 5 | `ICDAMOUNT` | XML trae `0`, pero la lógica esperada sería SUM del detalle. Validar |
| 6 | `RECEIPTDATE` | ¿De qué campo de LAGERBEW sale la fecha de creación? (¿`NEW_ZEIT`?) |
| 7 | `RECEIPTTYPE` | Excel: "NULL — VALIDAR CON EMMA" |
| 8 | `ITEM` (línea) | ¿Qué campo del detalle de LAGERBEW se relaciona con ARTIKEL? |
| 9 | `UOMUNITS` (línea) | ¿Qué campo del detalle de LAGERBEW indica la unidad de medida (VPK)? |
| 10 | `UNITCOUNT` (línea) | ¿Qué campo del detalle de LAGERBEW indica la cantidad transferida? |

---

## 🔁 Pseudocódigo

```
═══════════════════════════════════════════════════════════════
  PROCESO: Generar XML InventoryTransfer
═══════════════════════════════════════════════════════════════

INICIO

  1. Seleccionar cabeceras de LAGERBEW pendientes de exportar
     (filtro de condición de registro: [UNKNOWN])

  2. PARA CADA cabecera:

     2.1. Obtener líneas asociadas (detalle de LAGERBEW)

     2.2. Resolver datos de cabecera mediante lookups:
          - Local origen  (KOSTST por KST_ID origen)
          - Local destino (KOSTST por KST_ID destino)
          - Generar SequenceNumber (longitud 12, inicia con 9)
          - BusinessDayDate = nombre del archivo (AAAAMMDD)
          - BeginDateTime   = BusinessDayDate + BEGIN_DATE_OFFSET
          - EndDateTime     = BusinessDayDate + END_DATE_OFFSET

     2.3. Construir cabecera del XML
          - DocumentTypeCode = "InventoryTransfer"
          - InventoryControlDocumentState = 2  (Excel)
          - SourceRetailStore = RetailStoreID
          - Operador / User = "admin"

     2.4. PARA CADA línea:
          - Resolver lookups (ARTIKEL por ART_ID, VPCKEINH por VPK_ID)
          - DestinationLocation = "DEP1_OS"   -- fijo
          - SourceLocation      = "DEP1_OS"   -- fijo
          - ItemBrand           = 0            -- fijo (consistencia con otros TLOG)
          - UnitBaseCostAmount  = ARTIKEL.ART_OEKP
          - UnitSalesAmount     = ARTIKEL.ART_VKP
          - Stock = 0
          - SuggestedPurchaseOrder = 0
          - InventoryControlDocumentState (línea) = 2
          - Construir línea del XML con DetSequenceNumber secuencial

     2.5. Calcular ICDAmount (suma de CostTotalAmount del detalle)

     2.6. Escribir XML a disco

     2.7. Marcar cabecera como exportada

FIN
```

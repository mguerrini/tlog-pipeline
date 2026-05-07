# Mapeo TLOG `INVENTORY_ADJUSTMENT` — OCPRA / HORECA

**Archivo origen:** `TLOG_INVENTORY_ADJUSTMENT_REAL__revisado_.xml`
**Hoja de referencia en Excel:** `InventoryAdjusment (2)` (formato 8 columnas según `InventoryReception`)
**Fecha del mapeo:** 2026-05-07

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

## 1. Tablas driver

| Sección TLOG | Tabla driver | Notas |
|---|---|---|
| Cabecera (`<Transaction>` + `<InventoryControlTransaction>`) | **`INVENTUR`** (resuelto desde Excel) | Excel hoja `InventoryAdjusment (2)` row 19 indica que el filtro es `Inv_status=8 AND inv_typ=4` sobre tabla `Inventur` |
| Detalle (`<inventoryControlDocumentMerchandiseLineItem>`) | **`INVPOSART`** (relación natural por `INV_ID`) | Coincide con la relación `Inventur.INV_ID = Invposart.INV_ID` |

> **🔧 CAMBIO IMPORTANTE:** En la versión anterior del mapeo se proponía `His_verbrauch` como tabla driver. El Excel (hoja `InventoryAdjusment (2)`) clarifica que la condición es sobre `Inventur` (`Inv_status=8 AND inv_typ=4`). En el dataset, `Inventur` tiene `INV_ID=6` con `INV_STATUS=8 AND INV_TYP=4`, lo que coincide. **Validar formalmente con OCPRA.**
>
> **Hipótesis alternativa:** podría existir un evento dual representado en ambas tablas (`Inventur` y `His_verbrauch`) con `VBR_ID = INV_ID = 6`. Esto sugiere que un mismo evento de ajuste se registra en ambas; la cabecera del TLOG OCPRA usa `Inventur` según el Excel.

### 1.1. Hipótesis sobre la relación cabecera ↔ detalle

En el dump actual se observa la siguiente coincidencia entre `Inventur` e `His_verbrauch`:

| Concepto | `Inventur` | `His_verbrauch` |
|---|---|---|
| ID | `INV_ID` = 6 | `VBR_ID` = 6 |
| Nombre | `INV_NAME` = `INV2605-0006` | `VBR_NAME` = `VBR2605-00006` |
| Local | `KST_ID` = 15 | `KST_ID` = 15 |
| Fecha | `INV_DATUM` = `2026-05-05` | `VBR_DATUM` = `2026-05-05` |
| Estado | `INV_STATUS=8`, `INV_TYP=4` | `VBR_STATUS = 2` |

Según el Excel, **la cabecera del TLOG usa `Inventur`** (filtrado por `INV_STATUS=8 AND INV_TYP=4`) y **el detalle usa `INVPOSART`** (con `INV_ID = Inventur.INV_ID`).

---

## 2. Condición de registro (filtro de cabecera)

```sql
SELECT *
FROM   Inventur
WHERE  INV_STATUS = 8        -- estado bookeado/cerrado (Excel)
  AND  INV_TYP    = 4        -- tipo correspondiente al ajuste (Excel)
  AND  AKTIV     <> 'N'      -- no eliminado lógicamente
  AND  EXP_NR     IS NOT NULL -- documento listo para exportar a TLOG (si aplica)
```

**Justificación:** Excel hoja `InventoryAdjusment (2)` row 19: `INVENTORYCONTROLDOCUMENTSTATE = 2` viene de `Inv_status=8 AND inv_typ=4` sobre tabla `Inventur`. El TLOG fija ICDState=2 fijo.

---

## 3. Mapeo de Cabecera — `TLOG_OCPRA_CABECERA`

> Convención de columnas (8 col, igual que `InventoryReception`):
> **A**=Campo TLOG · **B**=Valor ejemplo · **C**=Campo driver · **D**=Tabla driver · **E**=Condición/Lógica · **F**=Descripción · **G**=*(vacía)* · **H**=Observaciones

| Campo TLOG | Valor ejemplo | Campo driver | Tabla driver | Condición / Lógica | Descripción | Observaciones |
|---|---|---|---|---|---|---|
| `RETAILSTOREID` | `0000084` | `KST_CODE` | `Kostst` | `Kostst.KST_ID = Inventur.KST_ID` → `KST_CODE` | APIES EESS | Excel confirma 7 dígitos (`0000084`). Padding según regla del proyecto |
| `WORKSTATIONID` | `0` | fijo | fijo | Constante `0` | PDV (0 = BME) | |
| `SEQUENCENUMBER` | `900000000006` | construido | — | SU: numérico longitud **12** comenzando por `9` | Numérico de 12 dígitos | Idem `InventoryReception` |
| `BUSINESSDAYDATE` | `2026-05-05` | nombre del archivo | Construir | Fecha del nombre del archivo (`AAAAMMDD`) | Fecha contable (período) | DEBE SER OBTENIDO DEL WS DE PERIODOS Y TURNOS EN BRIDGE |
| `PERIOD` | `0` | fijo | fijo | Constante `0` | Período | Validar con Emma |
| `SUBPERIOD` | `0` | fijo | fijo | Constante `0` | Subperíodo | Validar con Emma |
| `PERIODCODE` | *(vacío)* | — | — | Vacío en origen | Período | DEBE SER OBTENIDO DEL WS DE PERIODOS Y TURNOS EN BRIDGE |
| `SUBPERIODCODE` | *(vacío)* | — | — | Vacío en origen | Turno | DEBE SER OBTENIDO DEL WS DE PERIODOS Y TURNOS EN BRIDGE |
| `BEGINDATETIME` | `2026-05-05 22:00:01` | `BusinessDayDate + BEGIN_DATE_OFFSET` | Construir | Valor de inicio de día según configuración | Inicio registro operación usuario | Regla común del proyecto. (Versión anterior usaba `NEW_ZEIT`, alineamos a la regla común) |
| `ENDDATETIME` | `2026-05-06 22:00:00` | `BusinessDayDate + END_DATE_OFFSET` | Construir | Valor de cierre de día según configuración | Fin registro operación usuario | Regla común del proyecto |
| `OPERATORID` | `admin` | fijo | fijo | Constante `'admin'` | Usuario que generó el registro | Regla del proyecto: OperatorID = admin (Excel confirma "admin"). El XML de muestra trae `1` o resolución por NEW_USER, pero prevalece la regla del proyecto |
| `ORIGINALTRANSACTION` | *(vacío)* | — | — | Vacío | — | No aplica para Adjustment |
| `SERIALFORMID` | `900000000006` | = `SEQUENCENUMBER` | — | Mismo valor que `SequenceNumber` | Número de secuencia registro | No lo ingresa el HO |
| `DOCUMENTTYPECODE` | `INVENTORYADJUSTMENT` | fijo | fijo | Constante `'INVENTORYADJUSTMENT'` | Tipo de documento | |
| `INVENTORYCONTROLDOCUMENTSTATE` | `2` | derivado | `Inventur` | Cuando `INV_STATUS=8 AND INV_TYP=4` → `2` (fijo para Adjustment) | Estado del documento (Adjustment = 2) | El TLOG indica fijo `2` |
| `CONTRACREFERENCENUMBER` | `Generado desde Web` | `INV_INFO` o fijo | `Inventur` | Si `INV_INFO` no es nulo → usarlo; si no → `'Generado desde Web'` | Descripción del ajuste | `[UNKNOWN] - {Generado desde Web} - {El Excel dice "DESCRIPCIÓN DEL AJUSTE" sin definir campo origen. ¿Literal o INV_INFO?}` |
| `CREATEDATETIMESTAMP` | `2026-05-05 22:00:01.000 ART` | `=BeginDateTime + ms + ART` | Construir | `FORMAT(BeginDateTime, 'yyyy-MM-dd HH:mm:ss.fff')` + `' ART'` | Mismo dato que BeginDateTime con ms y timezone | Validar con Emma el formato exacto |
| `DESTINATIONRETAILSTOREID` | `0000084` | = `RETAILSTOREID` | — | Mismo valor que `RetailStoreID` | Mismo valor que RetailStoreID | |
| `EXPECTEDDELIVERYDATE` | `2026-05-05 00:00:00.0 ART` | `=BeginDateTime + ms + ART` | Construir | Igual a fecha de inicio con ms | Igual a fecha de creación con ms | Validar con Emma |
| `ICDAMOUNT` | `-317730.0000` | calculado | `Invposart` (detalle) | `SUM(CostTotalAmount)` de las líneas de detalle | Costo total del ajuste/merma | Suma de `(UnitCount × UnitBaseCostAmount)` de todas las líneas. Puede ser negativo (mermas) |
| `LASTUPDATEDATE` | `2026-05-05 22:00:01.000 ART` | `=BeginDateTime + ms + ART` | Construir | Mismo dato que EndDateTime con ms | | Validar con Emma |
| `ORIGINATOR` | *(vacío)* | — | — | Vacío | — | No aplica |
| `SOURCERETAILSTORE` | `0000084` | = `RETAILSTOREID` | — | Mismo valor que `RetailStoreID` | APIES EESS | |
| `SUPPLIER` | *(vacío)* | — | — | NULL | NULL para ajustes/mermas | |
| `ORDERDOCUMENTTYPE` | *(vacío)* | — | — | NULL | No se usa | |
| `USER` | `admin` | fijo | fijo | Constante `'admin'` | Usuario que generó el registro | Regla del proyecto. Mismo valor que `OperatorID` |
| `ICDQUANTITY` | *(vacío)* | — | — | NULL | No se usa | |
| `ICDTOTSALESAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `FREQUENCY` | *(vacío)* | — | — | NULL | No se usa | |
| `INVENTORYADJUSTMENTTYPE` | `UNJUSTIFIED_ADJUSTMENTS` | `[UNKNOWN]` | `[UNKNOWN]` | `[UNKNOWN] - {UNJUSTIFIED_ADJUSTMENTS} - {Excel da las 5 opciones válidas: UNJUSTIFIED_ADJUSTMENTS, UNJUSTIFIED_DEPLETIONS, CORRECTIVE_ADJUSTMENT, JUSTIFIED_ADJUSTMENTS, JUSTIFIED_DEPLETIONS. Falta tabla de mapeo VRT_ID → InventoryAdjustmentType. Tabla Verbrauchsart no está en el dump}` | Tipo de ajuste (ajuste / merma) | |
| `RECEIPTNUMBER` | *(vacío)* | — | — | NULL | NULL para ajustes/mermas | El XML de muestra trae `INV2006-0003` pero el comentario aclara: "En ADJUSTMENT debe venir en Null". Confirmado: vacío |
| `FISCALRECEIPTFLAG` | `false` | fijo | fijo | Constante `false` | Siempre false en Adjustment | |
| `RECEIPTTYPE` | *(vacío)* | — | — | NULL | NULL | Validar con Emma |
| `RECEIPTDATE` | `2026-05-05 00:00:00.0 ART` | `INV_DATUM` | `Inventur` | `FORMAT(INV_DATUM, 'yyyy-MM-dd HH:mm:ss.f')` + `' ART'` | Fecha creación (carga manual) | |
| `CAINUMBER` | *(vacío)* | — | — | NULL | No se usa | |
| `CAIDATE` | *(vacío)* | — | — | NULL | No se usa | |
| `PAGESQUANTITY` | *(vacío)* | — | — | NULL | No se usa | |
| `NETAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `EXEMPTAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `TAXAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `VATAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `SERVICESVATAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `DIFFERENCIALVATAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `IVATAXAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `IIBBTAXAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `TOTALAMOUNT` | *(vacío)* | — | — | NULL | No se usa | |
| `ESTADO` *(BMC, no TLOG)* | `REPLICADO` | fijo | fijo | Constante `'REPLICADO'` al exportar | Estado en BMC | Idem `InventoryReception` |

---

## 4. Mapeo de Detalle — `TLOG_OCPRA_DETALLE`

> **Driver:** `INVPOSART` con `INV_ID = Inventur.INV_ID`
>
> **Condición de registro de detalle:**
> ```sql
> SELECT *
> FROM   Invposart
> WHERE  INV_ID = Inventur.INV_ID
>   AND  (INP_IST - INP_SOLL) <> 0   -- opcional: solo líneas con variance
> ```

| Campo TLOG | Valor ejemplo | Campo driver | Tabla driver | Condición / Lógica | Descripción | Observaciones |
|---|---|---|---|---|---|---|
| `RETAILSTOREID` | `0000084` | = cabecera | — | Heredado de cabecera | APIES EESS | |
| `WORKSTATIONID` | `0` | = cabecera | — | Heredado de cabecera | PDV (0 = BME) | |
| `SEQUENCENUMBER` | `900000000006` | = cabecera | — | Heredado de cabecera | Mismo SequenceNumber del header | |
| `DET_SEQUENCENUMBER` | `1` | construido | — | Numérico (3), `ROW_NUMBER() OVER (PARTITION BY INV_ID ORDER BY <orden_natural>)`, valores 1..999 | Enumeración de ítems dentro del documento | Máximo 999 líneas por documento |
| `ITEM` | `568` | `ART_NR` (vía `ART_ID`) | `Invposart` | Resolver SKU: `Artikel.ART_NR` con `Artikel.ART_ID = Invposart.ART_ID` | SKU / Código de artículo | El nombre del campo en TLOG es `ItemCode` pero debe escribirse como `Item` según comentario del XML revisado |
| `UOMUNITS` | `1.0000` | `VPK_ID` | `Invposart` | `Vpckeinh.VPK_ID = Invposart.VPK_ID` | Código de unidad de medida | Excel confirma: es `VPK_ID`, no factor |
| `ITEMBRAND` | `0` | fijo | fijo | Constante `0` | Marca | **Valor fijo `0`** (Excel hoja `InventoryAdjusment (2)` row 60: "No manejamos Marca"). El XML de muestra trae rubros pero el Excel manda fijo `0` |
| `ITEMDESCRIPTION` | `Coca cola lata` | `ART_NAME` | `Artikel` | `Artikel.ART_ID = Invposart.ART_ID` | Descripción del artículo | |
| `UNITBASECOSTAMOUNT` | `20.0000` | `INP_EKP` | `Invposart` | Costo unitario al momento del ajuste | Costo unitario del artículo | NO se carga manualmente, se obtiene del maestro de artículos |
| `UNITCOUNT` | `100.0000` (o `-80.0000`) | **Variance** = `INP_IST − INP_SOLL` | `Invposart` | Diferencia de inventario (cantidad ajustada). Positivo = ajuste a favor; negativo = merma | Cantidad a ajustar (variance) | Confirmado por usuario y Excel: representa el variance |
| `DESTINATIONLOCATION` | `DEP1_OS` | fijo | fijo | Constante `'DEP1_OS'` | Depósito OPESSA | Valor fijo (Excel) |
| `SOURCELOCATION` | `DEP1_OS` | fijo | fijo | Constante `'DEP1_OS'` | Depósito OPESSA | Valor fijo (Excel) |
| `COSTTOTALAMOUNT` | `2000.0000` | calculado | — | `UnitCount × UnitBaseCostAmount` | Costo total del artículo en el ajuste | Conserva el signo del variance |
| `UNITSALESAMOUNT` | `0.0000` o `10.0000` | `ART_VKP` | `Artikel` | Precio venta del artículo (Excel: "Sales Price en el Articulo") | Precio venta del artículo | El XML mayormente trae `0.0000` (artículos sin precio); cuando hay precio (ej `ItemCode 616` con `10.0000`) se valoriza |
| `SALESTOTALAMOUNT` | `0.0000` | calculado | — | `UnitCount × UnitSalesAmount` | Precio total de venta del ajuste | |
| `STOCK` | `0.0000` | `[UNKNOWN]` | `[UNKNOWN]` | `[UNKNOWN] - {0.0000} - {Excel row 69: "Stock EOD - VALIDAR SI ES PREVIO A LA CANTIDAD AJUSTADA" + "VALIDAR CON EMMA SI ESTE CAMPO SE USA PARA ESTE DOCUMENTO". El Excel también marca dudas, así que el UNKNOWN persiste}` | Stock previo al ajuste | |
| `DAILYAVERAGESALES` | `0.0000` | fijo | fijo | Constante `0` | No se usa en este documento | |
| `SUGGESTEDPURCHASEORDER` | `0.0000` | fijo | fijo | Constante `0` | Siempre 0 | |
| `PICKUPCODE` | *(vacío)* | `[UNKNOWN]` | `[UNKNOWN]` | `[UNKNOWN] - {vacío} - {Marca de stock del artículo a ajustar (S1 maneja stock, S2 no maneja stock). El Excel define qué representa pero NO especifica el campo origen en el maestro de artículos de PRISMA}` | Marca de stock del artículo | S1 = maneja stock, S2 = no maneja stock |
| `LASTUPDATEDATE` | *(vacío)* | — | — | NULL | No se usa en este documento | |
| `DIFBME_ASNTYPEID` | *(vacío)* | — | — | NULL | No se usa en este documento | |
| `INVENTORYCONTROLDOCUMENTSTATE` *(línea)* | `2` | fijo | fijo | Constante `2` | Estado del documento a nivel línea | Igual que el del header. Para Adjustment siempre `2` |

---

## 5. Campos del XML revisado que **NO** deben aparecer en el TLOG final

Marcados explícitamente en el XML de referencia:

| Campo en el XML actual | Acción | Motivo |
|---|---|---|
| `<TRX_SEQ_NBR>` (en cada línea) | **ELIMINAR** | Comentario: "No debe venir este campo en TLOG_INVENTORY" |
| `<SequenceNumber>` *(de detalle)* | **RENOMBRAR** a `<DetSequenceNumber>` | Comentario: "El campo debe llamarse DetSequenceNumber" |
| `<ItemCode>` | **RENOMBRAR** a `<Item>` | Comentario: "El campo debe llamarse Item" |
| `<ReceiptNumber>` | **DEJAR VACÍO** (abrir y cerrar tag sin contenido) | Comentario: "En ADJUSTMENT debe venir en Null" |

---

## 6. Pendientes de validación

| # | Punto | Quién valida |
|---|---|---|
| 1 | **Tabla driver definitiva** (`Inventur` según Excel vs `His_verbrauch` de hipótesis previa) | Equipo OCPRA |
| 2 | Mapeo `VRT_ID` → `InventoryAdjustmentType` (5 valores posibles). Tabla `Verbrauchsart` no en dump | Negocio / Emma |
| 3 | Origen real de `PICKUPCODE` (S1/S2) — campo en `Artikel` o regla | Emma / Integración PRISMA |
| 4 | Si `STOCK` (línea) lleva `INP_SOLL` previo o constante `0` | Emma |
| 5 | Formato exacto de timezone en `CREATEDATETIMESTAMP`, `EXPECTEDDELIVERYDATE`, `LASTUPDATEDATE`, `RECEIPTDATE` | Emma |
| 6 | Si `BUSINESSDAYDATE` se respeta del nombre del archivo o lo sobrescribe el WS de Períodos/Turnos del Bridge | Bridge / Emma |
| 7 | `CONTRACREFERENCENUMBER`: literal "Generado desde Web" o desde `INV_INFO` | Emma |

---

## 📌 Resumen consolidado de [UNKNOWN] (los que persisten)

| # | Campo | Razón |
|---|---|---|
| 1 | `<INVENTORYADJUSTMENTTYPE>` | Excel da las 5 opciones válidas pero falta tabla de mapeo `VRT_ID → InventoryAdjustmentType`. Tabla `Verbrauchsart` no en dump |
| 2 | `<STOCK>` (línea) | Excel también marca "VALIDAR CON EMMA SI ESTE CAMPO SE USA". Persiste UNKNOWN |
| 3 | `<PICKUPCODE>` (detalle) | El Excel define qué representa (S1/S2) pero no el campo origen en el maestro de artículos de PRISMA |
| 4 | `<CONTRACREFERENCENUMBER>` | El Excel dice "DESCRIPCIÓN DEL AJUSTE" sin definir campo origen. ¿Literal o `INV_INFO`? |
| 5 | **Tabla driver de cabecera** | Excel sugiere `Inventur` (`INV_STATUS=8 AND INV_TYP=4`). En el dump existe `His_verbrauch` con `VBR_ID = INV_ID`. Validar formalmente con OCPRA cuál es la tabla driver definitiva |

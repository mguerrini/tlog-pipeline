# Mapeo TLOG Cierre de Día — BusinessEOS (TYPECODE=63)

## Tabla driver: `DAILYTOTALS`

Cada `<Item>` del XML corresponde a **una fila de `DAILYTOTALS`** con PK `(KST_ID, ART_ID, DAY_DATE)`. Las demás tablas (`KOSTST`, `ARTIKEL`) actúan solo como **lookups** para resolver códigos de negocio.

**Cardinalidad:** 1 archivo TLOG por `(KST_ID, DAY_DATE)`.
En el dataset de muestra: 438 ítems para KST=15 en fecha 2026-05-05.

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

---

## 📐 Regla de SEQUENCENUMBER (fuente: hoja `SEQUENCENUMBER` del Excel)

| Origen | Longitud SEQUENCENUMBER | Longitud DET_SEQUENCENUMBER |
|---|---|---|
| Bridge | **11** dígitos (ej `12345678901`) | 4 |
| SU | 12 dígitos (ej `900000000000`) | 3 |
| **Cierre** | **Usa el SEQUENCE de Bridge → 11 dígitos** | — |

> **IMPORTANTE:** el TLOG de Cierre usa el SEQUENCE de **Bridge (11 dígitos)**, NO el de SU (12 dígitos como otros TLOG OCPRA).

---

## Filtros de generación del TLOG

Determinan qué filas de `DAILYTOTALS` entran al XML.

| # | Filtro | Campo / Tabla | Lógica | Comentario |
|---|---|---|---|---|
| 1 | Fecha | `DAY_DATE` / dailytotals | `WHERE DAY_DATE = <fecha del cierre>` | Define el archivo. Genera 1 TLOG por (KST_ID, DAY_DATE). |
| 2 | EESS | `KST_ID` / dailytotals | `WHERE KST_ID = <id EESS que cierra>` | Solo el centro de costo (EESS) que está cerrando. |
| 3 | KST válida | `AKTIV` / `KST_TYPLAGER` / kostst | `JOIN dailytotals.KST_ID = kostst.KST_ID WHERE AKTIV = ' ' AND KST_TYPLAGER IN (5)` | Solo CCs activos y de tipo Tienda/Depósito. |
| 4 | Artículo válido | `AKTIV` / `ART_LAGER` / `ART_NOTINV` / artikel | `JOIN dailytotals.ART_ID = artikel.ART_ID WHERE AKTIV <> 'N' AND ART_LAGER <> 0 AND ART_NOTINV = 0` | Excluye artículos inactivos o no inventariables. |
| 5 | (opcional) Movimiento | múltiples / dailytotals | `WHERE DAY_SOHBEG<>0 OR DAY_SOHEND<>0 OR (DAY_QTYPURCH+DAY_QTYTRSFIN+DAY_QTYTRSFOUT+DAY_QTYUSAGE+DAY_QTYSOLD+DAY_QTYINV)<>0` | Decisión de negocio: ¿reportar artículos con todo en cero? En el dataset hay 16 filas en esa condición. |

---

## TLOG_OCPRA_CABECERA (1 ocurrencia por archivo)

| Campo TLOG | Valor ejemplo | Campo origen | Tabla | Comentario / Lógica | Notas |
|---|---|---|---|---|---|
| `RETAILSTOREID` | `00019` | `KST_CODE` | kostst | JOIN con `dailytotals.KST_ID = kostst.KST_ID` | Formato 5 dígitos con ceros a izquierda (igual que demás TLOGs). |
| `WORKSTATIONID` | `0` | fijo | fijo | Constante = 0 | El cierre se genera en BME, no en POS. |
| `SEQUENCENUMBER` | `12345678901` | construir (Bridge) | construir | **Numérico longitud 11** (Bridge). Usa el SEQUENCE de Bridge, NO el de SU | Formato `APIES‖PosID‖NroSecuencia`. No puede duplicarse. |
| `BUSINESSDAYDATE` | `2026-05-05` | nombre del archivo | Construir | Fecha que viene en el nombre del archivo (`AAAAMMDD`). Coincide con `DAY_DATE` del cierre | Define el nombre del archivo. |
| `BEGINDATETIME` | `2026-05-05 22:00:01` | `BusinessDayDate + BEGIN_DATE_OFFSET` | Construir | `BUSINESSDAYDATE + BEGIN_DATE_OFFSET` (config). Convención típica: `22:00:01` (inicio de día convencional) | Nombre del archivo usa fecha D+1. |
| `ENDDATETIME` | `2026-05-06 22:00:00` | `BusinessDayDate + END_DATE_OFFSET` | Construir | `BUSINESSDAYDATE + END_DATE_OFFSET` (config). Convención típica: `22:00:00` D+1 (cierre del día) | |
| `OPERATORID` | `admin` | fijo | fijo | Constante `'admin'` (regla del proyecto) | El XML de muestra trae `1`, debe enviarse `admin`. |
| `PERIODO` | `0` | fijo | fijo | Constante = 0 | Validar con Emma. |
| `SUBPERIOD` | `0` | fijo | fijo | Constante = 0 | Validar con Emma. |
| `PERIODCODE` | `0` | vacío / 0 | WS Bridge | Obtener del WS de Periodos y Turnos en Bridge | En el XML real aparece como `<PERIODCODE/>0<PERIODCODE/>` — **tags mal formadas, corregir a `<PERIODCODE>0</PERIODCODE>`**. |
| `SUBPERIODCODE` | `0` | vacío / 0 | WS Bridge | Obtener del WS de Periodos y Turnos en Bridge | Idem PERIODCODE — tags mal formadas en el ejemplo. |
| `TYPECODE` | `BusinessEOS` | fijo | fijo | Constante = `"BusinessEOS"` | Identifica al TLOG como cierre de día (End Of Sale). |
| `TYPEID` | `63` | fijo | fijo | Constante = 63 | Equivalente numérico de BusinessEOS. |

---

## ItemList / Item — TABLA DRIVER: `DAILYTOTALS`

1 fila por `(KST_ID, ART_ID, DAY_DATE)`.

| Campo TLOG | Valor ejemplo | Campo origen | Tabla | Comentario / Lógica | Notas |
|---|---|---|---|---|---|
| `TRX_TYPE` | `63` | fijo | fijo | Constante = 63 | Replica `TYPEID` de cabecera. Comentario en XML: "no es necesario, remover". |
| `TRX_WS` | `0` | fijo | fijo | Constante = 0 | Replica `WORKSTATIONID` de cabecera. |
| `TRX_SEQ_NBR` | `12345678901` | construir (Bridge) | construir | Idem `SEQUENCENUMBER` de cabecera (11 dígitos) | Mismo valor que la cabecera. |
| `STOCK_SEQ_NUMBER` | `1` | fijo | fijo | Constante = 1 | `[UNKNOWN] - {1} - {En el XML siempre es 1. Validar si tiene otra lógica con negocio}` |
| `LOCATION_CODE` | `TND2` / `DEP1_OS` | `KST_CODE` | kostst | JOIN `dailytotals.KST_ID = kostst.KST_ID → kostst.KST_CODE` | `[UNKNOWN] - {DEP1_OS o TND2} - {En el XML el primer item usa DEP1_OS y el resto TND2. ¿Lógica por sub-ubicación (depósito vs tienda)? ¿Tipo de stock? Validar con negocio}` |
| `REVENUE_CENTER` | `RCD` | fijo | fijo | Constante = `"RCD"` | `[UNKNOWN] - {RCD} - {En el XML siempre es RCD. Validar si depende de algo (¿warengruppe?)}` |
| `ITEM_INVENTORY_STATE` | `OnSale` | fijo | fijo | Constante = `"OnSale"` | Valor fijo según comentario del XML. |
| `ITEM_SEQ_NUMBER` | `1, 2, 3, ...` | construir | construir | Contador secuencial 1..N por archivo | `ROW_NUMBER()` ordenado por ART_ID o por orden de lectura. Empieza en 1. |
| `ITEM_CODE` | `566` | `ART_NUMMER` | artikel | JOIN `dailytotals.ART_ID = artikel.ART_ID → artikel.ART_NUMMER` | **No es ART_ID** (PK interna), es el código de negocio `ART_NUMMER`. |
| `BEGIN_UNIT_COUNT` | `0.0000` | `DAY_SOHBEG` | dailytotals | Mapeo directo. **4 decimales obligatorio** | Stock al inicio del día. |
| `GROSS_SALE_UNIT_COUNT` | `0.0000` | `DAY_QTYSOLD` | dailytotals | Mapeo directo. 4 decimales | En EESS de combustibles las ventas vienen del POS, no de DAILYTOTALS — **VALIDAR**. |
| `RETURN_UNIT_COUNT` | `0.0000` | `[UNKNOWN]` | `[UNKNOWN]` | `[UNKNOWN] - {0.0000} - {No hay campo directo en DAILYTOTALS para devoluciones de venta. ¿Viene de TLOGs de venta? ¿Agregación de LIEFERPOS RTV? El Excel no tiene hoja de Cierre. Validar con negocio}` |
| `RECEIVED_UNIT_COUNT` | `112.0000` | `DAY_QTYPURCH` | dailytotals | Mapeo directo. 4 decimales | Suma de recepciones (LIEFERSCHEIN) consolidadas en dailytotals. |
| `RETURN_TO_VENTOR_UNIT_COUNT` | `0.0000` | `[UNKNOWN]` | `[UNKNOWN]` | `[UNKNOWN] - {0.0000} - {No hay campo directo en DAILYTOTALS. ¿SUM(LFP_QTYRTV) por (KST_ID,ART_ID,fecha)? Además, "VENTOR" parece typo de "VENDOR" — confirmar si es typo del estándar HO o del archivo de muestra. El Excel no tiene hoja de Cierre}` |
| `TRANSFERIN_UNIT_COUNT` | `0.0000` | `DAY_QTYTRSFIN` | dailytotals | Mapeo directo. 4 decimales | Transferencias entrantes del día. |
| `TRANSFEROUT_UNIT_COUNT` | `0.0000` | `DAY_QTYTRSFOUT` | dailytotals | Mapeo directo. 4 decimales | Transferencias salientes del día. |
| `ADJUSTMENTIN_UNIT_COUNT` | `0.0000` | `[UNKNOWN]` | dailytotals | `[UNKNOWN] - {0.0000} - {DAY_QTYUSAGE es signado. ¿Separar positivos/negativos? ¿Considerar también DAY_QTYINV y DAY_QTYEXPENSE? El Excel no tiene hoja de Cierre. Validar con negocio}` |
| `ADJUSTMENTOUT_UNIT_COUNT` | `0.0000` | `[UNKNOWN]` | dailytotals | `[UNKNOWN] - {0.0000} - {DAY_QTYUSAGE negativo o DAY_QTYEXPENSE. Mermas, consumos internos. El Excel no tiene hoja de Cierre. Validar con negocio}` |
| `CURRENT_UNIT_COUNT` | `0.0000` | `DAY_SOHEND` | dailytotals | Mapeo directo. 4 decimales | Stock final = SOHBEG + entradas − salidas + ajustes. |

---

## Observaciones importantes

1. **Tabla driver:** `DAILYTOTALS`. Cada `<Item>` del XML = 1 fila con PK `(KST_ID, ART_ID, DAY_DATE)`.
2. **Cardinalidad:** 1 archivo TLOG por `(KST_ID, DAY_DATE)`. En el dataset de muestra: 438 items para KST=15 en 2026-05-05.
3. **Tablas de lookup:** `KOSTST` (KST_ID → KST_CODE para LOCATION_CODE / RETAILSTOREID) y `ARTIKEL` (ART_ID → ART_NUMMER para ITEM_CODE).
4. **Formato numérico:** TODOS los campos de cantidad del Item deben tener **4 decimales** (ej: `0.0000`, `112.0000`).
5. **SEQUENCENUMBER es de Bridge (11 dígitos)**, no de SU (12 dígitos como otros TLOG OCPRA). Confirmado en hoja `SEQUENCENUMBER` del Excel.
6. **OperatorID = `admin`** (fijo, regla del proyecto). El XML de muestra trae `1`, hay que cambiarlo.
7. **Errores en el XML ejemplo (corregir):**
   - `<PERIODCODE/>0<PERIODCODE/>` y `<SUBPERIODCODE/>0<SUBPERIODCODE/>` tienen tags mal cerradas. Corregir a `<PERIODCODE>0</PERIODCODE>`.
   - El campo `RETURN_TO_VENTOR_UNIT_COUNT` tiene typo (debería ser VENDOR). Verificar si es typo del estándar HO o solo del archivo de muestra.
8. **Campos sin origen directo en DAILYTOTALS** (Excel no tiene hoja de Cierre, persisten como `[UNKNOWN]`):
   - `RETURN_UNIT_COUNT` (devoluciones de venta).
   - `RETURN_TO_VENTOR_UNIT_COUNT` (devoluciones a proveedor).
   - `ADJUSTMENTIN_UNIT_COUNT` / `ADJUSTMENTOUT_UNIT_COUNT`.
   - `LOCATION_CODE` (variabilidad DEP1_OS / TND2).
9. **`ADJUSTMENTIN/OUT`:** la separación entre positivo/negativo debe definirse con negocio, ya que `DAY_QTYUSAGE` en dailytotals es signado.

---

## 📌 Resumen consolidado de [UNKNOWN]

| # | Campo / Concepto | Razón |
|---|---|---|
| 1 | `STOCK_SEQ_NUMBER` | El XML siempre trae `1`. Validar si tiene otra lógica |
| 2 | `LOCATION_CODE` | Variabilidad entre `DEP1_OS` y `TND2`. ¿Lógica por sub-ubicación? |
| 3 | `REVENUE_CENTER` | Siempre `RCD` en el XML. ¿Depende de warengruppe? |
| 4 | `RETURN_UNIT_COUNT` | Sin campo directo en DAILYTOTALS. Origen incierto |
| 5 | `RETURN_TO_VENTOR_UNIT_COUNT` | Sin campo directo. Posible typo de "VENDOR" |
| 6 | `ADJUSTMENTIN_UNIT_COUNT` / `ADJUSTMENTOUT_UNIT_COUNT` | Lógica de separación positivo/negativo a definir |

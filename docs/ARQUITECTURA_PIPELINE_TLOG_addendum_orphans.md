# Addendum — Validación de huérfanos en `create_db`

> **Integración:** este texto se incorpora a `ARQUITECTURA_PIPELINE_TLOG.md` como una sub‑sección nueva del paso `create_db` (sección 10.1 / step `create_db`). No modifica ningún otro paso. Marca un paso como "informativo": **un huérfano nunca falla el día**.

---

## 1. Qué chequea

Al final del step `create_db` — **después** de que todos los CSVs ya fueron volcados a SQLite — se ejecuta una rutina de **validación de integridad referencial lógica**. Las FKs están desactivadas en SQLite durante la carga (`foreign_keys=OFF`), así que este paso es la única red de seguridad.

Un **huérfano** es una fila de una tabla hija cuya columna FK apunta a un valor que **no existe** en la tabla padre. La fila ya está cargada en la DB; lo que se reporta es la inconsistencia, no se borra nada.

---

## 2. Relaciones a validar

Lista cerrada, derivada de los mapeos TLOG y de la inspección de columnas en los CSVs:

| # | Tabla hija | Columna hija | Tabla padre | Columna padre | Origen del vínculo |
|---|---|---|---|---|---|
| 1 | `LIEFERSCHEIN` | `LF_ID` | `LIEFER` | `LF_ID` | header de Reception/Return/FiscalDoc |
| 2 | `LIEFERPOS` | `LFS_ID` | `LIEFERSCHEIN` | `LFS_ID` | detalle ↔ cabecera |
| 3 | `LIEFERPOS` | `KST_ID` | `KOSTST` | `KST_ID` | RetailStoreID |
| 4 | `LIEFERPOS` | `LF_ID` | `LIEFER` | `LF_ID` | proveedor del detalle |
| 5 | `LIEFERPOS` | `ART_NR` | `ARTIKEL` | `ART_ID` | artículo del detalle |
| 6 | `LIEFERPOS` | `VPK_ID1` | `VPKEINH` | `VPK_ID` | unidad de empaque |
| 7 | `LIEFERPOS` | `VPK_ID2` | `VPKEINH` | `VPK_ID` | unidad de empaque alternativa |
| 8 | `INVPOSART` | `INV_ID` | `INVENTUR` | `INV_ID` | detalle de Adjustment / Count ↔ cabecera |
| 9 | `INVPOSART` | `ART_ID` | `ARTIKEL` | `ART_ID` | artículo inventariado |
| 10 | `INVPOSART` | `VPK_ID` | `VPKEINH` | `VPK_ID` | unidad de empaque |
| 11 | `INVPOSART` | `WGR_NR` | `WARENGRUPPE` | `WGR_ID` | rubro |
| 12 | `INVENTUR` | `KST_ID` | `KOSTST` | `KST_ID` | RetailStoreID del inventario |
| 13 | `HIS_VERBRAUCH` | `KST_ID` | `KOSTST` | `KST_ID` | RetailStoreID de consumo |
| 14 | `DAILYTOTALS` | `KST_ID` | `KOSTST` | `KST_ID` | RetailStoreID del cierre |
| 15 | `DAILYTOTALS` | `ART_ID` | `ARTIKEL` | `ART_ID` | artículo del cierre |
| 16 | `ARTIKEL` | `WGR_ID` | `WARENGRUPPE` | `WGR_ID` | rubro del artículo |
| 17 | `ARTIKEL` | `VPK_NR` | `VPKEINH` | `VPK_ID` | unidad por defecto |
| 18 | `VPKEINH` | `ART_NR` | `ARTIKEL` | `ART_ID` | artículo de la unidad |

> **Reglas comunes a las 18 validaciones:**
> - Filas con `NULL` en la columna hija **no son huérfanas** — se ignoran.
> - El conteo se hace sobre **filas hijas** y también sobre **valores distintos** huérfanos (un mismo valor inválido referenciado N veces cuenta 1 vez como "valor distinto huérfano" y N veces como "filas huérfanas").
> - Las relaciones tienen alcance global (no se filtran por `KST_ID` del retail). Reportan inconsistencias de los CSVs como un todo del día.

---

## 3. Implementación SQL

Cada chequeo es un `SELECT` de la forma:

```sql
SELECT
    COUNT(*)                       AS rows_total,
    COUNT(DISTINCT c.<col_hija>)   AS distinct_total,
    SUM(CASE WHEN p.<col_padre> IS NULL THEN 1 ELSE 0 END)               AS rows_orphan,
    COUNT(DISTINCT CASE WHEN p.<col_padre> IS NULL THEN c.<col_hija> END) AS distinct_orphan
FROM <tabla_hija> c
LEFT JOIN <tabla_padre> p ON c.<col_hija> = p.<col_padre>
WHERE c.<col_hija> IS NOT NULL;
```

Y un segundo `SELECT` para sample (top 10 valores distintos huérfanos):

```sql
SELECT DISTINCT c.<col_hija> AS orphan_value
FROM <tabla_hija> c
LEFT JOIN <tabla_padre> p ON c.<col_hija> = p.<col_padre>
WHERE c.<col_hija> IS NOT NULL AND p.<col_padre> IS NULL
LIMIT 10;
```

Las 18 relaciones se ejecutan en serie. Es barato: todas las tablas son chicas (≤ pocos miles de filas) y SQLite construye índices implícitos por PK. No requiere índices adicionales.

---

## 4. Output: archivo `AAAAMMDD_orphans.md`

### 4.1. Ubicación

```
target_root/AAAAMMDD/AAAAMMDD_orphans.md
```

Mismo directorio que `AAAAMMDD_pipeline.log` y `AAAAMMDD_day_status.json`. Se copia a `finished_root` junto con el resto de artefactos al final de `local_clean`.

### 4.2. Naming

`AAAAMMDD_orphans.md` — sigue el patrón `AAAAMMDD_*` definido en la sección 7 de la arquitectura.

### 4.3. Estructura del archivo

````markdown
# Orphans Report — 2026-05-05

- **Status:** OK_WITH_ORPHANS
- **Generated at:** 2026-05-07T14:32:11Z
- **Step:** create_db
- **Total relations checked:** 18
- **Relations with orphans:** 1
- **Total orphan rows:** 1015
- **Total distinct orphan values:** 3

---

## Resumen por relación

| # | Hija → Padre | Filas hija | Distintos hija | Filas huérfanas | Distintos huérfanos | Estado |
|---|---|---:|---:|---:|---:|:---:|
| 1 | LIEFERSCHEIN.LF_ID → LIEFER.LF_ID | 12 | 1 | 0 | 0 | ✅ |
| 2 | LIEFERPOS.LFS_ID → LIEFERSCHEIN.LFS_ID | 150 | 12 | 0 | 0 | ✅ |
| 3 | LIEFERPOS.KST_ID → KOSTST.KST_ID | 150 | 1 | 0 | 0 | ✅ |
| 4 | LIEFERPOS.LF_ID → LIEFER.LF_ID | 150 | 1 | 0 | 0 | ✅ |
| 5 | LIEFERPOS.ART_NR → ARTIKEL.ART_ID | 150 | 22 | 0 | 0 | ✅ |
| 6 | LIEFERPOS.VPK_ID1 → VPKEINH.VPK_ID | 150 | 3 | 0 | 0 | ✅ |
| 7 | LIEFERPOS.VPK_ID2 → VPKEINH.VPK_ID | 150 | 1 | 0 | 0 | ✅ |
| 8 | INVPOSART.INV_ID → INVENTUR.INV_ID | 1022 | 4 | 1015 | 3 | ⚠️ |
| 9 | INVPOSART.ART_ID → ARTIKEL.ART_ID | 1022 | 436 | 0 | 0 | ✅ |
| ... | ... | ... | ... | ... | ... | ... |

---

## Detalle de huérfanos

### ⚠️ INVPOSART.INV_ID → INVENTUR.INV_ID

- **Filas huérfanas:** 1015 / 1022
- **Valores distintos huérfanos:** 3 / 4
- **Sample (hasta 10):**

| orphan_value |
|---:|
| 1 |
| 4 |
| 5 |

> Solo `INV_ID = 6` existe en `INVENTUR`. Los otros 3 valores (`1`, `4`, `5`) están referenciados por `INVPOSART` pero no tienen cabecera. Las filas correspondientes **no se eliminan** de la DB; quedan disponibles, pero los TLOG que dependan de la relación cabecera‑detalle (Adjustment, Count) los van a ignorar al hacer el JOIN.
````

### 4.4. Reglas de redacción del archivo

- Si **todas las relaciones están OK** → archivo igualmente se genera, con `Status: OK`, una sola tabla resumen y la sección "Detalle de huérfanos" vacía (texto: `Sin huérfanos detectados.`).
- Si **alguna falla técnica** durante el chequeo (ej: tabla no existe en DB) → la fila correspondiente queda con estado `❌ ERROR` y el detalle muestra el mensaje. El status global del archivo pasa a `OK_WITH_CHECK_ERRORS`. El día sigue OK igual.
- **Sample limit:** 10 valores distintos por relación. Si hay más, se agrega línea `... y N valores distintos más.`.

### 4.5. Estados posibles del archivo

| Status | Significado |
|---|---|
| `OK` | 18 relaciones validadas, 0 huérfanos |
| `OK_WITH_ORPHANS` | 18 validadas, ≥1 con huérfanos |
| `OK_WITH_CHECK_ERRORS` | Alguna validación no pudo correr (ej: tabla faltante) |

> Los tres son **status del archivo de log**, no del step. **El step `create_db` queda OK siempre** que el bulk insert haya terminado bien, independientemente de los huérfanos. Esto cumple el requerimiento "Sigue OK (solo warning, log informativo, día continúa)".

---

## 5. Integración con el resto del pipeline

### 5.1. `_day_status.json`

Se agregan dos campos informativos al bloque del step `create_db`:

```json
{
  "global_steps": {
    "create_db": {
      "status": "ok",
      "started_at": "...",
      "finished_at": "...",
      "meta": {
        "rows_loaded": 2845,
        "orphans_status": "OK_WITH_ORPHANS",
        "orphans_relations_with_issues": 1,
        "orphans_total_rows": 1015,
        "orphans_report_file": "20260505_orphans.md"
      }
    }
  }
}
```

`overall_status` del día **no se ve afectado** por los huérfanos.

### 5.2. Logging técnico (`_pipeline.log`)

Al final de `create_db` se emite una línea por relación a nivel `INFO` (si OK) o `WARN` (si tiene huérfanos):

```
INFO  create_db.orphan_check  relation=LIEFERPOS.LFS_ID->LIEFERSCHEIN.LFS_ID rows=150 orphans=0
WARN  create_db.orphan_check  relation=INVPOSART.INV_ID->INVENTUR.INV_ID rows=1022 orphans=1015 distinct_orphans=3
INFO  create_db.orphan_summary status=OK_WITH_ORPHANS relations_with_issues=1 file=20260505_orphans.md
```

### 5.3. CLI

No agrega flags. No es desactivable: forma parte de `create_db`. Si alguien quiere "saltearlo", puede hacerlo solo con `enabled=false` para todo `create_db` (lo cual ya rompe el día por otro lado).

---

## 6. Estructura de código (Go)

Sub‑paquete dentro de `create_db`:

```
internal/steps/create_db/
├── step.go                  ← step principal (ya existente)
├── bulk_load.go             ← ya existente
└── orphan_check.go          ← NUEVO
```

Interfaz interna del nuevo módulo:

```go
// internal/steps/create_db/orphan_check.go

type Relation struct {
    ChildTable  string
    ChildCol    string
    ParentTable string
    ParentCol   string
}

type RelationResult struct {
    Relation        Relation
    RowsTotal       int64
    DistinctTotal   int64
    RowsOrphan      int64
    DistinctOrphan  int64
    SampleOrphans   []string  // hasta 10
    CheckError      error     // nil si la query corrió bien
}

type OrphanReport struct {
    Day              time.Time
    GeneratedAt      time.Time
    Results          []RelationResult
    OverallStatus    string  // OK | OK_WITH_ORPHANS | OK_WITH_CHECK_ERRORS
}

// API pública del módulo
func RunOrphanCheck(ctx context.Context, db *sql.DB, day time.Time) (*OrphanReport, error)
func WriteOrphanReportMD(report *OrphanReport, outPath string) error
```

La lista de 18 `Relation` está hardcodeada en `orphan_check.go` (constante `var DefaultRelations = []Relation{...}`). Cualquier alta/baja de relación pasa por commit en este archivo.

---

## 7. Ejemplo real con el dataset 2026-05-05

Con los CSVs actuales del proyecto, el chequeo arroja:

- **17 de 18 relaciones OK.**
- **1 relación con huérfanos:** `INVPOSART.INV_ID → INVENTUR.INV_ID`.
  - `INVPOSART` referencia 4 `INV_ID` distintos: `{1, 4, 5, 6}`.
  - `INVENTUR` solo contiene `INV_ID = 6`.
  - Resultado: 1015 filas huérfanas, 3 valores distintos huérfanos.
- **Status del archivo:** `OK_WITH_ORPHANS`.
- **Status del día:** `ok` (no afectado).

> Esto confirma el comportamiento esperado: el dataset puede tener detalles "sueltos" sin cabecera y el pipeline lo registra explícitamente, sin abortar.

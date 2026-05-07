# tlog-pipeline

Pipeline en Go para generación de archivos TLOG OCPRA a partir de exports CSV del sistema HORECA/OCPRA.

## Tabla de contenidos

- [Descripción](#descripción)
- [Arquitectura](#arquitectura)
- [Requisitos](#requisitos)
- [Compilación](#compilación)
- [Configuración](#configuración)
- [Uso](#uso)
- [Estructura del proyecto](#estructura-del-proyecto)
- [Tipos de TLOG generados](#tipos-de-tlog-generados)
- [Validación de integridad (orphan check)](#validación-de-integridad-orphan-check)
- [Archivos de salida](#archivos-de-salida)
- [Campos \[UNKNOWN\]](#campos-unknown)
- [Documentación](#documentación)

---

## Descripción

Este pipeline procesa los CSV exportados diariamente por el sistema HORECA y genera los 8 tipos de TLOG en formato XML requeridos por OCPRA para cada EESS (Estación de Servicio / centro de costo).

**Flujo básico:**

```
CSVs del día (source_root/AAAAMMDD/)
    └─► read_files   → validar archivos presentes
    └─► create_db    → cargar en store in-memory + chequeo de huérfanos
    └─► create_xml   → generar 8 XMLs por retail
    └─► local_clean  → mover artefactos a finished_root
    └─► ftp_upload   → subir XMLs al FTP destino (opcional)
    └─► ftp_end      → mover archivos en FTP a carpeta final (opcional)
```

---

## Arquitectura

Ver [`docs/ARQUITECTURA_PIPELINE_TLOG.md`](docs/ARQUITECTURA_PIPELINE_TLOG.md) para la especificación completa.

**Decisión de implementación:** en lugar de SQLite (dependencia externa no disponible en el entorno), las tablas CSV se cargan en una **store in-memory** (maps Go). Con los volúmenes reales (~3000 filas totales por día), el rendimiento es equivalente y la implementación es más simple.

---

## Requisitos

- **Go 1.22** o superior
- Sin CGO requerido (compilación cross-platform nativa)
- Sin dependencias externas problemáticas (solo `github.com/jlaffaye/ftp` para el cliente FTP, que se usa únicamente si FTP está habilitado)

---

## Compilación

```bash
# Clonar y compilar
git clone <repo>
cd tlog-pipeline
go build -o pipeline ./cmd/pipeline/

# Cross-compile para Linux amd64 desde otro OS
GOOS=linux GOARCH=amd64 go build -o pipeline-linux-amd64 ./cmd/pipeline/
```

---

## Configuración

Copiar y editar `config.json` (está junto al binario en runtime):

```json
{
  "local_folders": {
    "source_root": "/data/horeca/input",
    "target_root": "/data/horeca/output",
    "finished_root": "/data/horeca/finished"
  },
  "process": {
    "mode": "ALL",
    "execution_mode": "PARALLEL",
    "begin_date_offset": "00:00:00",
    "end_date_offset": "23:59:59",
    "operator_id": "admin"
  },
  "create_db":  { "enabled": true, "separator": "," },
  "create_xml": { "enabled": true },
  "local_clean":{ "enabled": true, "delete_source": false },
  "ftp_download":{ "enabled": false },
  "ftp_upload": { "enabled": false },
  "ftp_end":    { "enabled": false }
}
```

### Parámetros clave

| Campo | Descripción | Ejemplo |
|---|---|---|
| `local_folders.source_root` | Raíz donde están las carpetas `AAAAMMDD/` con los CSVs | `/data/input` |
| `local_folders.target_root` | Donde se generan los XMLs y logs por día | `/data/output` |
| `local_folders.finished_root` | Donde se mueven los artefactos al finalizar | `/data/finished` |
| `process.begin_date_offset` | Offset HH:MM:SS para `BeginDateTime` en los TLOG | `00:00:00` |
| `process.end_date_offset` | Offset HH:MM:SS para `EndDateTime` en los TLOG | `23:59:59` |
| `process.operator_id` | `OperatorID` y `User` en todos los TLOG | `admin` |
| `process.keep_db_after_run` | Guarda snapshot JSON de la store en memory | `false` |

---

## Uso

```bash
# Procesar todos los días disponibles en source_root
./pipeline

# Procesar un día específico
./pipeline --day 20260505

# Procesar un día sin FTP
./pipeline --day 20260505 --ftp-disabled

# Ejecutar solo el step create_xml de un día (asume create_db ya corrió)
./pipeline --day 20260505 --step create_xml

# Usar un config alternativo
./pipeline --config /etc/tlog/config.json --day 20260505

# Overrides de carpetas por CLI
./pipeline --folder-source-root /tmp/input --folder-target-root /tmp/output
```

### Flags disponibles

| Flag | Descripción | Default |
|---|---|---|
| `--config <path>` | Path al config.json | `./config.json` |
| `--day <AAAAMMDD>` | Día específico a procesar | (todos) |
| `--ftp-disabled` | Desactiva los 3 steps FTP | `false` |
| `--delete-source` | Fuerza eliminación de la carpeta source tras procesar | `false` |
| `--step <name>` | Ejecuta solo ese step (requiere `--day`) | — |
| `--folder-source-root` | Override de `local_folders.source_root` | — |
| `--folder-target-root` | Override de `local_folders.target_root` | — |
| `--folder-finished` | Override de `local_folders.finished_root` | — |

---

## Estructura del proyecto

```
tlog-pipeline/
├── cmd/pipeline/main.go              # Entrypoint
├── internal/
│   ├── config/                       # Carga, defaults y flags CLI
│   ├── pipeline/                     # Coordinator, Runner, Step interface, Status
│   ├── steps/
│   │   ├── ftp_download/             # Step FTP descarga (stub habilitado)
│   │   ├── read_files/               # Validación de archivos CSV
│   │   ├── create_db/                # Carga CSV → store in-memory + orphan check
│   │   ├── create_xml/               # Generación de XMLs por retail
│   │   ├── local_clean/              # Mover artefactos a finished
│   │   ├── ftp_upload/               # Step FTP subida (stub habilitado)
│   │   └── ftp_end/                  # Step FTP mover final (stub habilitado)
│   ├── tlog/
│   │   ├── tlog.go                   # Interface Generator
│   │   ├── factory/factory.go        # Construye la lista de 8 generators
│   │   ├── common/                   # XMLBuilder, HeaderCtx, format helpers
│   │   ├── unknown/                  # Helper Emit() para [UNKNOWN]
│   │   ├── reception/                # InventoryReception
│   │   ├── return_/                  # InventoryReturn
│   │   ├── transfer/                 # InventoryTransfer (stub — LAGERBEW ausente)
│   │   ├── adjustment/               # InventoryAdjustment
│   │   ├── count/                    # InventoryCount
│   │   ├── fiscaldoc_fc/             # InventoryFiscalDoc FC
│   │   ├── fiscaldoc_nc/             # InventoryFiscalDoc NC
│   │   └── cierre/                   # BusinessEOS (Cierre de día)
│   ├── db/                           # Store in-memory, lookups, orphan check
│   ├── csvio/                        # CSV reader + file detection
│   ├── naming/                       # FileNamer, TLOGType, TLOGOrder
│   ├── logger/                       # slog fanout (consola + archivo JSON)
│   └── timeutil/                     # ParseDay, ApplyOffset, formatos
├── docs/
│   ├── ARQUITECTURA_PIPELINE_TLOG.md
│   ├── ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md
│   ├── Estructura_Tablas_Spec.md
│   ├── mapeos/                       # MAPEO_TLOG_*.md (uno por tipo)
│   └── xmls_referencia/              # XMLs reales de referencia
├── sample_data/
│   └── 20260505/                     # CSVs de ejemplo del 2026-05-05
├── config.json                       # Config por defecto (apunta a sample_data)
├── go.mod
└── go.sum
```

---

## Tipos de TLOG generados

| NNNN | Tipo | Tabla driver | Condición |
|---|---|---|---|
| `0001` | `InventoryReception` | `LIEFERSCHEIN` | `LFS_STATUS=42, LFS_RTS≠1, NETTO>0, AKTIV=J` |
| `0002` | `InventoryReturn` | `LIEFERSCHEIN` | `LFS_RTS=1, LFS_STATUS IN(37,42), BRUTTO<0, AKTIV=J` |
| `0003` | `InventoryTransfer` | `LAGERBEW` | **[UNKNOWN]** — tabla no disponible en export actual |
| `0004` | `InventoryAdjustment` | `INVENTUR` | `INV_STATUS=8, INV_TYP=4` |
| `0005` | `InventoryCount` | `INVENTUR` | `INV_STATUS=8, INV_TYP=4` |
| `0006` | `InventoryFiscalDoc FC` | `LIEFERSCHEIN` | `LFS_STATUS=42, LFS_RTS≠1, NETTO>0, AKTIV=J` |
| `0007` | `InventoryFiscalDoc NC` | `LIEFERSCHEIN` | `LFS_STATUS=42, LFS_RTS=1, NETTO<0, AKTIV=J` |
| `0008` | `BusinessEOS` (Cierre) | `DAILYTOTALS` | Todas las filas del día por KST_ID |

**Naming de archivos XML:** `[KST_CODE]-AAAAMMDD-NNNN.xml`

Ejemplo: `31252-20260505-0004.xml` = Adjustment de la EESS 31252 del día 2026-05-05.

---

## Validación de integridad (orphan check)

Al finalizar `create_db`, el pipeline ejecuta **18 chequeos de integridad referencial** sobre las tablas cargadas. Los resultados se guardan en:

```
output/AAAAMMDD/AAAAMMDD_orphans.md
```

El archivo reporta por cada relación FK lógica: total de filas, filas huérfanas, valores distintos huérfanos y un sample de hasta 10 IDs problemáticos.

**Comportamiento:** la presencia de huérfanos **no falla el día**. El pipeline continúa y el reporte es informativo. Los campos `orphans_*` quedan reflejados en el `_day_status.json`.

Ver especificación completa en [`docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md`](docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md).

**Resultado con el dataset de ejemplo (2026-05-05):**
- 17/18 relaciones OK
- 1 con huérfanos: `INVPOSART.INV_ID → INVENTUR.INV_ID` (970 filas, valores `{1, 4, 5}` sin cabecera en INVENTUR)

---

## Archivos de salida

Por cada día procesado en `target_root/AAAAMMDD/`:

| Archivo | Descripción |
|---|---|
| `[KST_CODE]-AAAAMMDD-NNNN.xml` | TLOG OCPRA generado (uno por tipo y retail con datos) |
| `AAAAMMDD_day_status.json` | Estado de cada step con metadata |
| `AAAAMMDD_orphans.md` | Reporte de integridad referencial (18 relaciones FK) |
| `AAAAMMDD_pipeline.log` | Log estructurado JSON del día |
| `AAAAMMDD_pipeline.db.json` | Snapshot de la store (solo si `keep_db_after_run=true`) |

---

## Campos [UNKNOWN]

Los campos cuyo mapeo no pudo resolverse completamente se emiten con la convención:

```xml
<CAMPO>[UNKNOWN] - {valor xml ejemplo} - {dudas/opciones}</CAMPO>
```

Esto permite que los archivos sean procesados identificando claramente qué falta completar. Los campos `[UNKNOWN]` activos son:

| TLOG | Campo | Razón |
|---|---|---|
| Cierre | `RETURN_UNIT_COUNT` | Sin origen directo en DAILYTOTALS |
| Cierre | `RETURN_TO_VENTOR_UNIT_COUNT` | Sin origen directo. Posible typo VENTOR/VENDOR |
| Cierre | `ADJUSTMENTIN_UNIT_COUNT` | Lógica positivo/negativo a definir con negocio |
| Cierre | `ADJUSTMENTOUT_UNIT_COUNT` | Lógica positivo/negativo a definir con negocio |
| Transfer | Todo | Tabla `LAGERBEW` no disponible en export actual |

Ver [`docs/mapeos/`](docs/mapeos/) para el detalle completo de cada tipo.

---

## Documentación

| Documento | Descripción |
|---|---|
| [`docs/ARQUITECTURA_PIPELINE_TLOG.md`](docs/ARQUITECTURA_PIPELINE_TLOG.md) | Especificación técnica del pipeline |
| [`docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md`](docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md) | Spec del orphan check (requerimiento adicional) |
| [`docs/Estructura_Tablas_Spec.md`](docs/Estructura_Tablas_Spec.md) | Estructura y relaciones de las tablas CSV |
| [`docs/mapeos/MAPEO_TLOG_*.md`](docs/mapeos/) | Mapeo campo a campo de cada tipo de TLOG |
| [`docs/xmls_referencia/`](docs/xmls_referencia/) | XMLs reales de referencia para validación visual |

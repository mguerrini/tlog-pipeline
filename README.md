# tlog-pipeline

Pipeline en Go para generación de archivos TLOG OCPRA a partir de exports CSV del sistema HORECA/OCPRA.

## Tabla de contenidos

- [Descripción](#descripción)
- [Arquitectura](#arquitectura)
- [Requisitos](#requisitos)
- [Compilación y versionado](#compilación-y-versionado)
- [Configuración](#configuración)
- [Estructura de carpetas en runtime (appdata)](#estructura-de-carpetas-en-runtime-appdata)
- [Archivos de entrada CSV](#archivos-de-entrada-csv)
- [Uso](#uso)
- [Código — componentes clave](#código--componentes-clave)
- [Estructura del proyecto](#estructura-del-proyecto)
- [Tipos de TLOG generados](#tipos-de-tlog-generados)
- [Validación de integridad (orphan check)](#validación-de-integridad-orphan-check)
- [Archivos de salida](#archivos-de-salida)
- [Campos \[UNKNOWN\]](#campos-unknown)
- [Documentación](#documentación)

---

## Descripción

Este pipeline procesa los CSV exportados diariamente por el sistema HORECA y genera los tipos de TLOG en formato XML requeridos por OCPRA para cada EESS (Estación de Servicio / centro de costo).

**Flujo básico:**

```
CSVs del día (appdata/input/AAAAMMDD/)
    └─► split_by_date → repartir archivos planos por fecha (si aplica)
    └─► read_files    → validar archivos presentes
    └─► create_db     → cargar en store in-memory + chequeo de huérfanos
    └─► create_xml    → generar XMLs por retail
    └─► ftp_upload    → subir XMLs al FTP destino (opcional)
    └─► ftp_end       → mover archivos en FTP a carpeta final (opcional)
    └─► local_clean   → mover artefactos a appdata/finished (opcional)
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

## Compilación y versionado

### Dónde cambiar la versión

La versión está definida en **tres lugares** que deben mantenerse sincronizados:

| Archivo | Campo | Ejemplo |
|---|---|---|
| `cmd/pipeline/main.go` | `var Version = "7.0.0"` | valor por defecto si no se inyecta en build |
| `cmd/pipeline/versioninfo.json` | `FileVersion`, `ProductVersion`, `StringFileInfo.ProductVersion` | metadata del .exe en Windows |
| Script de build (`build.ps1` o `build-linux.ps1`) | parámetro `-Version` default | lo que se inyecta vía `-ldflags` |

El valor que queda embebido en el binario en producción es siempre el que pasa el script de build mediante `-ldflags "-X main.Version=<ver>"`. El valor en `main.go` es solo un fallback para desarrollo.

### Compilar en Windows (PowerShell)

```powershell
# Binario Windows con metadata de versión (.exe)
.\build.ps1 -Version 7.0.0 -Output tlog-gen.exe

# Cross-compile para Linux desde Windows
.\build-linux.ps1 -Version 7.0.0 -Output tlog-gen
```

`build.ps1` requiere `goversioninfo` (lo instala automáticamente si no está) para generar el `resource.syso` que incrusta la metadata del `.exe` (visible en Propiedades → Detalles en Windows Explorer).

### Compilar en Linux (Bash)

```bash
# Binario Linux
./build-linux.sh 7.0.0 tlog-gen

# Cross-compile para Linux desde cualquier OS con go build directo
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.Version=7.0.0" -o tlog-gen ./cmd/pipeline
```

---

## Configuración

El archivo `config.json` debe estar junto al binario (o indicarse con `--config`). A continuación se explica cada sección:

### `process`

```json
"process": {
  "file_name_include_document_type": false,
  "mode": "ALL",
  "execution_mode": "SERIAL",
  "parallel_retails_per_day": false,
  "begin_date_offset": "22:00:01",
  "end_date_offset": "22:00:00",
  "operator_id": "RY99999"
}
```

| Campo | Descripción | Valores |
|---|---|---|
| `file_name_include_document_type` | Si `true`, el nombre del XML de inventario incluye el tipo: `TLOG_INVENTORY_<Tipo>_<kst>_<seq>.xml`. Si `false`: `TLOG_INVENTORY_<kst>_<seq>.xml` | `true` / `false` |
| `mode` | Procesar todos los días disponibles en `source_root` o solo el indicado por `--day` | `ALL` / `DAY` |
| `execution_mode` | Procesar días en paralelo o en serie | `PARALLEL` / `SERIAL` |
| `parallel_retails_per_day` | Generar los XMLs de cada retail del día en paralelo | `true` / `false` |
| `begin_date_offset` | Tiempo `HH:MM:SS` usado como `BeginDateTime` en los TLOG. Con `22:00:01` corresponde al inicio del turno de la noche anterior | `HH:MM:SS` |
| `end_date_offset` | Tiempo `HH:MM:SS` usado como `EndDateTime` en los TLOG | `HH:MM:SS` |
| `operator_id` | Valor del campo `OperatorID` / `User` en todos los TLOG generados | cadena |

### `local_folders`

```json
"local_folders": {
  "all":           "./appdata/all",
  "source_root":   "./appdata/input",
  "target_root":   "./appdata/output",
  "finished_root": "./appdata/finished"
}
```

| Campo | Descripción |
|---|---|
| `all` | Carpeta donde `ftp_download` deposita los archivos cuando `ftp_download.split_by_date=true` (todos los CSV mezclados, sin subcarpeta por fecha). `split_by_date` los reparte después |
| `source_root` | Raíz que contiene subcarpetas `AAAAMMDD/` con los CSV del día a procesar |
| `target_root` | Donde se generan los XML, logs y artefactos del día (`target_root/AAAAMMDD/`) |
| `finished_root` | Destino final tras `local_clean`: los artefactos se mueven aquí cuando el paso está habilitado |

### `ftp_source` y `ftp_target`

Configuración de conexión a los servidores FTP/SFTP de origen y destino respectivamente.

```json
"ftp_source": { "url": "...", "port": 22, "username": "...", "pass": "..." },
"ftp_target": { "url": "...", "port": 22, "username": "...", "pass": "..." }
```

### `ftp_folders`

Rutas remotas en el servidor FTP.

```json
"ftp_folders": {
  "source_root":   "/SFTPYPF/YPF_Breakpoint_TLOG_Test/Input",
  "finished_root": "/SFTPYPF/YPF_Breakpoint_TLOG_Test/Finished",
  "target_root":   "/SFTPYPF/YPF_Breakpoint_TLOG_Test/Uploaded"
}
```

| Campo | Usado por |
|---|---|
| `source_root` | `ftp_download`: descarga los CSV desde esta ruta remota |
| `target_root` | `ftp_upload`: sube los XML generados a esta ruta remota |
| `finished_root` | `ftp_end`: mueve los archivos remotos aquí una vez procesados |

### `output`

Controla qué tipos de TLOG se generan. Si se omite la sección entera, todos quedan habilitados. Un campo ausente dentro de la sección también default-ea a `true`; solo un `false` explícito deshabilita el tipo.

```json
"output": {
  "cierre":                      true,
  "inventory_reception":         true,
  "inventory_fiscaldoc_fc":      true,
  "inventory_fiscaldoc_nc":      true,
  "inventory_return":            true,
  "inventory_adjustment_verbrauch": true,
  "inventory_adjustment_inventur":  true,
  "inventory_count_verbrauch":   true,
  "inventory_count_inventur":    true,
  "inventory_transfer":          true
}
```

### `logs`

Controla la escritura de archivos de reporte por día. Misma semántica que `output`: sección ausente → todo habilitado; campo ausente → `true`.

```json
"logs": {
  "pipeline_enabled":   false,
  "day_status_enabled": false,
  "sql_db_load":        false
}
```

| Campo | Archivo generado |
|---|---|
| `pipeline_enabled` | `AAAAMMDD_pipeline.log` — log estructurado JSON del día |
| `day_status_enabled` | `AAAAMMDD_day_status.json` — estado de cada step con tiempos |
| `sql_db_load` | `AAAAMMDD_sqldb_load.md` — reporte de carga de tablas CSV |

### Steps individuales

Cada step tiene su propia sección con al menos `"enabled": true/false`.

```json
"ftp_download": {
  "enabled": false,
  "split_by_date": true
}
```

| Step | Campo adicional | Descripción |
|---|---|---|
| `ftp_download` | `split_by_date` | Si `true`, los archivos remotos están todos en `ftp_folders.source_root` sin subcarpetas y se bajan a `local_folders.all`; el step `split_by_date` los reparte después. Si `false`, el servidor ya tiene subcarpetas `AAAAMMDD/` y se bajan directamente a `local_folders.source_root/AAAAMMDD/` |
| `split_by_date` | — | Reparte archivos de `local_folders.all` en subcarpetas `AAAAMMDD/` dentro de `local_folders.source_root` según la fecha del nombre de archivo |
| `read_files` | `expected_files`, `clear_cols` | Lista de patrones glob que deben existir en cada día. `clear_cols` permite vaciar columnas específicas antes de importar (ej. `INV_SELECT` de `INVENTUR`) |
| `create_db` | `separator` | Separador del CSV (usar `"\|"` para pipe) |
| `create_xml` | — | Solo `enabled` |
| `ftp_upload` | — | Solo `enabled` |
| `ftp_end` | `delete_local_source` | Si `true`, después de mover los archivos en el FTP remoto, borra también la carpeta local `target_root/AAAAMMDD/` |
| `local_clean` | `delete_source`, `delete_database` | `delete_source`: borra la carpeta `source_root/AAAAMMDD/` con los CSV originales. `delete_database`: borra el archivo `*_pipeline.db` del output |

---

## Estructura de carpetas en runtime (appdata)

El pipeline trabaja sobre cuatro carpetas locales configuradas en `local_folders`. El esquema completo de ciclo de vida de los archivos es:

```
appdata/
├── all/                          ← (solo si ftp_download.split_by_date=true)
│   ├── Kostst_20260505.csv       ← archivos FTP descargados, sin separar por fecha
│   ├── Liefer_20260505.csv
│   └── ...                       ← split_by_date los mueve a input/AAAAMMDD/
│
├── input/
│   └── 20260505/                 ← una carpeta por día a procesar
│       ├── Kostst_20260505.csv
│       ├── Liefer_20260505.csv
│       ├── Lieferschein-1_20260505.csv
│       └── ...                   ← leídos por read_files y create_db
│
├── output/
│   └── 20260505/                 ← creado por el pipeline al procesar el día
│       ├── TLOG_INVENTORY_31252_9260505000000.xml
│       ├── TLOG_Cierre_31252_9260505700000.xml
│       ├── 20260505_day_status.json
│       ├── 20260505_pipeline.log
│       ├── 20260505_orphans.md
│       └── 20260505_pipeline.db  ← eliminado si local_clean.delete_database=true
│
└── finished/
    └── 20260505/                 ← adonde local_clean mueve todo lo de output/
        ├── TLOG_INVENTORY_31252_9260505000000.xml
        ├── TLOG_Cierre_31252_9260505700000.xml
        └── ...
```

### Ciclo de vida por carpeta

**`input/AAAAMMDD/`**
- El pipeline lee los CSV desde aquí.
- Si `local_clean.delete_source=true`, la carpeta entera se **borra** al finalizar el día.
- Si `delete_source=false`, la carpeta queda intacta (útil para reprocesar).

**`output/AAAAMMDD/`**
- Se crea automáticamente al comenzar a procesar el día.
- Al finalizar, `local_clean` mueve todo su contenido a `finished/AAAAMMDD/`.
- Si `ftp_end.delete_local_source=true`, la carpeta se borra en lugar de moverse (cuando el upload FTP está habilitado).

**`finished/AAAAMMDD/`**
- Destino final de los artefactos generados.
- El pipeline no toca esta carpeta después de mover los archivos; su gestión queda a cargo del operador.

---

## Archivos de entrada CSV

### Nombre de archivo

El sistema HORECA exporta un CSV por tabla con el formato:

```
<Prefijo>_<AAAAMMDD>.csv
```

Ejemplos: `Kostst_20260505.csv`, `Lieferschein-1_20260505.csv`, `His_verbrauch_20260505.csv`

La fecha en el nombre identifica el día al que pertenecen los datos. El pipeline la usa para clasificarlos en la subcarpeta correcta (`split_by_date`) y para validar que todos los archivos esperados estén presentes (`read_files`).

### Formato del contenido

- **Separador:** pipe `|` (configurado en `create_db.separator`)
- **Primera fila:** header con nombres de columna en mayúsculas
- **Comillas opcionales:** los valores y nombres de columna pueden estar rodeados de comillas dobles (`"COLUMNA"` o `valor`). El reader las elimina automáticamente.
- **BOM:** si el archivo tiene BOM UTF-8 (`\xEF\xBB\xBF`), se descarta.
- **Campos faltantes:** si una fila tiene menos columnas que el header, los campos faltantes quedan como cadena vacía.
- **Campos vacíos:** se almacenan como `""` en memoria; los generadores los interpretan como NULL.

Ejemplo de encabezado real:
```
KST_ID|KST_NAME|KST_INDEX|KST_CODE|KST_PARENT|...
```

### Tablas esperadas

La lista de archivos requeridos se define en `read_files.expected_files` usando patrones glob. La tabla de mapeo prefijo → tabla interna es:

| Archivo CSV | Tabla interna | Opcional |
|---|---|---|
| `Kostst_*.csv` | `KOSTST` | no |
| `Liefer_*.csv` | `LIEFER` | no |
| `Warengruppe_*.csv` | `WARENGRUPPE` | no |
| `Vpckeinh_*.csv` | `VPCKEINH` | no |
| `Artikel_*.csv` | `ARTIKEL` | no |
| `Lieferschein-1_*.csv` | `LIEFERSCHEIN_VIEW` | sí |
| `Lieferpos_*.csv` | `LIEFERPOS` | no |
| `Inventur_*.csv` | `INVENTUR` | no |
| `Invposart_*.csv` | `INVPOSART` | no |
| `His_verbrauch_*.csv` | `HIS_VERBRAUCH` | no |
| `His_Verbrauchpos_*.csv` | `HIS_VERBRAUCHPOS` | no |
| `Dailytotals_*.csv` | `DAILYTOTALS` | no |
| `His_lagerbew_*.csv` | `HIS_LAGERBEW` | no |
| `His_lagbewpos_*.csv` | `HIS_LAGBEWPOS` | no |
| `Art_ItemCode_*.csv` | `ART_ITEM_CODE` | no |

Si falta un archivo no opcional, el step `read_files` falla el día y el pipeline se detiene.

---

## Uso

```bash
# Procesar todos los días disponibles en source_root
./tlog-gen

# Procesar un día específico
./tlog-gen --day 20260505

# Procesar un día sin FTP
./tlog-gen --day 20260505 --ftp-disabled

# Ejecutar solo el step create_xml de un día (asume create_db ya corrió)
./tlog-gen --day 20260505 --step create_xml

# Usar un config alternativo
./tlog-gen --config /etc/tlog/config.json --day 20260505

# Overrides de carpetas por CLI
./tlog-gen --folder-source-root /tmp/input --folder-target-root /tmp/output

# Mostrar versión
./tlog-gen --version
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

## Código — componentes clave

### Generadores de XML (`internal/tlogsql/`)

Cada tipo de TLOG tiene su propio archivo con la lógica de consulta sobre la store in-memory y construcción del XML:

| Archivo | Tipo generado |
|---|---|
| `reception.go` | `InventoryReception` |
| `return.go` | `InventoryReturn` |
| `transfer.go` | `InventoryTransfer` |
| `adjustment_verbrauch.go` | `InventoryAdjustment` (fuente: `HIS_VERBRAUCH`) |
| `adjustment_inventur.go` | `InventoryAdjustment` (fuente: `INVENTUR`) |
| `count_verbrauch.go` | `InventoryCount` (fuente: `HIS_VERBRAUCH`) |
| `count_inventur.go` | `InventoryCount` (fuente: `INVENTUR`) |
| `fiscaldoc_fc.go` | `InventoryFiscalDoc FC` |
| `fiscaldoc_nc.go` | `InventoryFiscalDoc NC` |
| `cierre.go` | `BusinessEOS` (Cierre de día) |

Todos implementan la interfaz definida en `internal/tlog/tlog.go`. El orquestador (`internal/steps/create_xml_sql/step.go`) itera los generadores habilitados según `config.output` y los ejecuta en el orden canónico de `naming.TLOGOrder`.

### Naming de archivos (`internal/naming/`)

La convención de nombres de todos los archivos de salida está centralizada en `DefaultNamer` (`namer.go`):

```
XML de inventario:
  IncludeDocumentType=false → TLOG_INVENTORY_<KstCode>_<SequenceNumber>.xml
  IncludeDocumentType=true  → TLOG_INVENTORY_<Tipo>_<KstCode>_<SequenceNumber>.xml

XML de cierre (siempre):
  TLOG_Cierre_<KstCode>_<SequenceNumber>.xml

Otros artefactos del día:
  AAAAMMDD_day_status.json
  AAAAMMDD_pipeline_status.json
  AAAAMMDD_pipeline.db
  AAAAMMDD_pipeline.log
  AAAAMMDD_orphans.md
```

### SequenceNumber (`internal/sequence/sequence.go`)

El campo `SEQUENCENUMBER` de los TLOG OCPRA sigue el formato de 13 dígitos:

```
9  AAMMDD  DOC  CONTADOR(4)
│  ──┬───  ─┬─  ────┬────
│    │       │       └─ secuencial por (día × tipo), empieza en 0000
│    │       └─ índice del tipo de documento (0..7)
│    └─ fecha de negocio en formato AAMMDD
└─ prefijo fijo '9'
```

Ejemplo: `9260505000000` = tipo `Reception` (doc=0), primer documento del día 2026-05-05, contador 0.

Los índices de tipo de documento son:

| DocumentNumber | Tipo |
|---|---|
| 0 | Reception |
| 1 | Return |
| 2 | Transfer |
| 3 | Adjustment (Verbrauch e Inventur comparten índice) |
| 4 | Count (Verbrauch e Inventur comparten índice) |
| 5 | FiscalDocFC |
| 6 | FiscalDocNC |
| 7 | Cierre |

Ver [`docs/SEQUENCENUMBER.md`](docs/SEQUENCENUMBER.md) para la especificación completa.

### Store in-memory / base de datos (`internal/sqldb/` y `internal/db/`)

El step `create_db` carga todos los CSV del día en una **store in-memory** (implementada con maps Go) en lugar de una base de datos externa. El módulo `internal/sqldb/` define:

- **`schema.go`**: esquema tipado de cada tabla (columnas, tipos `TEXT`/`INTEGER`/`REAL`, PKs, FKs, índices). Determina cómo se convierten los valores del CSV al insertarlos.
- **`loader.go`**: carga cada CSV en el orden de `LoadOrder` (respeta dependencias FK lógicas), aplica `clear_cols` si está configurado, y ejecuta el orphan check al finalizar.
- **`report.go`**: genera el archivo `AAAAMMDD_orphans.md` con el resultado de los 18 chequeos de integridad referencial.

El módulo `internal/db/helpers.go` expone funciones de lookup (`AsInt`, `AsFloat`, `Lookup`, etc.) que usan los generadores de TLOG para acceder a los datos en memoria sin repetir lógica de conversión de tipos.

### Lectura de CSV (`internal/csvio/`)

- **`reader.go`**: parsea un CSV respetando el separador configurado, elimina BOM, normaliza el header a mayúsculas, elimina comillas opcionales en nombres de columna y valores.
- **`files.go`**: resuelve qué CSV existen en una carpeta de día mediante `TableMapping` (prefijo → tabla) y determina el orden de carga.

---

## Estructura del proyecto

```
tlog-pipeline/
├── cmd/pipeline/
│   ├── main.go                   # Entrypoint; define var Version = "X.Y.Z"
│   └── versioninfo.json          # Metadata de versión para el .exe de Windows
├── internal/
│   ├── config/                   # Carga, defaults y flags CLI
│   ├── pipeline/                 # Coordinator, Runner, Step interface, Status
│   ├── steps/
│   │   ├── split_by_date/        # Repartir archivos planos por fecha
│   │   ├── read_files/           # Validación de archivos CSV esperados
│   │   ├── create_sql_db/        # Carga CSV → store in-memory + orphan check
│   │   ├── create_xml_sql/       # Generación de XMLs por retail
│   │   ├── ftp_download/         # Step FTP descarga
│   │   ├── ftp_upload/           # Step FTP subida
│   │   ├── ftp_end/              # Step FTP mover a carpeta final
│   │   └── local_clean/          # Mover artefactos a finished
│   ├── tlog/
│   │   ├── tlog.go               # Interface Generator
│   │   └── common/               # XMLBuilder, HeaderCtx, format helpers
│   ├── tlogsql/                  # Generadores de TLOG (uno por tipo)
│   ├── sqldb/                    # Schema tipado, loader, orphan check, reporte
│   ├── db/                       # Helpers de acceso a la store in-memory
│   ├── csvio/                    # CSV reader + file detection
│   ├── naming/                   # FileNamer, TLOGType, TLOGOrder
│   ├── sequence/                 # Construcción del SEQUENCENUMBER
│   ├── logger/                   # slog fanout (consola + archivo JSON)
│   └── timeutil/                 # ParseDay, ApplyOffset, formatos de fecha
├── docs/
│   ├── ARQUITECTURA_PIPELINE_TLOG.md
│   ├── ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md
│   ├── Estructura_Tablas_Spec.md
│   ├── SEQUENCENUMBER.md
│   └── mapeos/                   # MAPEO_TLOG_*.md (uno por tipo)
├── sample_data/
│   └── 20260505/                 # CSVs de ejemplo
├── build.ps1                     # Build Windows (.exe con metadata)
├── build-linux.ps1               # Cross-compile Linux desde Windows (PowerShell)
├── build-linux.sh                # Build Linux (Bash)
├── config.json                   # Config por defecto (apunta a ./appdata)
├── go.mod
└── go.sum
```

---

## Tipos de TLOG generados

| Tipo | Tabla driver | Condición |
|---|---|---|
| `InventoryReception` | `LIEFERSCHEIN` | `LFS_STATUS=42, LFS_RTS≠1, NETTO>0, AKTIV=J` |
| `InventoryReturn` | `LIEFERSCHEIN` | `LFS_RTS=1, LFS_STATUS IN(37,42), BRUTTO<0, AKTIV=J` |
| `InventoryTransfer` | `HIS_LAGERBEW` | **[UNKNOWN]** — tabla no disponible en export actual |
| `InventoryAdjustment` (Verbrauch) | `HIS_VERBRAUCH` | ajustes desde consumo histórico |
| `InventoryAdjustment` (Inventur) | `INVENTUR` | `INV_STATUS=8, INV_TYP=4` |
| `InventoryCount` (Verbrauch) | `HIS_VERBRAUCH` | conteos desde consumo histórico |
| `InventoryCount` (Inventur) | `INVENTUR` | `INV_STATUS=8, INV_TYP=4` |
| `InventoryFiscalDoc FC` | `LIEFERSCHEIN` | `LFS_STATUS=42, LFS_RTS≠1, NETTO>0, AKTIV=J` |
| `InventoryFiscalDoc NC` | `LIEFERSCHEIN` | `LFS_STATUS=42, LFS_RTS=1, NETTO<0, AKTIV=J` |
| `BusinessEOS` (Cierre) | `DAILYTOTALS` | Todas las filas del día por KST_ID |

**Naming de archivos XML:**
- Inventario: `TLOG_INVENTORY_<KstCode>_<SequenceNumber>.xml` (o con tipo si `file_name_include_document_type=true`)
- Cierre: `TLOG_Cierre_<KstCode>_<SequenceNumber>.xml`

Ejemplo: `TLOG_INVENTORY_31252_9260505000000.xml` = primer Reception de la EESS 31252 del día 2026-05-05.

---

## Validación de integridad (orphan check)

Al finalizar `create_db`, el pipeline ejecuta **18 chequeos de integridad referencial** sobre las tablas cargadas. Los resultados se guardan en:

```
output/AAAAMMDD/AAAAMMDD_orphans.md
```

El archivo reporta por cada relación FK lógica: total de filas, filas huérfanas, valores distintos huérfanos y un sample de hasta 10 IDs problemáticos.

**Comportamiento:** la presencia de huérfanos **no falla el día**. El pipeline continúa y el reporte es informativo. Los campos `orphans_*` quedan reflejados en el `_day_status.json`.

Ver especificación completa en [`docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md`](docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md).

---

## Archivos de salida

Por cada día procesado en `target_root/AAAAMMDD/`:

| Archivo | Descripción |
|---|---|
| `TLOG_INVENTORY_<KstCode>_<SequenceNumber>.xml` | TLOG OCPRA de inventario (uno por documento) |
| `TLOG_Cierre_<KstCode>_<SequenceNumber>.xml` | TLOG OCPRA de cierre de día |
| `AAAAMMDD_day_status.json` | Estado de cada step con metadata y tiempos |
| `AAAAMMDD_orphans.md` | Reporte de integridad referencial (18 relaciones FK) |
| `AAAAMMDD_pipeline.log` | Log estructurado JSON del día |
| `AAAAMMDD_pipeline.db` | BD SQLite intermedia (eliminada si `local_clean.delete_database=true`) |

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
| Transfer | Todo | Tabla `HIS_LAGERBEW` / `HIS_LAGBEWPOS` no disponible en export actual |

Ver [`docs/mapeos/`](docs/mapeos/) para el detalle completo de cada tipo.

---

## Documentación

| Documento | Descripción |
|---|---|
| [`docs/ARQUITECTURA_PIPELINE_TLOG.md`](docs/ARQUITECTURA_PIPELINE_TLOG.md) | Especificación técnica del pipeline |
| [`docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md`](docs/ARQUITECTURA_PIPELINE_TLOG_addendum_orphans.md) | Spec del orphan check |
| [`docs/Estructura_Tablas_Spec.md`](docs/Estructura_Tablas_Spec.md) | Estructura y relaciones de las tablas CSV |
| [`docs/SEQUENCENUMBER.md`](docs/SEQUENCENUMBER.md) | Especificación del campo SEQUENCENUMBER |
| [`docs/mapeos/MAPEO_TLOG_*.md`](docs/mapeos/) | Mapeo campo a campo de cada tipo de TLOG |

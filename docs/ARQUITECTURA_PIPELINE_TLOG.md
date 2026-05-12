# Arquitectura — Pipeline TLOG OCPRA

**Stack:** Go (sin CGO recomendado).
**Distribución:** binario único + `config.json` en el mismo directorio.
**Fecha:** 2026-05-07.
**Documento spec para codificación.**

---

## 1. Layout del proyecto

```
tlog-pipeline/
├── cmd/
│   └── pipeline/
│       └── main.go                     entrypoint (parse flags, load config, run)
├── internal/
│   ├── config/
│   │   ├── config.go                   structs del config + Load() + Validate()
│   │   ├── flags.go                    CLI flags + override sobre config
│   │   └── defaults.go                 valores por defecto
│   ├── pipeline/
│   │   ├── pipeline.go                 Coordinator: orquesta días y pasos
│   │   ├── step.go                     interface Step + StepResult
│   │   ├── status.go                   DayStatus + RetailStatus + persistencia JSON
│   │   └── runner.go                   ejecuta un día completo (multi-retail)
│   ├── steps/
│   │   ├── ftp_download/step.go        Step "ftp_download"
│   │   ├── read_days/step.go           Step "read_days"
│   │   ├── read_files/step.go          Step "read_files"
│   │   ├── create_db/step.go           Step "create_db"
│   │   ├── create_xml/step.go          Step "create_xml" (por retail)
│   │   ├── local_clean/step.go         Step "local_clean" (por retail)
│   │   ├── ftp_upload/step.go          Step "ftp_upload" (por retail)
│   │   └── ftp_end/step.go             Step "ftp_end" (por retail)
│   ├── tlog/
│   │   ├── tlog.go                     interface TLOGGenerator + factory
│   │   ├── unknown/unknown.go          helper Emit() para [UNKNOWN]
│   │   ├── common/                     header común, formatos fecha, sequence
│   │   ├── reception/                  mapper.go + generator.go + types.go + consts.go
│   │   ├── return_/
│   │   ├── transfer/
│   │   ├── adjustment/
│   │   ├── count/
│   │   ├── fiscaldoc_fc/
│   │   ├── fiscaldoc_nc/
│   │   └── cierre/
│   ├── naming/
│   │   ├── namer.go                    interface FileNamer + DefaultNamer
│   │   └── tlog_order.go               orden canónico NNNN
│   ├── db/
│   │   ├── db.go                       open SQLite, PRAGMAs
│   │   ├── ddl.go                      CREATE TABLE + índices
│   │   ├── load.go                     bulk insert CSV → tabla
│   │   └── lookups.go                  KostatLookup, ArtikelLookup, etc.
│   ├── csvio/
│   │   ├── reader.go                   parser genérico CSV → []map[string]string
│   │   └── files.go                    detectar archivos por patrón Tabla_*.csv
│   ├── ftp/
│   │   ├── client.go                   wrapper FTP (jlaffaye/ftp)
│   │   ├── download.go
│   │   └── upload.go
│   ├── logger/
│   │   └── logger.go                   slog JSON file + console
│   └── timeutil/
│       └── day.go                      parse AAAAMMDD / YYYY-MM-DD, formatos
├── config.json                         junto al binario, en runtime
├── go.mod
└── go.sum
```

---

## 2. Diagrama de flujo

```
                                 ┌──────────────────────┐
                                 │      main.go         │
                                 │  parse flags + load  │
                                 │      config          │
                                 └──────────┬───────────┘
                                            │
                                            ▼
                              ┌──────────────────────────┐
                              │  pipeline.Coordinator    │
                              │  - resolve days a procesar│
                              │  - paralelismo entre días │
                              └──────────┬───────────────┘
                                         │
                  ┌──────────────────────┴──────────────────────┐
                  │ por cada DÍA (paralelo o secuencial)         │
                  ▼                                              │
         ┌────────────────────┐                                  │
         │ pipeline.Runner    │                                  │
         │ ejecuta 1 día      │                                  │
         └────────┬───────────┘                                  │
                  │                                              │
                  ▼                                              │
        ┌─────────────────────────────────────────────┐          │
        │ STEPS GLOBALES (sin retail)                 │          │
        │                                             │          │
        │ ftp_download → read_days → read_files →     │          │
        │ create_db                                   │          │
        │                                             │          │
        │ Falla aquí ⇒ cancela todo el día            │          │
        └─────────────────────────────────────────────┘          │
                  │                                              │
                  ▼                                              │
        ┌─────────────────────────────────────────────┐          │
        │ DETECTAR retails con datos del día          │          │
        │ SELECT DISTINCT KST_ID FROM ... WHERE ...   │          │
        │ → ["00019", "00020", "00021"]               │          │
        └─────────────────────────────────────────────┘          │
                  │                                              │
                  ▼                                              │
        ┌─────────────────────────────────────────────┐          │
        │ por cada RETAIL (paralelo o secuencial      │          │
        │ según parallel_retails_per_day)             │          │
        │                                             │          │
        │ create_xml → local_clean → ftp_upload →     │          │
        │ ftp_end                                     │          │
        │                                             │          │
        │ Falla en retail ⇒ ese retail "failed",      │          │
        │ los demás siguen                            │          │
        └─────────────────────────────────────────────┘          │
                  │                                              │
                  ▼                                              │
        ┌─────────────────────────────────────────────┐          │
        │ Consolidar _day_status.json                 │          │
        │ (ok / completed_with_errors / failed)       │          │
        └─────────────────────────────────────────────┘          │
                                                                 │
                  ◄──────────────────────────────────────────────┘
```

---

## 3. Contrato de Step

### 3.1. Interface común

```go
// internal/pipeline/step.go

type StepKind int
const (
    StepKindGlobal StepKind = iota  // se ejecuta 1 vez por día
    StepKindRetail                  // se ejecuta 1 vez por retail
)

type StepStatus string
const (
    StatusOK      StepStatus = "ok"
    StatusError   StepStatus = "error"
    StatusSkipped StepStatus = "skipped"
    StatusEmpty   StepStatus = "empty"        // ejecutó sin datos para producir
)

type StepResult struct {
    Step       string        `json:"step"`
    Status     StepStatus    `json:"status"`
    Error      string        `json:"error,omitempty"`
    StartedAt  time.Time     `json:"started_at"`
    FinishedAt time.Time     `json:"finished_at"`
    DurationMs int64         `json:"duration_ms"`
    Artifacts  []string      `json:"artifacts,omitempty"` // paths generados
    Meta       map[string]any `json:"meta,omitempty"`     // info extra
}

type Step interface {
    Name() string
    Kind() StepKind
    Enabled() bool
    Run(ctx context.Context, in StepInput) StepResult
}

// StepInput: lo que le llega al paso desde el coordinator
type StepInput struct {
    Day         time.Time            // día a procesar
    DayDir      string               // target_root/AAAAMMDD/
    SourceDir   string               // source_root/AAAAMMDD/
    FinishedDir string               // finished_root/AAAAMMDD/
    RetailID    string               // solo para steps Kind=Retail
    RetailDir   string               // target_root/AAAAMMDD/RETAILID/
    DB          *sql.DB              // disponible desde create_db en adelante
    Logger      *slog.Logger
    GlobalCfg   *config.Config
}
```

### 3.2. Cada step en `internal/steps/<name>/step.go`

Implementa la interface `Step`. Recibe **solo su sub-config** del Coordinator (no el global). Ejemplo:

```go
// internal/steps/create_xml/step.go

type Config struct {
    Enabled         bool   `json:"enabled"`
    FolderSource    string `json:"folder_source"`     // dónde está la DB
    FolderTargetRoot string `json:"folder_target_root"` // dónde escribir XMLs
}

type Step struct {
    cfg Config
}

func New(cfg Config) *Step              { return &Step{cfg: cfg} }
func (s *Step) Name() string            { return "create_xml" }
func (s *Step) Kind() pipeline.StepKind { return pipeline.StepKindRetail }
func (s *Step) Enabled() bool           { return s.cfg.Enabled }
func (s *Step) Run(ctx context.Context, in pipeline.StepInput) pipeline.StepResult { ... }
```

### 3.3. Sub-config que recibe cada step (mapping)

| Step | Recibe del Coordinator |
|---|---|
| `ftp_download` | `cfg.FTPSource` + `cfg.FTPDownload` + flags `--ftp-disabled` |
| `read_days` | `cfg.ReadDays` + `cfg.LocalFolders.SourceRoot` |
| `read_files` | `cfg.ReadFiles` + día + paths |
| `create_db` | `cfg.CreateDB` |
| `create_xml` | `cfg.CreateXML` + retail + paths |
| `local_clean` | `cfg.LocalClean` + `cfg.LocalFolders` + flag `--delete-source` |
| `ftp_upload` | `cfg.FTPTarget` + `cfg.FTPUpload` + retail |
| `ftp_end` | `cfg.FTPSource` + `cfg.FTPEnd` |

### 3.4. Reglas de cada step

1. **Idempotente**: pisar la salida anterior antes de escribir.
2. **Inputs por filesystem**: lee solo lo que dejó el paso previo en disco.
3. **Output por filesystem**: escribe en la ubicación que define su sub-config.
4. **No conoce a otros steps**.
5. **No persiste su propio status**: devuelve `StepResult`, el Coordinator lo persiste.
6. **Carpeta del día origen vacía/ausente** → log a consola, retorna `StatusSkipped` con motivo.

---

## 4. Esquemas JSON de status

Ambos se guardan con prefijo `AAAAMMDD_` y se actualizan después de cada paso (write atómico: write + rename).

### 4.1. `target_root/AAAAMMDD/AAAAMMDD_day_status.json`

Status global del día (multi-retail).

```json
{
  "day": "20260505",
  "started_at": "2026-05-07T10:00:00-03:00",
  "finished_at": "2026-05-07T10:12:34-03:00",
  "duration_ms": 754000,
  "overall_status": "completed_with_errors",
  "total_retails": 3,
  "succeeded_retails": 2,
  "failed_retails_count": 1,
  "failed_retails": ["00021"],
  "global_steps": [
    { "step": "ftp_download", "status": "ok",      "duration_ms": 1200 },
    { "step": "read_days",    "status": "ok",      "duration_ms":   30 },
    { "step": "read_files",   "status": "ok",      "duration_ms":   80 },
    { "step": "create_db",    "status": "ok",      "duration_ms": 4500 }
  ],
  "retails": {
    "00019": "ok",
    "00020": "ok",
    "00021": "error"
  }
}
```

**`overall_status`** posibles:
- `ok`: todos los retails OK + todos los steps globales OK.
- `completed_with_errors`: al menos un retail OK + al menos uno con error.
- `failed`: todos los retails fallaron, **o** un step global falló (no se ejecutaron retails).
- `skipped`: día no procesado (carpeta origen vacía).

### 4.2. `target_root/AAAAMMDD/RETAILID/AAAAMMDD_pipeline_status.json`

Status del retail.

```json
{
  "day": "20260505",
  "retail_id": "00019",
  "started_at": "2026-05-07T10:05:00-03:00",
  "finished_at": "2026-05-07T10:08:12-03:00",
  "duration_ms": 192000,
  "status": "ok",
  "steps": [
    {
      "step": "create_xml",
      "status": "ok",
      "started_at": "2026-05-07T10:05:00-03:00",
      "finished_at": "2026-05-07T10:07:30-03:00",
      "duration_ms": 150000,
      "artifacts": [
        "00019-20260505-0001.xml",
        "00019-20260505-0002.xml",
        "00019-20260505-0008.xml"
      ],
      "meta": { "tlogs_generated": 3, "tlogs_empty": 5 }
    },
    { "step": "local_clean", "status": "ok",      "duration_ms": 800 },
    { "step": "ftp_upload",  "status": "ok",      "duration_ms": 41200 },
    { "step": "ftp_end",     "status": "ok",      "duration_ms":   200 }
  ]
}
```

### 4.3. Copia a finished

Al finalizar `local_clean` exitosamente, el `_day_status.json` se **copia** a `finished_root/AAAAMMDD/AAAAMMDD_day_status.json`. Snapshot del momento; no se actualiza después.

---

## 5. Configuración final

### 5.1. `config.json`

```json
{
  "ftp_source": {
    "url": "",
    "username": "",
    "pass": ""
  },
  "ftp_target": {
    "url": "",
    "username": "",
    "pass": ""
  },
  "local_folders": {
    "source_root": "",
    "target_root": "",
    "finished_root": ""
  },
  "process": {
    "mode": "ALL",
    "execution_mode": "PARALLEL",
    "parallel_retails_per_day": false,
    "begin_date_offset": "00:00:00",
    "end_date_offset": "23:59:59",
    "operator_id": "admin"
  },
  "ftp_download": {
    "enabled": false,
    "folder_source_root": "",
    "folder_target_root": ""
  },
  "read_days": {
    "enabled": true,
    "folder_source_root": ""
  },
  "read_files": {
    "enabled": true,
    "folder_source": "",
    "folder_target_root": "",
    "expected_files": [
      "Kostst_*.csv", "Liefer_*.csv", "Warengruppe_*.csv", "Vpckeinh_*.csv",
      "Artikel_*.csv", "Lieferschein_*.csv", "Lieferpos_*.csv",
      "Inventur_*.csv", "Invposart_*.csv", "His_verbrauch_*.csv",
      "Dailytotals_*.csv"
    ]
  },
  "create_db": {
    "enabled": true,
    "separator": ",",
    "folder_source": "",
    "folder_target_root": ""
  },
  "create_xml": {
    "enabled": true,
    "folder_source": "",
    "folder_target_root": ""
  },
  "ftp_upload": {
    "enabled": false,
    "folder_source": "",
    "folder_target": ""
  },
  "local_clean": {
    "enabled": true,
    "folder_source": "",
    "folder_target": "",
    "delete_source": false
  },
  "ftp_end": {
    "enabled": false,
    "folder_source": "",
    "folder_target": ""
  }
}
```

### 5.2. Cambios respecto del sample original

| Cambio | Motivo |
|---|---|
| Sintaxis JSON corregida (comas, dos puntos, comillas) | el sample tenía errores de parsing |
| Eliminado `start_step_name` | se reemplaza por `enabled` por step + `--step` |
| Agregado `parallel_retails_per_day` | paralelismo dentro del día |
| Agregado `local_clean.delete_database` | qué hacer con la BD SQLite intermedia |
| Agregado `begin_date_offset`, `end_date_offset`, `operator_id` | comunes a todos los TLOG |
| `local_folders.source_root` pasa de array a string | unificar tipo |
| Agregado `read_files.expected_files` | validación explícita de archivos requeridos |

---

## 6. CLI flags

| Flag | Tipo | Pisa | Default |
|---|---|---|---|
| `--config <path>` | string | path al config.json | `./config.json` (junto al ejecutable) |
| `--day <AAAAMMDD\|YYYY-MM-DD>` | string | día específico a procesar | (todos los disponibles) |
| `--folder-source-root <path>` | string | `local_folders.source_root` | — |
| `--folder-target-root <path>` | string | `local_folders.target_root` | — |
| `--folder-finished <path>` | string | `local_folders.finished_root` | — |
| `--ftp-disabled` | bool | `ftp_download.enabled = ftp_upload.enabled = ftp_end.enabled = false` | false |
| `--delete-source` | bool | `local_clean.delete_source` | false |
| `--step <name>` | string | ejecuta SOLO ese paso (requiere `--day`) | — |

### 6.1. Reglas de validación

- `--step` requiere `--day` (sí o sí). Sin `--day` → error fatal.
- `--step` solo acepta UN paso. Lista no soportada en v1.
- `--day` acepta `20260505` o `2026-05-05`. Internamente normaliza a `AAAAMMDD`.
- Si ningún flag de carpeta se pasa, los del config son obligatorios. Si faltan en config → error fatal.

### 6.2. Modos de ejecución resultantes

| Modo | Cómo se invoca | Qué hace |
|---|---|---|
| Run completo | `./pipeline` | Procesa todos los días disponibles con todos los pasos `enabled: true` |
| Run de un día | `./pipeline --day 20260505` | Procesa ese día completo |
| Run de un paso | `./pipeline --day 20260505 --step create_xml` | Ejecuta solo ese paso de ese día |
| Run sin FTP | `./pipeline --ftp-disabled` | Skipea los 3 steps FTP |

---

## 7. Naming spec

### 7.1. Archivos XML generados

Formato: `[KST_CODE]-AAAAMMDD-NNNN.xml`

Ejemplos:
```
00019-20260505-0001.xml   ← Reception
00019-20260505-0002.xml   ← Return
00019-20260505-0008.xml   ← Cierre
```

### 7.2. Orden canónico NNNN

```go
// internal/naming/tlog_order.go

var TLOGOrder = []TLOGType{
    TLOGReception,    // 0001
    TLOGReturn,       // 0002
    TLOGTransfer,     // 0003
    TLOGAdjustment,   // 0004
    TLOGCount,        // 0005
    TLOGFiscalDocFC,  // 0006
    TLOGFiscalDocNC,  // 0007
    TLOGCierre,       // 0008
}
```

### 7.3. Interface `FileNamer`

```go
// internal/naming/namer.go

type FileNamer interface {
    XMLFile(retailID string, day time.Time, tlog TLOGType) string
    DayStatusFile(day time.Time) string         // "20260505_day_status.json"
    RetailStatusFile(day time.Time) string      // "20260505_pipeline_status.json"
    DBFile(day time.Time) string                // "20260505_pipeline.db"
    LogFile(day time.Time) string               // "20260505_pipeline.log"
}

type DefaultNamer struct{}

func (DefaultNamer) XMLFile(retail string, day time.Time, t TLOGType) string {
    n := indexOf(TLOGOrder, t) + 1
    return fmt.Sprintf("%s-%s-%04d.xml", retail, day.Format("20060102"), n)
}
```

Cambiar la estrategia de naming = implementar otra `FileNamer` y enchufarla en el Coordinator.

---

## 8. Mappers de TLOG

### 8.1. Estructura por TLOG type

Cada TLOG type vive en `internal/tlog/<name>/` con:

```
reception/
├── types.go         tipos de input (filas DB) y output (XML)
├── consts.go        valores fijos del Excel
├── mapper.go        ReceptionMapper struct + 1 método por campo XML
├── generator.go     orquesta: lee DB, instancia mapper, escribe XML
└── mapper_test.go
```

### 8.2. Patrón del mapper

```go
// internal/tlog/reception/mapper.go

type ReceptionMapper struct {
    Header      LieferscheinRow
    Lines       []LieferposRow
    Kostst      KostatLookup
    Artikel     ArtikelLookup
    Vpckeinh    VpckeinhLookup
    BusinessDay time.Time
    Config      MapperConfig
    SeqNumber   string
}

// ─── Cabecera ─────────────────────────────────────────

func (m *ReceptionMapper) RetailStoreID() string  { ... }
func (m *ReceptionMapper) WorkstationID() string  { return WorkstationIDFixed }
func (m *ReceptionMapper) SequenceNumber() string { return m.SeqNumber }
func (m *ReceptionMapper) BusinessDayDate() string { ... }
func (m *ReceptionMapper) Period() string    { return "0" }
func (m *ReceptionMapper) Subperiod() string { return "0" }
func (m *ReceptionMapper) PeriodCode() string    { return "" }
func (m *ReceptionMapper) SubPeriodCode() string { return "" }
func (m *ReceptionMapper) BeginDateTime() string { ... }
func (m *ReceptionMapper) EndDateTime() string   { ... }
func (m *ReceptionMapper) OperatorID() string    { return m.Config.OperatorID }
func (m *ReceptionMapper) ContraReferenceNumber() string {
    return unknown.Emit(
        "Generado desde Web",
        "Excel: 'DESCRIPCIÓN DEL AJUSTE' sin campo origen. ¿Literal o INV_INFO?",
    )
}

// ─── Detalle (lineIdx 0..N) ───────────────────────────

func (m *ReceptionMapper) DetSequenceNumber(lineIdx int) string  { ... }
func (m *ReceptionMapper) Item(lineIdx int) string                { ... }
func (m *ReceptionMapper) UOMUnits(lineIdx int) string            { ... }
func (m *ReceptionMapper) ItemBrand(lineIdx int) string           { return "0" }
func (m *ReceptionMapper) ItemDescription(lineIdx int) string     { ... }
func (m *ReceptionMapper) UnitBaseCostAmount(lineIdx int) string  { ... }
func (m *ReceptionMapper) UnitCount(lineIdx int) string           { ... }
func (m *ReceptionMapper) PickupCode(lineIdx int) string {
    return unknown.Emit("S1", "Excel: S1/S2 sin campo origen en ARTIKEL")
}
```

### 8.3. Helper para `[UNKNOWN]`

```go
// internal/tlog/unknown/unknown.go

// Emit retorna la string formateada según convención del proyecto:
//   "[UNKNOWN] - {valor xml} - {dudas/opciones}"
func Emit(xmlValue, doubts string) string {
    return fmt.Sprintf("[UNKNOWN] - {%s} - {%s}", xmlValue, doubts)
}
```

### 8.4. Reglas de mappers

1. **1 método por campo XML**. No agrupar varios campos en una función.
2. **Métodos puros**: solo dependen de los campos del struct, sin globals.
3. **Métodos de cabecera sin args**, métodos de detalle con `lineIdx int`.
4. **Constantes en `consts.go`** del paquete (no inline en métodos).
5. **`unknown.Emit` para todo lo que el Excel marque como pendiente** o sin origen claro.
6. **Test por método** con casos del XML real de `/mnt/project/`.
7. **Cambiar mapeo = editar 1 método**. Sin tocar generator ni types.

### 8.5. Generator orquesta

```go
// internal/tlog/reception/generator.go

type Generator struct {
    cfg     MapperConfig
    namer   naming.FileNamer
    seqGen  SequenceGenerator
    kostst  KostatLookup
    artikel ArtikelLookup
    vpckeinh VpckeinhLookup
}

// Generate escribe el XML del retail+día en outputDir.
// Retorna el path del archivo generado, o "" si no había datos (StatusEmpty).
func (g *Generator) Generate(
    ctx context.Context,
    retailID string,
    day time.Time,
    db *sql.DB,
    outputDir string,
) (path string, err error) {
    headers, err := loadLieferscheinHeaders(db, retailID, day)
    if err != nil { return "", err }
    if len(headers) == 0 { return "", nil } // empty

    var xmlDocs []ReceptionXML
    for _, h := range headers {
        lines, err := loadLieferposLines(db, h.LfsID)
        if err != nil { return "", err }

        m := &ReceptionMapper{
            Header: h, Lines: lines,
            Kostst: g.kostst, Artikel: g.artikel, Vpckeinh: g.vpckeinh,
            BusinessDay: day,
            Config: g.cfg,
            SeqNumber: g.seqGen.Next(retailID),
        }
        xmlDocs = append(xmlDocs, buildXMLFromMapper(m))
    }

    fname := g.namer.XMLFile(retailID, day, naming.TLOGReception)
    fullPath := filepath.Join(outputDir, fname)
    return fullPath, writeXMLFile(fullPath, xmlDocs)
}
```

### 8.6. Filtros de generación por TLOG type

Definidos en cada `generator.go` según los mapeos MD existentes:

| TLOG | Tabla driver | Filtro |
|---|---|---|
| Reception | LIEFERSCHEIN | `LFS_RTS != 1 AND LFS_NETTO > 0 AND LFS_STATUS IN (42, ...) AND AKTIV='J'` |
| Return | LIEFERSCHEIN | `LFS_RTS = 1 AND LFS_NETTO < 0 AND AKTIV='J'` |
| FiscalDocFC | LIEFERSCHEIN | `LFS_STATUS=42 AND LFS_RTS!=1 AND LFS_NETTO>0 AND AKTIV='J'` |
| FiscalDocNC | LIEFERSCHEIN | `LFS_STATUS=42 AND LFS_RTS=1 AND LFS_NETTO<0 AND AKTIV='J'` |
| Adjustment | INVENTUR | `INV_STATUS=8 AND INV_TYP=4` |
| Count | INVENTUR | (definido en MAPEO_TLOG_INVENTORY_COUNT_REAL.md) |
| Transfer | (definido en MAPEO_TLOG_INVENTORY_TRANSFER.md) | — |
| Cierre | DAILYTOTALS | `DAY_DATE = <día> AND filtros KOSTST/ARTIKEL` |

Todos los filtros se aplican con `WHERE KST_ID = <retail>` (la DB es global del día).

---

## 9. Manejo de errores y reanudación

### 9.1. Tabla de combinaciones

| Escenario | Comportamiento |
|---|---|
| Carpeta `source_root/AAAAMMDD/` no existe | log a consola, NO se crea status, día no se procesa |
| Carpeta existe pero vacía | log a consola, NO se crea status, día no se procesa |
| Falla `ftp_download` / `read_days` / `read_files` / `create_db` | step global → cancela todo el día. `_day_status.json` con `overall_status: "failed"`. NO se procesan retails |
| Falla `create_xml` para retail X | retail X queda `failed`. Otros retails siguen. `_day_status.json` queda `completed_with_errors` (si al menos uno OK) |
| Falla `local_clean` para retail X | igual que arriba |
| Falla `ftp_upload` para retail X | igual. CSVs YA fueron movidos a finished (local_clean ya corrió). XMLs quedan en target |
| Falla `ftp_end` para retail X | igual |
| Día completo OK | `overall_status: "ok"`. `_day_status.json` se copia a finished |
| Día sin retails con datos (steps globales OK pero no hay datos) | `overall_status: "ok"` con `total_retails: 0`. `global_steps` todos OK |
| TLOG type sin datos en un retail | step `create_xml` retorna `StatusOK` con `meta.tlogs_empty` incrementado. NO se genera ese XML específico |

### 9.2. Reanudación

No hay `start_step_name`. Las formas de reanudar son:

1. **Volver a correr todo**: `./pipeline --day 20260505`. Idempotente, pisa todo.
2. **Correr solo un paso**: `./pipeline --day 20260505 --step create_xml`. Asume que los pasos previos dejaron sus artefactos en filesystem.
3. **Modificar `enabled`** en config: setear `false` los pasos que ya corrieron OK y correr de nuevo.

### 9.3. Política de fail-fast

- **Día**: fail-fast por step. Primer step global que falla → cancela día.
- **Retail dentro del día**: fail-fast por step del retail. Primer step del retail que falla → cancela ese retail.
- **Entre días**: aislados. Un día failed no afecta a otros días en paralelo.
- **Entre retails del mismo día**: aislados. Un retail failed no afecta a otros retails del mismo día.

### 9.4. Logging técnico

- Path: `target_root/AAAAMMDD/AAAAMMDD_pipeline.log`.
- Formato: JSON estructurado vía `log/slog`.
- Niveles: `DEBUG | INFO | WARN | ERROR`.
- También a stdout en formato humano (para operación manual).
- Cada step recibe un logger con campos `day`, `step`, `retail_id` (si aplica) ya rellenados.

---

## 10. Detalles técnicos por componente

### 10.1. `internal/db/`

- SQLite vía `modernc.org/sqlite` (sin CGO).
- PRAGMAs en `Open()`: `journal_mode=WAL`, `synchronous=NORMAL`, `foreign_keys=OFF` durante carga.
- DDL en `ddl.go` siguiendo `Estructura_Tablas_Spec.md` del proyecto.
- Lookups (`KostatLookup`, `ArtikelLookup`, etc.) son cachés in-memory cargados al abrir la DB.

### 10.2. `internal/csvio/`

- `encoding/csv` stdlib.
- Detección de archivos por patrón: `Tabla_*.csv` (regex sobre nombre de archivo).
- Strings vacíos → NULL.
- Conversión de tipos al insertar (no en lectura).

### 10.3. `internal/ftp/`

- Lib: `github.com/jlaffaye/ftp`.
- Wrapper con retry (3 intentos, backoff exponencial).
- Modos `download`, `upload`, `move` (para `ftp_end`).

### 10.4. `internal/timeutil/`

```go
func ParseDay(s string) (time.Time, error) // "20260505" o "2026-05-05"
func FormatYYYYMMDD(t time.Time) string    // "20260505"
```

### 10.5. `internal/pipeline/`

- `Coordinator` levanta días con `read_days` y los despacha a `Runner`.
- `Runner` ejecuta los steps de un día.
- Paralelismo: `errgroup.Group` con límite (configurable, default `runtime.NumCPU()`).
- Cancelación: `context.Context` propagado, primer error en step global cancela el resto del día.

---

## 11. Resumen ejecutivo

| Concepto | Decisión |
|---|---|
| Idempotencia | Cada step pisa lo anterior |
| Modo de ejecución | `enabled` por step + `--step` para single |
| Paralelismo | Entre días (`PARALLEL`) × dentro del día (`parallel_retails_per_day`) |
| Steps globales | `ftp_download`, `read_days`, `read_files`, `create_db` |
| Steps por retail | `create_xml`, `local_clean`, `ftp_upload`, `ftp_end` |
| DB | Una por día con todos los retails (filtro por `KST_ID`) |
| Naming XMLs | `[KST_CODE]-AAAAMMDD-NNNN.xml` (NNNN: orden canónico) |
| Naming status/log/db | `AAAAMMDD_*` con guion bajo |
| Status | `_day_status.json` + `_pipeline_status.json` por retail |
| Copia status a finished | Al finalizar `local_clean` |
| Mappers TLOG | 1 archivo `mapper.go` con struct + 1 método por campo XML |
| `[UNKNOWN]` | `unknown.Emit(xmlValue, doubts)` |
| Fail global | Cancela día. Retails no se ejecutan |
| Fail retail | Aísla a ese retail. Otros del mismo día siguen |

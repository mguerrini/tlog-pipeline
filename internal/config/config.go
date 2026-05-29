// Package config carga config.json y aplica defaults + overrides de flags.
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/opessa/tlog-pipeline/internal/naming"
)

type FTP struct {
	URL      string `json:"url"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Pass     string `json:"pass"`
}

type LocalFolders struct {
	All          string `json:"all"`
	SourceRoot   string `json:"source_root"`
	TargetRoot   string `json:"target_root"`
	FinishedRoot string `json:"finished_root"`
}

// FtpFolders agrupa las rutas remotas usadas por los steps de FTP. Centraliza
// la configuración para que ningún step tenga que repetir folder_source_root /
// folder_target_root: ftp_download usa SourceRoot, ftp_upload usa TargetRoot y
// ftp_end mueve archivos de SourceRoot a FinishedRoot.
type FtpFolders struct {
	SourceRoot   string `json:"source_root"`
	FinishedRoot string `json:"finished_root"`
	TargetRoot   string `json:"target_root"`
}

// Output controla qué TLOGs (XMLs) se generan en create_xml / create_xml_sql.
// Cada flag corresponde a un naming.TLOGType. Si la sección entera se omite
// del JSON, defaults.go la resuelve en "todo true" (compat retro). Si la
// sección está presente pero falta un flag, ese flag default-ea a true.
type Output struct {
	Cierre               bool `json:"cierre"`
	InventoryReception   bool `json:"inventory_reception"`
	InventoryFiscalDocFC bool `json:"inventory_fiscaldoc_fc"`
	InventoryFiscalDocNC bool `json:"inventory_fiscaldoc_nc"`
	InventoryReturn      bool `json:"inventory_return"`
	InventoryAdjustment  bool `json:"inventory_adjustment"`
	InventoryCount       bool `json:"inventory_count"`
	InventoryTransfer    bool `json:"inventory_transfer"`
}

// UnmarshalJSON: cualquier flag omitido default-ea a true. Esto distingue
// "campo ausente" de "campo explícito en false" — sin esto, json.Unmarshal
// produciría false en ambos casos y no podríamos respetar un opt-out
// explícito (false) cuando todos los flags están en false.
func (o *Output) UnmarshalJSON(data []byte) error {
	raw := struct {
		Cierre               *bool `json:"cierre"`
		InventoryReception   *bool `json:"inventory_reception"`
		InventoryFiscalDocFC *bool `json:"inventory_fiscaldoc_fc"`
		InventoryFiscalDocNC *bool `json:"inventory_fiscaldoc_nc"`
		InventoryReturn      *bool `json:"inventory_return"`
		InventoryAdjustment  *bool `json:"inventory_adjustment"`
		InventoryCount       *bool `json:"inventory_count"`
		InventoryTransfer    *bool `json:"inventory_transfer"`
	}{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*o = Output{
		Cierre:               boolOrTrue(raw.Cierre),
		InventoryReception:   boolOrTrue(raw.InventoryReception),
		InventoryFiscalDocFC: boolOrTrue(raw.InventoryFiscalDocFC),
		InventoryFiscalDocNC: boolOrTrue(raw.InventoryFiscalDocNC),
		InventoryReturn:      boolOrTrue(raw.InventoryReturn),
		InventoryAdjustment:  boolOrTrue(raw.InventoryAdjustment),
		InventoryCount:       boolOrTrue(raw.InventoryCount),
		InventoryTransfer:    boolOrTrue(raw.InventoryTransfer),
	}
	return nil
}

func boolOrTrue(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

// Enabled indica si el TLOG de tipo t debe generarse según la configuración.
func (o Output) Enabled(t naming.TLOGType) bool {
	switch t {
	case naming.TLOGCierre:
		return o.Cierre
	case naming.TLOGReception:
		return o.InventoryReception
	case naming.TLOGFiscalDocFC:
		return o.InventoryFiscalDocFC
	case naming.TLOGFiscalDocNC:
		return o.InventoryFiscalDocNC
	case naming.TLOGReturn:
		return o.InventoryReturn
	case naming.TLOGAdjustment:
		return o.InventoryAdjustment
	case naming.TLOGCount:
		return o.InventoryCount
	case naming.TLOGTransfer:
		return o.InventoryTransfer
	}
	return false
}

// Logs habilita / deshabilita la escritura de los archivos de log y reporte
// que el pipeline produce por día. Misma semántica de defaults que Output:
// sección ausente → defaults.go la resuelve en "todo true"; campo ausente
// dentro de una sección presente → default-ea a true (ver UnmarshalJSON).
type Logs struct {
	PipelineEnabled  bool `json:"pipeline_enabled"`   // AAAAMMDD_pipeline.log
	DayStatusEnabled bool `json:"day_status_enabled"` // AAAAMMDD_day_status.json
	SQLDBLoad        bool `json:"sql_db_load"`        // AAAAMMDD_sqldb_load.md
}

// UnmarshalJSON: flags omitidos default-ean a true.
func (l *Logs) UnmarshalJSON(data []byte) error {
	raw := struct {
		PipelineEnabled  *bool `json:"pipeline_enabled"`
		DayStatusEnabled *bool `json:"day_status_enabled"`
		SQLDBLoad        *bool `json:"sql_db_load"`
	}{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*l = Logs{
		PipelineEnabled:  boolOrTrue(raw.PipelineEnabled),
		DayStatusEnabled: boolOrTrue(raw.DayStatusEnabled),
		SQLDBLoad:        boolOrTrue(raw.SQLDBLoad),
	}
	return nil
}

type Process struct {
	Mode                        string `json:"mode"`              // ALL | DAY
	ExecutionMode               string `json:"execution_mode"`    // PARALLEL | SERIAL
	ParallelRetailsPerDay       bool   `json:"parallel_retails_per_day"`
	BeginDateOffset             string `json:"begin_date_offset"` // HH:MM:SS
	EndDateOffset               string `json:"end_date_offset"`   // HH:MM:SS
	OperatorID                  string `json:"operator_id"`
	// FileNameIncludeDocumentType controla si el nombre del XML incluye el
	// tipo de documento. true → TLOG_INVENTORY_<Tipo>_<kst>_<seq>.xml;
	// false → TLOG_INVENTORY_<kst>_<seq>.xml.
	FileNameIncludeDocumentType bool `json:"file_name_include_document_type"`
	// IsProduction controla qué ART_NR se usan en las queries de fiscal docs.
	// false (default): 1120/1100/1098/1096 (entorno de pruebas).
	// true: 2207/2204/2205/2206 (entorno productivo).
	IsProduction bool `json:"is_production"`
}

// Cada step toma sus rutas de cfg.FtpFolders / cfg.LocalFolders y solo expone
// los flags de comportamiento que le son propios (enabled, separator, etc.).
type FTPDownload struct {
	Enabled bool `json:"enabled"`
	// SplitByDate: si true, los archivos remotos están todos juntos bajo
	// ftp_folders.source_root y se bajan a local_folders.all (después un
	// split_by_date local los reparte por día). Si false, los archivos remotos
	// ya están agrupados en subcarpetas con nombre de fecha (AAAAMMDD o
	// AAAA-MM-DD) y se bajan a local_folders.source_root/AAAAMMDD/ — el formato
	// AAAA-MM-DD se normaliza a AAAAMMDD al crear la carpeta local.
	SplitByDate bool `json:"split_by_date"`
}

type SplitByDate struct {
	Enabled bool `json:"enabled"`
}

type ReadDays struct {
	Enabled bool `json:"enabled"`
}

type ReadFiles struct {
	Enabled       bool                `json:"enabled"`
	ExpectedFiles []string            `json:"expected_files"`
	ClearCols     map[string][]string `json:"clear_cols"` // tabla → columnas a vaciar antes de importar
}

type CreateDB struct {
	Enabled   bool   `json:"enabled"`
	Separator string `json:"separator"`
}

type CreateXML struct {
	Enabled bool `json:"enabled"`
}

type FTPUpload struct {
	Enabled bool `json:"enabled"`
}

type LocalClean struct {
	Enabled        bool `json:"enabled"`
	DeleteSource   bool `json:"delete_source"`
	DeleteDatabase bool `json:"delete_database"`
}

type FTPEnd struct {
	Enabled bool `json:"enabled"`
	// DeleteLocalSource: si true, una vez que ftp_end completó el archivado
	// remoto borra también la carpeta local target_root/AAAAMMDD entera.
	DeleteLocalSource bool `json:"delete_local_source"`
}

// Config es el modelo completo de config.json.
type Config struct {
	FTPSource    FTP          `json:"ftp_source"`
	FTPTarget    FTP          `json:"ftp_target"`
	FtpFolders   FtpFolders   `json:"ftp_folders"`
	LocalFolders LocalFolders `json:"local_folders"`
	Output       *Output      `json:"output"`
	Logs         *Logs        `json:"logs"`
	Process      Process      `json:"process"`
	FTPDownload  FTPDownload  `json:"ftp_download"`
	SplitByDate  SplitByDate  `json:"split_by_date"`
	ReadDays     ReadDays     `json:"read_days"`
	ReadFiles    ReadFiles    `json:"read_files"`
	CreateDB     CreateDB     `json:"create_db"`
	CreateXML    CreateXML    `json:"create_xml"`
	FTPUpload    FTPUpload    `json:"ftp_upload"`
	LocalClean   LocalClean   `json:"local_clean"`
	FTPEnd       FTPEnd       `json:"ftp_end"`
}

// Load lee config.json y aplica defaults a campos vacíos.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leer config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parsear config: %w", err)
	}
	applyDefaults(&c)
	return &c, nil
}

// Validate chequea las invariantes mínimas que el pipeline necesita.
func (c *Config) Validate() error {
	if c.LocalFolders.SourceRoot == "" {
		return fmt.Errorf("local_folders.source_root requerido")
	}
	if c.LocalFolders.TargetRoot == "" {
		return fmt.Errorf("local_folders.target_root requerido")
	}
	if c.Process.BeginDateOffset == "" || c.Process.EndDateOffset == "" {
		return fmt.Errorf("process.begin_date_offset y end_date_offset requeridos")
	}
	if c.Process.OperatorID == "" {
		return fmt.Errorf("process.operator_id requerido")
	}
	return nil
}

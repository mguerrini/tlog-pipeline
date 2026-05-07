// Package config carga config.json y aplica defaults + overrides de flags.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type FTP struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Pass     string `json:"pass"`
}

type LocalFolders struct {
	SourceRoot   string `json:"source_root"`
	TargetRoot   string `json:"target_root"`
	FinishedRoot string `json:"finished_root"`
}

type Process struct {
	Mode                  string `json:"mode"`              // ALL | DAY
	ExecutionMode         string `json:"execution_mode"`    // PARALLEL | SERIAL
	ParallelRetailsPerDay bool   `json:"parallel_retails_per_day"`
	KeepDBAfterRun        bool   `json:"keep_db_after_run"`
	BeginDateOffset       string `json:"begin_date_offset"` // HH:MM:SS
	EndDateOffset         string `json:"end_date_offset"`   // HH:MM:SS
	OperatorID            string `json:"operator_id"`
}

type FTPDownload struct {
	Enabled          bool   `json:"enabled"`
	FolderRootSource string `json:"folder_root_source"`
	FolderRootTarget string `json:"folder_root_target"`
}

type ReadDays struct {
	Enabled          bool   `json:"enabled"`
	FolderSourceRoot string `json:"folder_source_root"`
}

type ReadFiles struct {
	Enabled          bool     `json:"enabled"`
	FolderSource     string   `json:"folder_source"`
	FolderTargetRoot string   `json:"folder_target_root"`
	ExpectedFiles    []string `json:"expected_files"`
}

type CreateDB struct {
	Enabled          bool   `json:"enabled"`
	Separator        string `json:"separator"`
	FolderSource     string `json:"folder_source"`
	FolderTargetRoot string `json:"folder_target_root"`
}

type CreateXML struct {
	Enabled          bool   `json:"enabled"`
	FolderSource     string `json:"folder_source"`
	FolderTargetRoot string `json:"folder_target_root"`
}

type FTPUpload struct {
	Enabled      bool   `json:"enabled"`
	FolderSource string `json:"folder_source"`
	FolderTarget string `json:"folder_target"`
}

type LocalClean struct {
	Enabled      bool   `json:"enabled"`
	FolderSource string `json:"folder_source"`
	FolderTarget string `json:"folder_target"`
	DeleteSource bool   `json:"delete_source"`
}

type FTPEnd struct {
	Enabled      bool   `json:"enabled"`
	FolderSource string `json:"folder_source"`
	FolderTarget string `json:"folder_target"`
}

// Config es el modelo completo de config.json.
type Config struct {
	FTPSource    FTP          `json:"ftp_source"`
	FTPTarget    FTP          `json:"ftp_target"`
	LocalFolders LocalFolders `json:"local_folders"`
	Process      Process      `json:"process"`
	FTPDownload  FTPDownload  `json:"ftp_download"`
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

package config

import (
	"flag"
	"fmt"
	"os"
)

// Flags es el resultado del parseo de los flags CLI.
type Flags struct {
	ConfigPath        string
	Day               string
	FolderSourceRoot  string
	FolderTargetRoot  string
	FolderFinished    string
	FTPDisabled       bool
	DeleteSource      bool
	Step              string
}

// ParseFlags lee los flags de os.Args[1:].
func ParseFlags() *Flags {
	fs := flag.NewFlagSet("pipeline", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	f := &Flags{}
	fs.StringVar(&f.ConfigPath, "config", "./config.json", "path al config.json")
	fs.StringVar(&f.Day, "day", "", "día específico AAAAMMDD o AAAA-MM-DD (vacío = todos los disponibles)")
	fs.StringVar(&f.FolderSourceRoot, "folder-source-root", "", "override local_folders.source_root")
	fs.StringVar(&f.FolderTargetRoot, "folder-target-root", "", "override local_folders.target_root")
	fs.StringVar(&f.FolderFinished, "folder-finished", "", "override local_folders.finished_root")
	fs.BoolVar(&f.FTPDisabled, "ftp-disabled", false, "desactiva los 3 steps de FTP")
	fs.BoolVar(&f.DeleteSource, "delete-source", false, "fuerza local_clean.delete_source = true")
	fs.StringVar(&f.Step, "step", "", "ejecuta solo ese paso (requiere --day)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if f.Step != "" && f.Day == "" {
		fmt.Fprintln(os.Stderr, "error: --step requiere --day")
		os.Exit(2)
	}
	return f
}

// Apply aplica los flags sobre la Config in-place.
func (f *Flags) Apply(c *Config) {
	if f.FolderSourceRoot != "" {
		c.LocalFolders.SourceRoot = f.FolderSourceRoot
	}
	if f.FolderTargetRoot != "" {
		c.LocalFolders.TargetRoot = f.FolderTargetRoot
	}
	if f.FolderFinished != "" {
		c.LocalFolders.FinishedRoot = f.FolderFinished
	}
	if f.FTPDisabled {
		c.FTPDownload.Enabled = false
		c.FTPUpload.Enabled = false
		c.FTPEnd.Enabled = false
	}
	if f.DeleteSource {
		c.LocalClean.DeleteSource = true
	}
}

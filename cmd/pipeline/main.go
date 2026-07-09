//go:generate goversioninfo -o resource.syso versioninfo.json
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/opessa/tlog-pipeline/internal/config"
	"github.com/opessa/tlog-pipeline/internal/logger"
	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/steps/create_sql_db"
	"github.com/opessa/tlog-pipeline/internal/steps/create_xml_sql"
	"github.com/opessa/tlog-pipeline/internal/steps/ftp_end"
	"github.com/opessa/tlog-pipeline/internal/steps/ftp_upload"
	"github.com/opessa/tlog-pipeline/internal/steps/local_clean"
	"github.com/opessa/tlog-pipeline/internal/steps/read_files"
	"github.com/opessa/tlog-pipeline/internal/steps/split_by_date"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

// Version se inyecta en build time vía -ldflags "-X main.Version=..."
var Version = "7.0.0"

func main() {
	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-version" || a == "-v" {
			fmt.Println("tlog-pipeline", Version)
			return
		}
	}

	flags := config.ParseFlags()

	// Cargar config
	cfg, err := config.Load(flags.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error cargando config: %v\n", err)
		os.Exit(1)
	}
	flags.Apply(cfg)

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "configuración inválida: %v\n", err)
		os.Exit(1)
	}

	// Logger global (sin archivo; cada día abre su propio log)
	log, closer, err := logger.New(slog.LevelInfo, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error inicializando logger: %v\n", err)
		os.Exit(1)
	}
	defer closer.Close()

	log.Info("tlog-pipeline iniciado",
		"version", Version,
		"config", flags.ConfigPath,
		"mode", cfg.Process.Mode,
		"execution_mode", cfg.Process.ExecutionMode,
	)

	// Parsear --day si se indicó
	var onlyDay time.Time
	if flags.Day != "" {
		onlyDay, err = timeutil.ParseDay(flags.Day)
		if err != nil {
			fmt.Fprintf(os.Stderr, "día inválido: %v\n", err)
			os.Exit(1)
		}
	}

	// Fase 1: download + generación de XML per-día. ftp_download corre una
	// sola vez antes (pre-step global en el Coordinator) — los demás se
	// ejecutan por día sobre source_root/AAAAMMDD → target_root/AAAAMMDD.
	phase1Steps := []pipeline.Step{
		split_by_date.Step{},
		read_files.Step{},
		create_sql_db.Step{},
		create_xml_sql.Step{},
	}
	// Fase 2: upload + cierre. Per-día sobre target_root/AAAAMMDD, gobernada
	// por ftp_status.json en cada carpeta. Siempre corre, incluso si la fase 1
	// no produjo días nuevos.
	phase2Steps := []pipeline.Step{
		ftp_upload.Step{},
		ftp_end.Step{},
		local_clean.Step{},
	}

	coord := pipeline.NewCoordinator(cfg, phase1Steps, phase2Steps, onlyDay, flags.Step, log)
	ctx := context.Background()

	if err := coord.Run(ctx); err != nil {
		log.Error("pipeline terminó con error", "err", err)
		os.Exit(1)
	}
	log.Info("pipeline completado")
}

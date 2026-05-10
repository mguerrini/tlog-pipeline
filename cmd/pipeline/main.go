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
	"github.com/opessa/tlog-pipeline/internal/steps/create_db"
	"github.com/opessa/tlog-pipeline/internal/steps/create_sql_db"
	"github.com/opessa/tlog-pipeline/internal/steps/create_xml"
	"github.com/opessa/tlog-pipeline/internal/steps/create_xml_sql"
	"github.com/opessa/tlog-pipeline/internal/steps/ftp_download"
	"github.com/opessa/tlog-pipeline/internal/steps/ftp_end"
	"github.com/opessa/tlog-pipeline/internal/steps/ftp_upload"
	"github.com/opessa/tlog-pipeline/internal/steps/local_clean"
	"github.com/opessa/tlog-pipeline/internal/steps/read_files"
	"github.com/opessa/tlog-pipeline/internal/steps/split_by_date"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

func main() {
	flags := config.ParseFlags()

	// Cargar config
	cfg, err := config.Load(flags.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error cargando config: %v\n", err)
		os.Exit(1)
	}
	flags.Apply(cfg)

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "config inválida: %v\n", err)
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

	// Orden de steps según arquitectura
	steps := []pipeline.Step{
		ftp_download.Step{},
		split_by_date.Step{},
		read_files.Step{},
		create_db.Step{},
		create_sql_db.Step{},
		create_xml_sql.Step{},
		create_xml.Step{},
		ftp_upload.Step{},
		ftp_end.Step{},
		local_clean.Step{},
	}

	coord := pipeline.NewCoordinator(cfg, steps, onlyDay, flags.Step, log)
	ctx := context.Background()

	if err := coord.Run(ctx); err != nil {
		log.Error("pipeline terminó con error", "err", err)
		os.Exit(1)
	}
	log.Info("pipeline completado")
}

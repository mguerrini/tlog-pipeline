// Package pipeline define las interfaces y estructuras centrales del pipeline.
package pipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/opessa/tlog-pipeline/internal/config"
	"github.com/opessa/tlog-pipeline/internal/db"
)

// StepStatus enumera los posibles estados de un step.
type StepStatus string

const (
	StatusOK      StepStatus = "ok"
	StatusSkipped StepStatus = "skipped"
	StatusFailed  StepStatus = "failed"
)

// StepMeta es metadata libre que cada step puede adjuntar al resultado.
type StepMeta map[string]any

// StepResult encapsula el resultado de ejecutar un step.
type StepResult struct {
	Status     StepStatus
	StartedAt  time.Time
	FinishedAt time.Time
	Meta       StepMeta
	Err        error
}

// DayCtx es el contexto de un día en ejecución: config + día + store + logger.
type DayCtx struct {
	Cfg    *config.Config
	Day    time.Time
	DayDir string // source_root/AAAAMMDD
	OutDir string // target_root/AAAAMMDD
	Store  *db.Store
	Log    *slog.Logger
}

// Step es la interfaz que implementa cada paso del pipeline.
type Step interface {
	Name() string
	Run(ctx context.Context, d *DayCtx) *StepResult
}

package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Runner ejecuta la secuencia de steps para un único día.
type Runner struct {
	GlobalSteps []Step // steps globales del día
	log         *slog.Logger
}

// NewRunner construye un Runner con los steps en el orden de ejecución.
func NewRunner(steps []Step, log *slog.Logger) *Runner {
	return &Runner{GlobalSteps: steps, log: log}
}

// RunDay procesa un día completo y actualiza el DayStatus.
func (r *Runner) RunDay(ctx context.Context, d *DayCtx, onlyStep string) error {
	dayStr := d.Day.Format("20060102")
	status := newDayStatus(d.Day)
	statusPath := filepath.Join(d.OutDir, dayStr+"_day_status.json")

	if err := os.MkdirAll(d.OutDir, 0o755); err != nil {
		return fmt.Errorf("crear out_dir: %w", err)
	}

	for _, step := range r.GlobalSteps {
		if onlyStep != "" && step.Name() != onlyStep {
			continue
		}
		d.Log.Info("iniciando step", "step", step.Name(), "day", dayStr)
		result := step.Run(ctx, d)
		status.setStep(step.Name(), result)

		dur := result.FinishedAt.Sub(result.StartedAt).Round(time.Millisecond)
		if result.Status == StatusFailed {
			d.Log.Error("step fallido", "step", step.Name(), "err", result.Err, "dur", dur)
			_ = status.save(statusPath)
			return fmt.Errorf("step %s fallido: %w", step.Name(), result.Err)
		}
		d.Log.Info("step completado",
			"step", step.Name(), "status", result.Status, "dur", dur)

		// Modo debug: el step pidió detener el pipeline aquí.
		if result.StopAfter {
			d.Log.Info("pipeline detenido por StopAfter", "step", step.Name())
			_ = status.save(statusPath)
			return nil
		}
	}

	_ = status.save(statusPath)
	return nil
}

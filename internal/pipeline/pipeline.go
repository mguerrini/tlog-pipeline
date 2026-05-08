package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/opessa/tlog-pipeline/internal/config"
	"github.com/opessa/tlog-pipeline/internal/logger"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

var reDay = regexp.MustCompile(`^\d{8}$`)

// Coordinator orquesta el pipeline completo: múltiples días, múltiples retails.
type Coordinator struct {
	cfg      *config.Config
	steps    []Step
	onlyDay  time.Time // zero = todos los días
	onlyStep string
	log      *slog.Logger
}

// NewCoordinator construye el Coordinator.
func NewCoordinator(cfg *config.Config, steps []Step, onlyDay time.Time, onlyStep string, log *slog.Logger) *Coordinator {
	return &Coordinator{
		cfg:      cfg,
		steps:    steps,
		onlyDay:  onlyDay,
		onlyStep: onlyStep,
		log:      log,
	}
}

// Run ejecuta el pipeline.
func (c *Coordinator) Run(ctx context.Context) error {
	// Determinar días a procesar
	var days []time.Time
	if !c.onlyDay.IsZero() {
		days = []time.Time{c.onlyDay}
	} else {
		var err error
		days, err = findDays(c.cfg.LocalFolders.SourceRoot)
		if err != nil {
			return fmt.Errorf("read_days: %w", err)
		}
	}

	if len(days) == 0 {
		c.log.Info("no hay días para procesar", "source_root", c.cfg.LocalFolders.SourceRoot)
		return nil
	}

	c.log.Info("días a procesar", "count", len(days))

	if len(days) > 1 && c.cfg.Process.ExecutionMode == "PARALLEL" && !c.cfg.Process.ParallelRetailsPerDay {
		return c.runParallel(ctx, days)
	}
	return c.runSerial(ctx, days)
}

func (c *Coordinator) runSerial(ctx context.Context, days []time.Time) error {
	for _, day := range days {
		if err := c.processDay(ctx, day); err != nil {
			c.log.Error("día fallido", "day", timeutil.FormatCompact(day), "err", err)
			// No abortar: continuar con los siguientes días
		}
	}
	return nil
}

func (c *Coordinator) runParallel(ctx context.Context, days []time.Time) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, max(1, numCPU()))
	for _, day := range days {
		day := day
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := c.processDay(ctx, day); err != nil {
				c.log.Error("día fallido", "day", timeutil.FormatCompact(day), "err", err)
			}
		}()
	}
	wg.Wait()
	return nil
}

func (c *Coordinator) processDay(ctx context.Context, day time.Time) error {
	dayStr := timeutil.FormatCompact(day)
	dayDir := filepath.Join(c.cfg.LocalFolders.SourceRoot, dayStr)
	outDir := filepath.Join(c.cfg.LocalFolders.TargetRoot, dayStr)

	if _, err := os.Stat(dayDir); os.IsNotExist(err) {
		c.log.Warn("carpeta del día no existe, skip", "dir", dayDir)
		return nil
	}

	// Logger con archivo por día (si está habilitado en config)
	_ = os.MkdirAll(outDir, 0o755)
	logPath := ""
	if c.cfg.Logs.PipelineEnabled {
		logPath = filepath.Join(outDir, dayStr+"_pipeline.log")
	}
	dayLog, closer, err := logger.New(slog.LevelInfo, logPath)
	if err != nil {
		dayLog = c.log // fallback al log global
	}
	defer closer.Close()

	dayLog = dayLog.With("day", dayStr)

	d := &DayCtx{
		Cfg:    c.cfg,
		Day:    day,
		DayDir: dayDir,
		OutDir: outDir,
		Log:    dayLog,
	}

	runner := NewRunner(c.steps, dayLog)
	return runner.RunDay(ctx, d, c.onlyStep)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func numCPU() int {
	return 4
}

// findDays escanea root buscando sub-carpetas con nombre AAAAMMDD.
func findDays(root string) ([]time.Time, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("leer source_root %s: %w", root, err)
	}
	var days []time.Time
	for _, e := range entries {
		if !e.IsDir() || !reDay.MatchString(e.Name()) {
			continue
		}
		t, err := timeutil.ParseDay(e.Name())
		if err != nil {
			continue
		}
		sub := filepath.Join(root, e.Name())
		subs, err := os.ReadDir(sub)
		if err != nil || len(subs) == 0 {
			continue
		}
		days = append(days, t)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days, nil
}

package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/opessa/tlog-pipeline/internal/config"
	"github.com/opessa/tlog-pipeline/internal/logger"
	"github.com/opessa/tlog-pipeline/internal/steps/ftp_download"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

var reDay = regexp.MustCompile(`^\d{8}$`)

// Coordinator orquesta el pipeline en dos fases:
//   - Fase 1 (download + procesamiento): ftp_download global + steps per-day
//     desde split_by_date hasta create_xml. Genera target_root/AAAAMMDD/.
//   - Fase 2 (upload + cierre): ftp_upload, ftp_end, local_clean per-day.
//     SIEMPRE corre, incluso si la fase 1 no produjo nuevos días, para retomar
//     trabajo pendiente (días que ya tienen XMLs pero no se subieron o no se
//     limpiaron — el archivo ftp_status.json en cada target_root/AAAAMMDD/
//     dice qué falta hacer).
type Coordinator struct {
	cfg         *config.Config
	phase1Steps []Step
	phase2Steps []Step
	onlyDay     time.Time // zero = todos los días
	onlyStep    string
	log         *slog.Logger
}

func NewCoordinator(cfg *config.Config, phase1Steps, phase2Steps []Step, onlyDay time.Time, onlyStep string, log *slog.Logger) *Coordinator {
	return &Coordinator{
		cfg:         cfg,
		phase1Steps: phase1Steps,
		phase2Steps: phase2Steps,
		onlyDay:     onlyDay,
		onlyStep:    onlyStep,
		log:         log,
	}
}

func (c *Coordinator) Run(ctx context.Context) error {
	if c.shouldRunPhase1() {
		if err := c.runPhase1(ctx); err != nil {
			return err
		}
		if c.onlyStep == "ftp_download" {
			return nil
		}
	}

	if c.shouldRunPhase2() {
		if err := c.runPhase2(ctx); err != nil {
			return err
		}
	}
	return nil
}

// shouldRunPhase1 decide si entrar a la fase 1. Con --step set, solo entra si
// el step es ftp_download o pertenece a phase1Steps.
func (c *Coordinator) shouldRunPhase1() bool {
	if c.onlyStep == "" {
		return true
	}
	if c.onlyStep == "ftp_download" {
		return true
	}
	return stepInList(c.onlyStep, c.phase1Steps)
}

// shouldRunPhase2 decide si entrar a la fase 2.
func (c *Coordinator) shouldRunPhase2() bool {
	if c.onlyStep == "" {
		return true
	}
	return stepInList(c.onlyStep, c.phase2Steps)
}

func stepInList(name string, steps []Step) bool {
	for _, s := range steps {
		if s.Name() == name {
			return true
		}
	}
	return false
}

// runPhase1 ejecuta ftp_download (una vez) y después corre los steps per-day
// de la fase 1 sobre los días disponibles en source_root/all.
func (c *Coordinator) runPhase1(ctx context.Context) error {
	if c.onlyStep == "" || c.onlyStep == "ftp_download" {
		if err := ftp_download.Download(ctx, c.cfg, c.log); err != nil {
			return fmt.Errorf("ftp_download: %w", err)
		}
		if c.onlyStep == "ftp_download" {
			c.log.Info("--step ftp_download: completado")
			return nil
		}
	}
	days, err := c.findPhase1Days()
	if err != nil {
		return fmt.Errorf("phase1: discover days: %w", err)
	}
	return c.runDays(ctx, "phase1", days, c.phase1Steps, true)
}

// runPhase2 corre upload + ftp_end + local_clean sobre los días que existen
// bajo target_root. Esta fase no depende del source_root, así que corre
// incluso si la fase 1 no descubrió nada nuevo.
func (c *Coordinator) runPhase2(ctx context.Context) error {
	days, err := c.findPhase2Days()
	if err != nil {
		return fmt.Errorf("phase2: discover days: %w", err)
	}
	return c.runDays(ctx, "phase2", days, c.phase2Steps, false)
}

func (c *Coordinator) findPhase1Days() ([]time.Time, error) {
	if !c.onlyDay.IsZero() {
		return []time.Time{c.onlyDay}, nil
	}
	allDir := ""
	if c.cfg.SplitByDate.Enabled {
		allDir = c.cfg.LocalFolders.All
	}
	return findDays(c.cfg.LocalFolders.SourceRoot, allDir)
}

func (c *Coordinator) findPhase2Days() ([]time.Time, error) {
	if !c.onlyDay.IsZero() {
		return []time.Time{c.onlyDay}, nil
	}
	return findTargetDays(c.cfg.LocalFolders.TargetRoot)
}

// runDays ejecuta steps sobre la lista de días, en serial o paralelo según
// config. requireSource determina si se exige que source_root/AAAAMMDD exista
// para procesar el día (true en fase 1, false en fase 2 — esta última opera
// sobre target_root y no necesita el source).
func (c *Coordinator) runDays(ctx context.Context, phase string, days []time.Time, steps []Step, requireSource bool) error {
	if len(days) == 0 {
		c.log.Info("no hay días para procesar", "phase", phase)
		return nil
	}
	c.log.Info("días a procesar", "phase", phase, "count", len(days))

	if len(days) > 1 && c.cfg.Process.ExecutionMode == "PARALLEL" && !c.cfg.Process.ParallelRetailsPerDay {
		return c.runParallel(ctx, days, steps, requireSource)
	}
	return c.runSerial(ctx, days, steps, requireSource)
}

func (c *Coordinator) runSerial(ctx context.Context, days []time.Time, steps []Step, requireSource bool) error {
	for _, day := range days {
		if err := c.processDay(ctx, day, steps, requireSource); err != nil {
			c.log.Error("día fallido", "day", timeutil.FormatCompact(day), "err", err)
		}
	}
	return nil
}

func (c *Coordinator) runParallel(ctx context.Context, days []time.Time, steps []Step, requireSource bool) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, max(1, numCPU()))
	for _, day := range days {
		day := day
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := c.processDay(ctx, day, steps, requireSource); err != nil {
				c.log.Error("día fallido", "day", timeutil.FormatCompact(day), "err", err)
			}
		}()
	}
	wg.Wait()
	return nil
}

func (c *Coordinator) processDay(ctx context.Context, day time.Time, steps []Step, requireSource bool) error {
	dayStr := timeutil.FormatCompact(day)
	dayDir := filepath.Join(c.cfg.LocalFolders.SourceRoot, dayStr)
	outDir := filepath.Join(c.cfg.LocalFolders.TargetRoot, dayStr)

	if requireSource {
		// En fase 1 la carpeta del día tiene que existir o el split_by_date
		// la creará — sin eso no hay nada que procesar.
		if _, err := os.Stat(dayDir); os.IsNotExist(err) {
			if !c.cfg.SplitByDate.Enabled {
				c.log.Warn("carpeta del día no existe, skip", "dir", dayDir)
				return nil
			}
		}
	}

	_ = os.MkdirAll(outDir, 0o755)
	logPath := ""
	if c.cfg.Logs.PipelineEnabled {
		logPath = filepath.Join(outDir, dayStr+"_pipeline.log")
	}
	dayLog, closer, err := logger.New(slog.LevelInfo, logPath)
	if err != nil {
		dayLog = c.log
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

	runner := NewRunner(steps, dayLog)
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

// findDays descubre los días a procesar combinando dos fuentes:
//   - sub-carpetas AAAAMMDD bajo sourceRoot (input/AAAAMMDD/)
//   - últimos 8 caracteres del basename de los CSVs en allDir (si != "")
//
// La segunda fuente existe para que split_by_date tenga días que procesar
// cuando todavía no se creó la carpeta input/AAAAMMDD/ — ese step es
// justamente quien la crea. Si allDir == "", se usa solo la primera.
func findDays(sourceRoot, allDir string) ([]time.Time, error) {
	seen := map[string]time.Time{}

	entries, err := os.ReadDir(sourceRoot)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("leer source_root %s: %w", sourceRoot, err)
	}
	for _, e := range entries {
		if !e.IsDir() || !reDay.MatchString(e.Name()) {
			continue
		}
		t, err := timeutil.ParseDay(e.Name())
		if err != nil {
			continue
		}
		sub := filepath.Join(sourceRoot, e.Name())
		subs, err := os.ReadDir(sub)
		if err != nil || len(subs) == 0 {
			continue
		}
		seen[e.Name()] = t
	}

	if allDir != "" {
		if files, err := os.ReadDir(allDir); err == nil {
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				name := f.Name()
				ext := filepath.Ext(name)
				if !strings.EqualFold(ext, ".csv") {
					continue
				}
				base := strings.TrimSuffix(name, ext)
				if len(base) < 8 {
					continue
				}
				date := base[len(base)-8:]
				if !reDay.MatchString(date) {
					continue
				}
				if _, dup := seen[date]; dup {
					continue
				}
				if t, err := timeutil.ParseDay(date); err == nil {
					seen[date] = t
				}
			}
		}
	}

	days := make([]time.Time, 0, len(seen))
	for _, t := range seen {
		days = append(days, t)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days, nil
}

// findTargetDays lista las subcarpetas AAAAMMDD bajo targetRoot. Es la fuente
// de días para la fase 2 — son los días que ya tienen XMLs generados y por lo
// tanto están en condiciones de ser subidos / archivados / limpiados.
func findTargetDays(targetRoot string) ([]time.Time, error) {
	entries, err := os.ReadDir(targetRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("leer target_root %s: %w", targetRoot, err)
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
		days = append(days, t)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days, nil
}

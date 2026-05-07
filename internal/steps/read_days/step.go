package read_days

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

var reDay = regexp.MustCompile(`^\d{8}$`)

// Step implementa el step "read_days": enumera carpetas AAAAMMDD en source_root.
type Step struct{}

func (Step) Name() string { return "read_days" }

// Run no necesita DayCtx completo; se usa solo en el Coordinator.
// Expuesto como función separada para reuso.
func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	r := pipeline.NewResult()
	days, err := FindDays(d.Cfg.LocalFolders.SourceRoot)
	if err != nil {
		return r.Fail(err)
	}
	r.SetMeta("days_found", len(days))
	return r.OK()
}

// FindDays escanea root buscando sub-carpetas con nombre AAAAMMDD.
func FindDays(root string) ([]time.Time, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("leer source_root %s: %w", root, err)
	}
	var days []time.Time
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !reDay.MatchString(e.Name()) {
			continue
		}
		t, err := timeutil.ParseDay(e.Name())
		if err != nil {
			continue
		}
		// Verificar que la carpeta no esté vacía
		subpath := filepath.Join(root, e.Name())
		subs, err := os.ReadDir(subpath)
		if err != nil || len(subs) == 0 {
			continue
		}
		days = append(days, t)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days, nil
}

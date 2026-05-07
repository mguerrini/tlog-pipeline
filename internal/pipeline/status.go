package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DayStatus es el modelo del archivo AAAAMMDD_day_status.json.
type DayStatus struct {
	Day           string                `json:"day"`
	OverallStatus string                `json:"overall_status"` // ok | failed | completed_with_errors
	GlobalSteps   []*StepJSON           `json:"global_steps"`
	Retails       map[string]*RetailSt  `json:"retails,omitempty"`
}

// StepJSON serializa un StepResult al JSON de estado.
type StepJSON struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	StartedAt  string   `json:"started_at,omitempty"`
	FinishedAt string   `json:"finished_at,omitempty"`
	Meta       StepMeta `json:"meta,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// RetailSt agrupa los steps por retail ID.
type RetailSt struct {
	Steps map[string]*StepJSON `json:"steps"`
	Status string              `json:"status"`
}

func newDayStatus(day time.Time) *DayStatus {
	return &DayStatus{
		Day:           day.Format("20060102"),
		OverallStatus: "ok",
		GlobalSteps:   make([]*StepJSON, 0),
		Retails:       make(map[string]*RetailSt),
	}
}

func (ds *DayStatus) setStep(name string, r *StepResult) {
	sj := &StepJSON{
		Name:       name,
		Status:     string(r.Status),
		StartedAt:  r.StartedAt.Format(time.RFC3339),
		FinishedAt: r.FinishedAt.Format(time.RFC3339),
		Meta:       r.Meta,
	}
	if r.Err != nil {
		sj.Error = r.Err.Error()
	}
	for i, existing := range ds.GlobalSteps {
		if existing.Name == name {
			ds.GlobalSteps[i] = sj
			if r.Status == StatusFailed {
				ds.OverallStatus = "failed"
			}
			return
		}
	}
	ds.GlobalSteps = append(ds.GlobalSteps, sj)
	if r.Status == StatusFailed {
		ds.OverallStatus = "failed"
	}
}

func (ds *DayStatus) save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

package ftp_end

import (
	"context"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
)

type Step struct{}

func (Step) Name() string { return "ftp_end" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()
	if !d.Cfg.FTPEnd.Enabled {
		return b.Skip("disabled in config")
	}
	d.Log.Info("ftp_end: habilitado pero sin URL configurada, skip")
	return b.Skip("ftp_end no configurado")
}

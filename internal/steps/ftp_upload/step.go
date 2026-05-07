package ftp_upload

import (
	"context"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
)

type Step struct{}

func (Step) Name() string { return "ftp_upload" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()
	if !d.Cfg.FTPUpload.Enabled {
		return b.Skip("disabled in config")
	}
	d.Log.Info("ftp_upload: habilitado pero sin URL configurada, skip")
	return b.Skip("ftp_upload no configurado")
}

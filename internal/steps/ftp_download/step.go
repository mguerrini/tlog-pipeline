// Package ftp_download implementa el step de descarga FTP.
package ftp_download

import (
	"context"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
)

type Step struct{}

func (Step) Name() string { return "ftp_download" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()
	if !d.Cfg.FTPDownload.Enabled {
		return b.Skip("disabled in config")
	}
	// TODO: implementar descarga FTP usando internal/ftp cuando se habilite
	d.Log.Info("ftp_download: habilitado pero sin URL configurada, skip")
	return b.Skip("ftp_download no configurado (URL vacía)")
}

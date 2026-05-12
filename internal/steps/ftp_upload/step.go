// Package ftp_upload implementa la subida SFTP de un día como pipeline.Step.
//
// Es un step de la fase 2: corre per-day sobre target_root/AAAAMMDD y sube esa
// estructura a cfg.FtpFolders.TargetRoot/AAAAMMDD en el server cfg.FTPTarget.
// Usa el archivo ftp_status.json en target_root/AAAAMMDD para no repetir subida
// si ya estaba marcada como hecha. Si el step está deshabilitado, no genera
// ftp_status — los steps posteriores (ftp_end) verán Uploaded=false y se
// saltearán solos.
package ftp_upload

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

type Step struct{}

func (Step) Name() string { return "ftp_upload" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()

	// Step deshabilitado: no leemos ni generamos ftp_status — sin subida no hay
	// nada que trackear. ftp_end leerá Uploaded=false y se salteará solo.
	if !d.Cfg.FTPUpload.Enabled {
		return b.SetMeta("reason", "disabled in config").Skip("disabled in config")
	}

	status, err := pipeline.LoadFtpStatus(d.OutDir)
	if err != nil {
		return b.Fail(fmt.Errorf("leer ftp_status: %w", err))
	}
	if status.Uploaded {
		return b.SetMeta("reason", "already uploaded").Skip("already uploaded")
	}

	dayStr := timeutil.FormatCompact(d.Day)
	status.Day = dayStr

	tgt := d.Cfg.FTPTarget
	if tgt.URL == "" {
		return b.Fail(fmt.Errorf("ftp_target.url vacío"))
	}
	remoteRoot := d.Cfg.FtpFolders.TargetRoot
	if remoteRoot == "" {
		return b.Fail(fmt.Errorf("ftp_folders.target_root vacío"))
	}
	port := tgt.Port
	if port == 0 {
		port = 22
	}
	remoteDayDir := path.Join(remoteRoot, dayStr)
	addr := net.JoinHostPort(tgt.URL, strconv.Itoa(port))

	d.Log.Info("ftp_upload: conectando", "addr", addr, "user", tgt.Username, "local", d.OutDir, "remote", remoteDayDir)

	sshCfg := &ssh.ClientConfig{
		User:            tgt.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(tgt.Pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
	sshConn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return b.Fail(fmt.Errorf("conectar SSH %s: %w", addr, err))
	}
	defer sshConn.Close()

	client, err := sftp.NewClient(sshConn)
	if err != nil {
		return b.Fail(fmt.Errorf("abrir SFTP: %w", err))
	}
	defer client.Close()

	if err := client.MkdirAll(remoteDayDir); err != nil {
		return b.Fail(fmt.Errorf("crear remoto %s: %w", remoteDayDir, err))
	}

	uploaded, totalBytes, err := uploadDir(ctx, client, d.OutDir, remoteDayDir, d.Log)
	if err != nil {
		return b.Fail(err)
	}

	status.MarkUploaded(remoteDayDir)
	if err := status.Save(d.OutDir); err != nil {
		return b.Fail(fmt.Errorf("guardar ftp_status: %w", err))
	}

	d.Log.Info("ftp_upload ok", "uploaded", uploaded, "bytes", totalBytes, "remote", remoteDayDir)
	b.SetMeta("uploaded", uploaded).SetMeta("bytes", totalBytes).SetMeta("remote", remoteDayDir)
	return b.OK()
}

// uploadDir sube recursivamente los .xml de localDir a remoteDir. El resto de
// los archivos (ftp_status.json, .db, .log, .md, etc.) son estado local o
// artefactos intermedios — no forman parte del entregable.
func uploadDir(ctx context.Context, client *sftp.Client, localDir, remoteDir string, log *slog.Logger) (int, int64, error) {
	var uploaded int
	var totalBytes int64
	err := filepath.Walk(localDir, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(localDir, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		remotePath := path.Join(remoteDir, filepath.ToSlash(rel))
		if fi.IsDir() {
			if err := client.MkdirAll(remotePath); err != nil {
				return fmt.Errorf("mkdir remoto %s: %w", remotePath, err)
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".xml") {
			return nil
		}
		n, err := uploadFile(client, p, remotePath)
		if err != nil {
			return fmt.Errorf("subir %s: %w", p, err)
		}
		uploaded++
		totalBytes += n
		log.Info("ftp_upload: archivo", "local", p, "remote", remotePath, "bytes", n)
		return nil
	})
	return uploaded, totalBytes, err
}

// uploadFile copia un archivo local a la ruta remota usando un .part temporal
// y rename atómico, para no dejar parciales si la copia se interrumpe.
func uploadFile(c *sftp.Client, localPath, remotePath string) (int64, error) {
	lf, err := os.Open(localPath)
	if err != nil {
		return 0, fmt.Errorf("abrir local: %w", err)
	}
	defer lf.Close()

	tmpPath := remotePath + ".part"
	_ = c.Remove(tmpPath)
	rf, err := c.Create(tmpPath)
	if err != nil {
		return 0, fmt.Errorf("crear remoto: %w", err)
	}

	n, copyErr := io.Copy(rf, lf)
	closeErr := rf.Close()
	if copyErr != nil {
		_ = c.Remove(tmpPath)
		return n, copyErr
	}
	if closeErr != nil {
		_ = c.Remove(tmpPath)
		return n, closeErr
	}
	_ = c.Remove(remotePath)
	if err := c.Rename(tmpPath, remotePath); err != nil {
		_ = c.Remove(tmpPath)
		return n, fmt.Errorf("rename: %w", err)
	}
	return n, nil
}

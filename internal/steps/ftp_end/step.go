// Package ftp_end archiva los CSV del día en el SFTP de origen, moviéndolos de
// la carpeta de input remota a una carpeta de "uploaded" remota — para que no
// vuelvan a descargarse en la próxima corrida del pipeline.
//
// Es un step de la fase 2. Lee target_root/AAAAMMDD/ftp_status.json: sólo
// trabaja si uploaded=true (no tiene sentido archivar sin subir) y aún no se
// marcó source_deleted. Si el step está deshabilitado, marca source_deleted=true
// sin tocar nada remoto, para que local_clean pueda seguir.
package ftp_end

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

type Step struct{}

func (Step) Name() string { return "ftp_end" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()

	status, err := pipeline.LoadFtpStatus(d.OutDir)
	if err != nil {
		return b.Fail(fmt.Errorf("leer ftp_status: %w", err))
	}
	if !status.Uploaded {
		return b.Skip("ftp_upload no completado")
	}
	if status.SourceDeleted {
		return b.SetMeta("reason", "already done").Skip("source ya archivado")
	}

	if !d.Cfg.FTPEnd.Enabled {
		status.MarkSourceDeleted()
		if err := status.Save(d.OutDir); err != nil {
			return b.Fail(fmt.Errorf("guardar ftp_status: %w", err))
		}
		return b.SetMeta("reason", "disabled — marked source_deleted").Skip("disabled in config")
	}

	src := d.Cfg.FtpFolders.SourceRoot
	dst := d.Cfg.FtpFolders.FinishedRoot
	if src == "" || dst == "" {
		return b.Fail(fmt.Errorf("ftp_end: ftp_folders.source_root o ftp_folders.finished_root vacíos"))
	}

	srv := d.Cfg.FTPSource
	if srv.URL == "" {
		return b.Fail(fmt.Errorf("ftp_source.url vacío"))
	}
	port := srv.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(srv.URL, strconv.Itoa(port))
	dayStr := timeutil.FormatCompact(d.Day)
	dstDayDir := path.Join(dst, dayStr)

	d.Log.Info("ftp_end: conectando", "addr", addr, "src", src, "dst", dstDayDir)

	sshCfg := &ssh.ClientConfig{
		User:            srv.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(srv.Pass)},
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

	if err := client.MkdirAll(dstDayDir); err != nil {
		return b.Fail(fmt.Errorf("crear destino remoto %s: %w", dstDayDir, err))
	}

	entries, err := client.ReadDir(src)
	if err != nil {
		return b.Fail(fmt.Errorf("listar %s: %w", src, err))
	}

	var moved int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return b.Fail(err)
		}
		name := e.Name()
		if !matchesDayCSV(name, dayStr) {
			continue
		}
		from := path.Join(src, name)
		to := path.Join(dstDayDir, name)
		if err := moveRemote(client, from, to); err != nil {
			return b.Fail(fmt.Errorf("mover %s -> %s: %w", from, to, err))
		}
		moved++
		d.Log.Info("ftp_end: movido", "from", from, "to", to)
	}

	status.MarkSourceDeleted()
	if err := status.Save(d.OutDir); err != nil {
		return b.Fail(fmt.Errorf("guardar ftp_status: %w", err))
	}
	d.Log.Info("ftp_end ok", "moved", moved, "dst", dstDayDir)
	b.SetMeta("moved", moved).SetMeta("dst", dstDayDir)

	// delete_local_source: cierra el día en el lado local borrando todo el
	// output. Se hace después de Save porque el status acaba de marcar
	// source_deleted=true y queremos persistirlo antes de remover el archivo.
	if d.Cfg.FTPEnd.DeleteLocalSource {
		if err := os.RemoveAll(d.OutDir); err != nil {
			d.Log.Warn("ftp_end: no se pudo eliminar out_dir local", "dir", d.OutDir, "err", err)
			b.SetMeta("local_deleted", false)
		} else {
			d.Log.Info("ftp_end: out_dir local eliminado", "dir", d.OutDir)
			b.SetMeta("local_deleted", true)
		}
	}
	return b.OK()
}

// matchesDayCSV es true si name es un .csv cuyo basename termina en AAAAMMDD —
// el patrón que usa todo el pipeline para identificar archivos del día.
func matchesDayCSV(name, dayStr string) bool {
	const ext = ".csv"
	if len(name) < len(ext) || !strings.EqualFold(name[len(name)-len(ext):], ext) {
		return false
	}
	base := name[:len(name)-len(ext)]
	if len(base) < len(dayStr) {
		return false
	}
	return strings.HasSuffix(base, dayStr)
}

// moveRemote intenta rename SFTP (atómico, mismo volumen); si el server lo
// rechaza — algunos no permiten rename con destino existente o entre paths —
// cae a copy+delete.
func moveRemote(c *sftp.Client, from, to string) error {
	_ = c.Remove(to)
	if err := c.Rename(from, to); err == nil {
		return nil
	}
	in, err := c.Open(from)
	if err != nil {
		return fmt.Errorf("abrir %s: %w", from, err)
	}
	defer in.Close()
	out, err := c.Create(to)
	if err != nil {
		return fmt.Errorf("crear %s: %w", to, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = c.Remove(to)
		return fmt.Errorf("copiar: %w", err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	return c.Remove(from)
}


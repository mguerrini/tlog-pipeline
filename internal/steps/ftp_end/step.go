// Package ftp_end archiva los CSV del día en el SFTP de origen, moviéndolos de
// la carpeta de input remota a una carpeta de "uploaded" remota — para que no
// vuelvan a descargarse en la próxima corrida del pipeline.
//
// Es un step de la fase 2. Lee target_root/AAAAMMDD/ftp_status.json: sólo
// trabaja si uploaded=true (no tiene sentido archivar sin subir) y aún no se
// marcó source_deleted. Si el step está deshabilitado, no hace nada — no toca
// el ftp_status ni los archivos remotos.
package ftp_end

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
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

	if !d.Cfg.FTPEnd.Enabled {
		return b.SetMeta("reason", "disabled in config").Skip("disabled in config")
	}

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

	srcDayDir, err := findSourceDayDir(client, src, d.Day)
	if err != nil {
		return b.Fail(err)
	}
	if srcDayDir == "" {
		d.Log.Info("ftp_end: carpeta origen del día no existe, nada para mover", "src", src, "day", dayStr)
		status.MarkSourceDeleted()
		if err := status.Save(d.OutDir); err != nil {
			return b.Fail(fmt.Errorf("guardar ftp_status: %w", err))
		}
		return b.SetMeta("reason", "source day folder not found").Skip("nada para archivar")
	}

	dayEntries, err := client.ReadDir(srcDayDir)
	if err != nil {
		return b.Fail(fmt.Errorf("listar %s: %w", srcDayDir, err))
	}

	var moved int
	for _, e := range dayEntries {
		if e.IsDir() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return b.Fail(err)
		}
		from := path.Join(srcDayDir, e.Name())
		to := path.Join(dstDayDir, e.Name())
		if err := moveRemote(client, from, to); err != nil {
			return b.Fail(fmt.Errorf("mover %s -> %s: %w", from, to, err))
		}
		moved++
		d.Log.Info("ftp_end: movido", "from", from, "to", to)
	}

	if err := client.RemoveDirectory(srcDayDir); err != nil {
		d.Log.Warn("ftp_end: no se pudo eliminar carpeta origen", "dir", srcDayDir, "err", err)
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

// findSourceDayDir busca dentro de src la subcarpeta cuyo nombre representa
// day, aceptando tanto AAAAMMDD como AAAA-MM-DD (los dos formatos en los que
// ftp_download crea subcarpetas remotas). Devuelve el path remoto absoluto, o
// "" si no existe ninguna que matchee — caso en el que no hay nada para
// archivar.
func findSourceDayDir(c *sftp.Client, src string, day time.Time) (string, error) {
	entries, err := c.ReadDir(src)
	if err != nil {
		return "", fmt.Errorf("listar %s: %w", src, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, err := timeutil.ParseDay(e.Name())
		if err != nil {
			continue
		}
		if t.Equal(day) {
			return path.Join(src, e.Name()), nil
		}
	}
	return "", nil
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

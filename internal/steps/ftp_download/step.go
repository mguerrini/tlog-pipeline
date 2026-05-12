// Package ftp_download implementa la descarga SFTP global del pipeline.
//
// Es una operación pre-pipeline: corre una sola vez antes de que el
// Coordinator busque los días disponibles, porque justamente se encarga de
// traer los archivos que después findDays/split_by_date descubrirán. Por eso
// NO implementa pipeline.Step (que es por-día); se expone como una función
// `Download` que main.go invoca directamente.
//
// Se conecta al servidor definido en cfg.FTPSource (host:port + usuario/pass)
// y descarga el contenido de cfg.FtpFolders.SourceRoot. El flag
// ftp_download.split_by_date elige la modalidad:
//   - true → archivos sueltos en SourceRoot, copiados planos a LocalFolders.All
//     (split_by_date local los reparte por día después).
//   - false → SourceRoot contiene subcarpetas con nombre de fecha
//     (AAAAMMDD o AAAA-MM-DD); cada una se baja a
//     LocalFolders.SourceRoot/AAAAMMDD/. AAAA-MM-DD se normaliza a AAAAMMDD.
package ftp_download

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

	"github.com/opessa/tlog-pipeline/internal/config"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

// Download baja vía SFTP el contenido de ftp_folders.source_root al destino
// local correspondiente (ver doc del package). Devuelve nil con un log "skip"
// si la sección está deshabilitada o no hay configuración suficiente; sólo
// retorna error ante fallas reales (conexión, listado, escritura).
func Download(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	if !cfg.FTPDownload.Enabled {
		log.Info("ftp_download: skip", "reason", "disabled in config")
		return nil
	}

	src := cfg.FTPSource
	if src.URL == "" {
		log.Info("ftp_download: skip", "reason", "ftp_source.url vacío")
		return nil
	}
	port := src.Port
	if port == 0 {
		port = 22
	}

	remoteDir := cfg.FtpFolders.SourceRoot
	if remoteDir == "" {
		log.Info("ftp_download: skip", "reason", "ftp_folders.source_root vacío")
		return nil
	}

	addr := net.JoinHostPort(src.URL, strconv.Itoa(port))
	log.Info("ftp_download: conectando",
		"addr", addr, "user", src.Username, "remote", remoteDir,
		"split_by_date", cfg.FTPDownload.SplitByDate)

	sshCfg := &ssh.ClientConfig{
		User:            src.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(src.Pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	sshConn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return fmt.Errorf("conectar SSH %s: %w", addr, err)
	}
	defer sshConn.Close()

	client, err := sftp.NewClient(sshConn)
	if err != nil {
		return fmt.Errorf("abrir SFTP: %w", err)
	}
	defer client.Close()

	// Resolver el path remoto sin distinción de mayúsculas para tolerar
	// diferencias entre lo configurado y la capitalización real del server.
	resolved, err := resolveCaseInsensitive(client, remoteDir)
	if err != nil {
		return fmt.Errorf("resolver remote %q: %w", remoteDir, err)
	}
	if resolved != remoteDir {
		log.Info("ftp_download: remoto resuelto", "configurado", remoteDir, "real", resolved)
	}

	if cfg.FTPDownload.SplitByDate {
		return downloadByDate(ctx, client, resolved, cfg.LocalFolders.SourceRoot, log)
	}
	return downloadFlat(ctx, client, resolved, cfg.LocalFolders.All, log)
}

// resolveCaseInsensitive resuelve un path SFTP tolerando diferencias de
// mayúsculas. Estrategia:
//  1. Fast path: si Stat sobre el path entero funciona, se devuelve tal cual.
//  2. Bottom-up: se prueban prefijos cada vez más cortos con Stat hasta
//     encontrar el ancestro más profundo que el server reconoce. Muchos SFTP
//     bloquean ReadDir sobre raíces altas (p.ej. "/") pero sí responden Stat
//     en sub-paths, así que partir desde la raíz a listar puede fallar con
//     "permission denied" aunque el path real exista.
//  3. Top-down desde el ancestro: en cada nivel restante se hace ReadDir y se
//     elige la entrada que matchea insensitive con la siguiente componente.
//
// Devuelve el path con la capitalización real del servidor.
func resolveCaseInsensitive(c *sftp.Client, p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path vacío")
	}
	if _, err := c.Stat(p); err == nil {
		return p, nil
	}
	clean := path.Clean(p)
	abs := strings.HasPrefix(clean, "/")
	rawParts := strings.Split(strings.Trim(clean, "/"), "/")
	parts := rawParts[:0]
	for _, x := range rawParts {
		if x != "" && x != "." {
			parts = append(parts, x)
		}
	}
	if len(parts) == 0 {
		if abs {
			return "/", nil
		}
		return ".", nil
	}

	join := func(n int) string {
		s := strings.Join(parts[:n], "/")
		if abs {
			return "/" + s
		}
		if s == "" {
			return "."
		}
		return s
	}

	// Buscar el prefijo más largo que Stat reconoce. Si ninguno funciona caemos
	// a la raíz como último recurso (puede fallar más adelante con permission
	// denied, pero el error queda más claro).
	anchorIdx := 0
	for i := len(parts) - 1; i >= 1; i-- {
		if _, err := c.Stat(join(i)); err == nil {
			anchorIdx = i
			break
		}
	}
	cur := join(anchorIdx)
	if anchorIdx == 0 {
		if abs {
			cur = "/"
		} else {
			cur = "."
		}
	}

	for _, part := range parts[anchorIdx:] {
		entries, err := c.ReadDir(cur)
		if err != nil {
			return "", fmt.Errorf("listar %s: %w", cur, err)
		}
		match := ""
		for _, e := range entries {
			if strings.EqualFold(e.Name(), part) {
				match = e.Name()
				break
			}
		}
		if match == "" {
			return "", fmt.Errorf("componente %q no existe en %s", part, cur)
		}
		switch cur {
		case "/":
			cur = "/" + match
		case ".":
			cur = match
		default:
			cur = path.Join(cur, match)
		}
	}
	return cur, nil
}

// downloadFlat copia los archivos sueltos del nivel superior de remoteDir a
// localDir (sin recurrir en sub-carpetas).
func downloadFlat(ctx context.Context, client *sftp.Client, remoteDir, localDir string, log *slog.Logger) error {
	if localDir == "" {
		return fmt.Errorf("destino local vacío (local_folders.all)")
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("crear destino %s: %w", localDir, err)
	}

	entries, err := client.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("listar %s: %w", remoteDir, err)
	}

	var downloaded int
	var totalBytes int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		remotePath := path.Join(remoteDir, e.Name())
		localPath := filepath.Join(localDir, e.Name())
		n, err := downloadFile(client, remotePath, localPath)
		if err != nil {
			return fmt.Errorf("descargar %s: %w", remotePath, err)
		}
		downloaded++
		totalBytes += n
		log.Info("ftp_download: archivo", "remote", remotePath, "local", localPath, "bytes", n)
	}

	log.Info("ftp_download ok", "downloaded", downloaded, "bytes", totalBytes, "local", localDir)
	return nil
}

// downloadByDate baja cada subcarpeta de remoteDir cuyo nombre sea una fecha
// AAAAMMDD o AAAA-MM-DD a localRoot/AAAAMMDD/. Subcarpetas con otros nombres
// se ignoran; archivos sueltos en el nivel superior también.
func downloadByDate(ctx context.Context, client *sftp.Client, remoteDir, localRoot string, log *slog.Logger) error {
	if localRoot == "" {
		return fmt.Errorf("destino local vacío (local_folders.source_root)")
	}
	if err := os.MkdirAll(localRoot, 0o755); err != nil {
		return fmt.Errorf("crear destino %s: %w", localRoot, err)
	}

	entries, err := client.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("listar %s: %w", remoteDir, err)
	}

	var dayFolders, downloaded int
	var totalBytes int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		day, err := timeutil.ParseDay(e.Name())
		if err != nil {
			log.Info("ftp_download: subcarpeta ignorada (nombre no es fecha)", "name", e.Name())
			continue
		}
		dayStr := timeutil.FormatCompact(day)
		remoteDay := path.Join(remoteDir, e.Name())
		localDay := filepath.Join(localRoot, dayStr)
		if err := os.MkdirAll(localDay, 0o755); err != nil {
			return fmt.Errorf("crear %s: %w", localDay, err)
		}

		dayEntries, err := client.ReadDir(remoteDay)
		if err != nil {
			return fmt.Errorf("listar %s: %w", remoteDay, err)
		}
		for _, df := range dayEntries {
			if df.IsDir() {
				continue
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			remotePath := path.Join(remoteDay, df.Name())
			localPath := filepath.Join(localDay, df.Name())
			n, err := downloadFile(client, remotePath, localPath)
			if err != nil {
				return fmt.Errorf("descargar %s: %w", remotePath, err)
			}
			downloaded++
			totalBytes += n
			log.Info("ftp_download: archivo", "remote", remotePath, "local", localPath, "bytes", n)
		}
		dayFolders++
	}

	log.Info("ftp_download ok",
		"day_folders", dayFolders, "downloaded", downloaded, "bytes", totalBytes,
		"local_root", localRoot)
	return nil
}

// downloadFile copia un archivo remoto a la ruta local indicada. Usa un .part
// temporal y rename atómico para evitar dejar archivos parciales si la copia
// se interrumpe a mitad de camino.
func downloadFile(c *sftp.Client, remotePath, localPath string) (int64, error) {
	rf, err := c.Open(remotePath)
	if err != nil {
		return 0, fmt.Errorf("abrir remoto: %w", err)
	}
	defer rf.Close()

	tmpPath := localPath + ".part"
	lf, err := os.Create(tmpPath)
	if err != nil {
		return 0, fmt.Errorf("crear local: %w", err)
	}

	n, copyErr := io.Copy(lf, rf)
	closeErr := lf.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return n, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return n, closeErr
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		_ = os.Remove(tmpPath)
		return n, fmt.Errorf("rename: %w", err)
	}
	return n, nil
}

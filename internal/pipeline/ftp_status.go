package pipeline

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// FtpStatusFile es el nombre del archivo de estado por día. Vive bajo
// target_root/AAAAMMDD/ y orquesta la fase 2 del pipeline (ftp_upload, ftp_end,
// local_clean): cada step lo lee al entrar para saber si tiene que hacer
// trabajo y lo escribe al salir. Hace falta para reanudar la fase 2 sin
// repetir trabajo y para que steps deshabilitados puedan "marcar como hecho"
// y dejar avanzar al siguiente.
const FtpStatusFile = "ftp_status.json"

// FtpStatus es el contenido serializado del ftp_status.json.
type FtpStatus struct {
	Day             string `json:"day,omitempty"`
	Uploaded        bool   `json:"uploaded"`
	UploadedAt      string `json:"uploaded_at,omitempty"`
	UploadedRemote  string `json:"uploaded_remote,omitempty"`
	SourceDeleted   bool   `json:"source_deleted"`
	SourceDeletedAt string `json:"source_deleted_at,omitempty"`
}

// FtpStatusPath devuelve la ruta absoluta del ftp_status.json para un dayDir.
func FtpStatusPath(dayDir string) string {
	return filepath.Join(dayDir, FtpStatusFile)
}

// LoadFtpStatus lee el ftp_status.json de un dayDir. Si no existe, devuelve un
// FtpStatus vacío sin error: ese es el estado inicial válido (nada subido,
// nada borrado todavía).
func LoadFtpStatus(dayDir string) (*FtpStatus, error) {
	b, err := os.ReadFile(FtpStatusPath(dayDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &FtpStatus{}, nil
		}
		return nil, err
	}
	var s FtpStatus
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save persiste el FtpStatus a target_root/AAAAMMDD/ftp_status.json.
// Crea el directorio si hace falta.
func (s *FtpStatus) Save(dayDir string) error {
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(FtpStatusPath(dayDir), b, 0o644)
}

// MarkUploaded marca Uploaded=true con timestamp UTC. remote es opcional —
// se guarda como referencia de a dónde se subió.
func (s *FtpStatus) MarkUploaded(remote string) {
	s.Uploaded = true
	s.UploadedAt = time.Now().UTC().Format(time.RFC3339)
	if remote != "" {
		s.UploadedRemote = remote
	}
}

// MarkSourceDeleted marca SourceDeleted=true con timestamp UTC.
func (s *FtpStatus) MarkSourceDeleted() {
	s.SourceDeleted = true
	s.SourceDeletedAt = time.Now().UTC().Format(time.RFC3339)
}

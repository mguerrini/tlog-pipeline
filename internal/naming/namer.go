package naming

import (
	"fmt"
	"time"
)

// FileNamer abstrae la convención de nombres de archivos del pipeline.
type FileNamer interface {
	XMLFile(t TLOGType, kstCode, seqNum string) string
	DayStatusFile(day time.Time) string
	RetailStatusFile(day time.Time) string
	DBFile(day time.Time) string
	LogFile(day time.Time) string
	OrphansFile(day time.Time) string
}

// DefaultNamer aplica la convención del proyecto.
type DefaultNamer struct {
	// IncludeDocumentType controla si el nombre del XML incluye el tipo de
	// documento (Reception, Cierre, etc.). Mapea a process.file_name_include_document_type.
	IncludeDocumentType bool
}

// XMLFile devuelve el nombre del XML.
// IncludeDocumentType=true  → TLOG_INVENTORY_<Tipo>_<KstCode>_<SequenceNumber>.xml
// IncludeDocumentType=false → TLOG_INVENTORY_<KstCode>_<SequenceNumber>.xml
func (n DefaultNamer) XMLFile(t TLOGType, kstCode, seqNum string) string {
	if n.IncludeDocumentType {
		return fmt.Sprintf("TLOG_INVENTORY_%s_%s_%s.xml", string(t), kstCode, seqNum)
	}
	return fmt.Sprintf("TLOG_INVENTORY_%s_%s.xml", kstCode, seqNum)
}

func (DefaultNamer) DayStatusFile(day time.Time) string {
	return fmt.Sprintf("%s_day_status.json", day.Format("20060102"))
}

func (DefaultNamer) RetailStatusFile(day time.Time) string {
	return fmt.Sprintf("%s_pipeline_status.json", day.Format("20060102"))
}

func (DefaultNamer) DBFile(day time.Time) string {
	return fmt.Sprintf("%s_pipeline.db", day.Format("20060102"))
}

func (DefaultNamer) LogFile(day time.Time) string {
	return fmt.Sprintf("%s_pipeline.log", day.Format("20060102"))
}

func (DefaultNamer) OrphansFile(day time.Time) string {
	return fmt.Sprintf("%s_orphans.md", day.Format("20060102"))
}

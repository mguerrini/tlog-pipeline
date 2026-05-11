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
type DefaultNamer struct{}

// XMLFile devuelve el nombre del XML: TLOG_<Tipo>_<KstCode>_<SequenceNumber>.xml.
func (DefaultNamer) XMLFile(t TLOGType, kstCode, seqNum string) string {
	return fmt.Sprintf("TLOG_%s_%s_%s.xml", string(t), kstCode, seqNum)
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

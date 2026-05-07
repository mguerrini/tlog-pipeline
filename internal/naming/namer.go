package naming

import (
	"fmt"
	"time"
)

// FileNamer abstrae la convención de nombres de archivos del pipeline.
type FileNamer interface {
	XMLFile(retailID string, day time.Time, t TLOGType) string
	DayStatusFile(day time.Time) string
	RetailStatusFile(day time.Time) string
	DBFile(day time.Time) string
	LogFile(day time.Time) string
	OrphansFile(day time.Time) string
}

// DefaultNamer aplica la convención del proyecto: AAAAMMDD para fecha.
type DefaultNamer struct{}

func (DefaultNamer) XMLFile(retailID string, day time.Time, t TLOGType) string {
	n := indexOf(t)
	return fmt.Sprintf("%s-%s-%s-%04d.xml", retailID, day.Format("20060102"), originalNameOf(t), n)
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

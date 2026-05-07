package reception

import (
	"github.com/opessa/tlog-pipeline/internal/db"
)

// header agrupa los datos resueltos para una cabecera de LIEFERSCHEIN.
type header struct {
	Lieferschein db.Row
	Liefer       db.Row
	Kostst       db.Row // resuelto via primer Lieferpos.KST_ID
	Lieferpos    []db.Row
}

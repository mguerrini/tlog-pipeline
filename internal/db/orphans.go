package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Relation describe una FK lógica entre dos tablas.
type Relation struct {
	ChildTable  string
	ChildCol    string
	ParentTable string
	ParentCol   string
}

// RelationResult contiene el resultado de chequear una relación.
type RelationResult struct {
	Relation       Relation
	RowsTotal      int    // filas hijas con el campo no nulo
	DistinctTotal  int    // valores distintos no nulos
	RowsOrphan     int    // filas huérfanas
	DistinctOrphan int    // valores distintos huérfanos
	SampleOrphans  []string
	CheckError     error
}

// OrphanReport agrupa todos los chequeos del día.
type OrphanReport struct {
	Day           time.Time
	GeneratedAt   time.Time
	Results       []RelationResult
	OverallStatus string // OK | OK_WITH_ORPHANS | OK_WITH_CHECK_ERRORS
}

// DefaultRelations es la lista cerrada de FKs lógicas a verificar, derivada
// de los mapeos TLOG y de Estructura_Tablas_Spec.md.
var DefaultRelations = []Relation{
	{"LIEFERSCHEIN", "LF_ID", "LIEFER", "LF_ID"},
	{"LIEFERPOS", "LFS_ID", "LIEFERSCHEIN", "LFS_ID"},
	{"LIEFERPOS", "KST_ID", "KOSTST", "KST_ID"},
	{"LIEFERPOS", "LF_ID", "LIEFER", "LF_ID"},
	{"LIEFERPOS", "ART_NR", "ARTIKEL", "ART_ID"},
	{"LIEFERPOS", "VPK_ID1", "VPCKEINH", "VPK_ID"},
	{"LIEFERPOS", "VPK_ID2", "VPCKEINH", "VPK_ID"},
	{"INVPOSART", "INV_ID", "INVENTUR", "INV_ID"},
	{"INVPOSART", "ART_ID", "ARTIKEL", "ART_ID"},
	{"INVPOSART", "VPK_ID", "VPCKEINH", "VPK_ID"},
	{"INVPOSART", "WGR_NR", "WARENGRUPPE", "WGR_ID"},
	{"INVENTUR", "KST_ID", "KOSTST", "KST_ID"},
	{"HIS_VERBRAUCH", "KST_ID", "KOSTST", "KST_ID"},
	{"DAILYTOTALS", "KST_ID", "KOSTST", "KST_ID"},
	{"DAILYTOTALS", "ART_ID", "ARTIKEL", "ART_ID"},
	{"ARTIKEL", "WGR_ID", "WARENGRUPPE", "WGR_ID"},
	{"ARTIKEL", "VPK_NR", "VPCKEINH", "VPK_ID"},
	{"VPCKEINH", "ART_NR", "ARTIKEL", "ART_ID"},
}

// RunOrphanCheck recorre las relaciones contra la store y produce el reporte.
func RunOrphanCheck(s *Store, day time.Time, relations []Relation) *OrphanReport {
	if relations == nil {
		relations = DefaultRelations
	}
	report := &OrphanReport{
		Day:         day,
		GeneratedAt: time.Now().UTC(),
	}
	hasOrphans := false
	hasErrors := false

	for _, rel := range relations {
		res := RelationResult{Relation: rel}
		child := s.Tables[rel.ChildTable]
		parent := s.Tables[rel.ParentTable]

		if child == nil {
			res.CheckError = fmt.Errorf("tabla hija %s no cargada", rel.ChildTable)
			hasErrors = true
			report.Results = append(report.Results, res)
			continue
		}
		if parent == nil {
			res.CheckError = fmt.Errorf("tabla padre %s no cargada", rel.ParentTable)
			hasErrors = true
			report.Results = append(report.Results, res)
			continue
		}

		parentSet := make(map[string]struct{}, len(parent.Rows))
		for _, r := range parent.Rows {
			v := r[rel.ParentCol]
			if !IsNull(v) {
				parentSet[v] = struct{}{}
			}
		}

		distinctChild := make(map[string]struct{})
		distinctOrphan := make(map[string]struct{})
		rowsTotal := 0
		rowsOrphan := 0

		for _, r := range child.Rows {
			v := r[rel.ChildCol]
			if IsNull(v) {
				continue
			}
			rowsTotal++
			distinctChild[v] = struct{}{}
			if _, ok := parentSet[v]; !ok {
				rowsOrphan++
				distinctOrphan[v] = struct{}{}
			}
		}
		res.RowsTotal = rowsTotal
		res.DistinctTotal = len(distinctChild)
		res.RowsOrphan = rowsOrphan
		res.DistinctOrphan = len(distinctOrphan)
		if rowsOrphan > 0 {
			hasOrphans = true
			res.SampleOrphans = sampleSorted(distinctOrphan, 10)
		}
		report.Results = append(report.Results, res)
	}

	switch {
	case hasErrors:
		report.OverallStatus = "OK_WITH_CHECK_ERRORS"
	case hasOrphans:
		report.OverallStatus = "OK_WITH_ORPHANS"
	default:
		report.OverallStatus = "OK"
	}
	return report
}

func sampleSorted(set map[string]struct{}, max int) []string {
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// CountOrphanRelations cuenta cuántas relaciones presentan al menos un huérfano.
func (r *OrphanReport) CountOrphanRelations() int {
	n := 0
	for _, res := range r.Results {
		if res.RowsOrphan > 0 {
			n++
		}
	}
	return n
}

// TotalOrphanRows suma todas las filas huérfanas.
func (r *OrphanReport) TotalOrphanRows() int {
	n := 0
	for _, res := range r.Results {
		n += res.RowsOrphan
	}
	return n
}

// WriteOrphanReportMD genera el archivo Markdown según el formato definido
// en el addendum a ARQUITECTURA_PIPELINE_TLOG.md.
func WriteOrphanReportMD(report *OrphanReport, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Orphans Report — %s\n\n", report.Day.Format("2006-01-02"))
	fmt.Fprintf(&b, "- **Status:** %s\n", report.OverallStatus)
	fmt.Fprintf(&b, "- **Generated at:** %s\n", report.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(&b, "- **Step:** create_db\n")
	fmt.Fprintf(&b, "- **Total relations checked:** %d\n", len(report.Results))
	fmt.Fprintf(&b, "- **Relations with orphans:** %d\n", report.CountOrphanRelations())
	fmt.Fprintf(&b, "- **Total orphan rows:** %d\n", report.TotalOrphanRows())
	fmt.Fprintf(&b, "- **Total distinct orphan values:** %d\n\n", totalDistinctOrphans(report))
	b.WriteString("---\n\n## Resumen por relación\n\n")
	b.WriteString("| # | Hija → Padre | Filas hija | Distintos hija | Filas huérfanas | Distintos huérfanos | Estado |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|:---:|\n")
	for i, res := range report.Results {
		state := "✅"
		if res.CheckError != nil {
			state = "❌ ERROR"
		} else if res.RowsOrphan > 0 {
			state = "⚠️"
		}
		fmt.Fprintf(&b, "| %d | %s.%s → %s.%s | %d | %d | %d | %d | %s |\n",
			i+1,
			res.Relation.ChildTable, res.Relation.ChildCol,
			res.Relation.ParentTable, res.Relation.ParentCol,
			res.RowsTotal, res.DistinctTotal, res.RowsOrphan, res.DistinctOrphan, state)
	}
	b.WriteString("\n---\n\n## Detalle de huérfanos\n\n")

	hasDetail := false
	for _, res := range report.Results {
		if res.CheckError != nil {
			fmt.Fprintf(&b, "### ❌ %s.%s → %s.%s\n\n",
				res.Relation.ChildTable, res.Relation.ChildCol,
				res.Relation.ParentTable, res.Relation.ParentCol)
			fmt.Fprintf(&b, "Error de validación: %s\n\n", res.CheckError.Error())
			hasDetail = true
			continue
		}
		if res.RowsOrphan == 0 {
			continue
		}
		hasDetail = true
		fmt.Fprintf(&b, "### ⚠️ %s.%s → %s.%s\n\n",
			res.Relation.ChildTable, res.Relation.ChildCol,
			res.Relation.ParentTable, res.Relation.ParentCol)
		fmt.Fprintf(&b, "- **Filas huérfanas:** %d / %d\n", res.RowsOrphan, res.RowsTotal)
		fmt.Fprintf(&b, "- **Valores distintos huérfanos:** %d / %d\n",
			res.DistinctOrphan, res.DistinctTotal)
		fmt.Fprintf(&b, "- **Sample (hasta 10):**\n\n")
		b.WriteString("| orphan_value |\n|---|\n")
		for _, v := range res.SampleOrphans {
			fmt.Fprintf(&b, "| %s |\n", v)
		}
		if res.DistinctOrphan > len(res.SampleOrphans) {
			fmt.Fprintf(&b, "\n... y %d valores distintos más.\n",
				res.DistinctOrphan-len(res.SampleOrphans))
		}
		b.WriteString("\n")
	}
	if !hasDetail {
		b.WriteString("Sin huérfanos detectados.\n")
	}

	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}

func totalDistinctOrphans(r *OrphanReport) int {
	n := 0
	for _, res := range r.Results {
		n += res.DistinctOrphan
	}
	return n
}

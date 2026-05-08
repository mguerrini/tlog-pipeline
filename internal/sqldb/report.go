package sqldb

import (
	"fmt"
	"strings"
	"time"
)

// WriteReportMD genera el reporte Markdown del LoadResult y lo devuelve como string.
func WriteReportMD(r *LoadResult, day time.Time) string {
	var b strings.Builder

	status := "✅ OK"
	if !r.OverallOK {
		status = "❌ CON ERRORES"
	}

	fmt.Fprintf(&b, "# SQL DB Load Report — %s\n\n", day.Format("2006-01-02"))
	fmt.Fprintf(&b, "- **Estado:** %s\n", status)
	fmt.Fprintf(&b, "- **Generado:** %s\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(&b, "- **Base de datos:** `%s`\n\n", r.DBPath)

	// Carga
	b.WriteString("---\n\n## Carga de tablas\n\n")
	b.WriteString("| Tabla | Archivo | Filas | Tiempo | Estado |\n")
	b.WriteString("|---|---|---:|---:|:---:|\n")
	for _, s := range r.Stats {
		icon := "✅"
		if s.Err != nil {
			icon = "❌"
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | %d | %s | %s |\n",
			s.Table, s.CSVFile, s.Inserted,
			s.Duration.Round(time.Millisecond), icon)
		if s.Err != nil {
			fmt.Fprintf(&b, "| | _error: %s_ | | | |\n", s.Err.Error())
		}
	}

	// Conteos
	b.WriteString("\n---\n\n## Conteos post-carga\n\n")
	b.WriteString("| Tabla | Esperadas | Cargadas | Estado |\n")
	b.WriteString("|---|---:|---:|:---:|\n")
	for _, c := range r.Counts {
		icon := "✅"
		if !c.Match {
			icon = "⚠️"
		}
		fmt.Fprintf(&b, "| `%s` | %d | %d | %s |\n", c.Table, c.Expected, c.Got, icon)
	}

	// Integridad
	b.WriteString("\n---\n\n## Integridad referencial\n\n")
	b.WriteString("| Relación | Huérfanas | Estado |\n")
	b.WriteString("|---|---:|:---:|\n")
	for _, o := range r.Orphans {
		icon := "✅"
		if !o.OK {
			icon = "❌"
		} else if o.OrphanRows > 0 {
			icon = "⚠️ esperado"
		}
		fmt.Fprintf(&b, "| %s | %d | %s |\n", o.Label, o.OrphanRows, icon)
	}

	return b.String()
}

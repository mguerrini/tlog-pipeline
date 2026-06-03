package csvio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TableMapping mapea prefijo del archivo (sin _yyyymmdd.csv) → nombre de tabla SQLite.
// Sigue exactamente Estructura_Tablas_Spec.md sección 6.8.
var TableMapping = map[string]string{
	"Kostst":        "KOSTST",
	"Liefer":        "LIEFER",
	"Warengruppe":   "WARENGRUPPE",
	"Vpckeinh":      "VPCKEINH",
	"Artikel":       "ARTIKEL",
	"Lieferschein":  "LIEFERSCHEIN",
	"Lieferpos":     "LIEFERPOS",
	"Rechnung":      "RECHNUNG",
	"Rechlfs":       "RECHLFS",
	"Inventur":      "INVENTUR",
	"Invposart":     "INVPOSART",
	"His_verbrauch":    "HIS_VERBRAUCH",
	"His_Verbrauchpos": "HIS_VERBRAUCHPOS",
	"Dailytotals":      "DAILYTOTALS",
}

// LoadOrder define el orden de carga respetando dependencias FK lógicas.
var LoadOrder = []string{
	"KOSTST",
	"LIEFER",
	"WARENGRUPPE",
	"VPCKEINH",
	"ARTIKEL",
	"LIEFERSCHEIN",
	"LIEFERPOS",
	"RECHNUNG",
	"RECHLFS",
	"INVENTUR",
	"INVPOSART",
	"HIS_VERBRAUCH",
	"HIS_VERBRAUCHPOS",
	"DAILYTOTALS",
}

// PatternToTable convierte un patrón de expected_files (ej. "Kostst_*.csv") al
// nombre de tabla SQLite que le corresponde.
func PatternToTable(pattern string) string {
	// Extrae el prefijo antes del primer "_"
	idx := strings.Index(pattern, "_")
	if idx < 0 {
		return ""
	}
	prefix := pattern[:idx]
	// Casos especiales con "_" en el prefijo
	if strings.HasPrefix(pattern, "His_Verbrauchpos") {
		prefix = "His_Verbrauchpos"
	} else if strings.HasPrefix(pattern, "His_verbrauch") {
		prefix = "His_verbrauch"
	}
	return TableMapping[prefix]
}

// FoundFile resuelve archivo concreto + tabla destino.
type FoundFile struct {
	Table string
	Path  string
}

// FindFiles busca en dir los CSVs cuyo prefijo está en TableMapping.
// Si hay varios para una misma tabla, toma el primero (alfabético) y avisa.
func FindFiles(dir string) (map[string]string, error) {
	out := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("leer dir %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".csv") {
			continue
		}
		// El prefijo es la parte antes del primer "_". Ej: "Kostst_20260505.csv" → "Kostst".
		idx := strings.Index(name, "_")
		if idx < 0 {
			continue
		}
		prefix := name[:idx]
		// Casos especiales con "_" en el prefijo.
		if strings.HasPrefix(name, "His_Verbrauchpos_") {
			prefix = "His_Verbrauchpos"
		} else if strings.HasPrefix(name, "His_verbrauch_") {
			prefix = "His_verbrauch"
		}
		table, ok := TableMapping[prefix]
		if !ok {
			continue
		}
		// Si ya hay un mapping previo, dejar el primero
		if _, exists := out[table]; !exists {
			out[table] = filepath.Join(dir, name)
		}
	}
	return out, nil
}

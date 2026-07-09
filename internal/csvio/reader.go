// Package csvio carga CSVs como slices de mapas string→string. La conversión
// de tipos se hace al insertar en la DB, no al parsear.
package csvio

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// Read lee un CSV con el separador dado y devuelve:
//   - la lista de columnas en el orden del header
//   - los registros como []map[string]string
//
// Strings vacíos quedan como "" en el mapa; el caller decide cómo tratar NULL.
func Read(path, sep string) ([]string, []map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	if sep == "" {
		sep = ","
	}
	r.Comma = []rune(sep)[0]
	r.FieldsPerRecord = -1 // tolerante a líneas con campos desparejos
	r.LazyQuotes = true
	r.TrimLeadingSpace = false

	header, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil, fmt.Errorf("archivo %s vacío", path)
		}
		return nil, nil, fmt.Errorf("leer header de %s: %w", path, err)
	}
	// Limpiar BOM si presente en el primer header
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	for i := range header {
		h := strings.TrimSpace(strings.TrimSuffix(header[i], "\r"))
		h = strings.Trim(h, "\"") // strip comillas opcionales (ej. "RNG_NAME" \u2192 RNG_NAME)
		header[i] = strings.ToUpper(h)
	}

	var rows []map[string]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("leer fila de %s: %w", path, err)
		}
		row := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(rec) {
				v := strings.TrimSuffix(rec[i], "\r")
				row[h] = strings.Trim(v, "\"")
			} else {
				row[h] = ""
			}
		}
		rows = append(rows, row)
	}
	return header, rows, nil
}

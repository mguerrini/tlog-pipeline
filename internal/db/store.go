// Package db provee una "base de datos" en memoria construida desde los CSVs
// del día. Reemplaza el uso de SQLite original: dado que los volúmenes son
// chicos (≤ pocos miles de filas por tabla), los maps en memoria son más que
// suficientes y eliminan toda dependencia externa.
//
// Convenciones:
//   - Las celdas se almacenan como string (tal cual del CSV); los helpers
//     `helpers.go` ofrecen conversiones a int / float / NULL.
//   - Cada tabla preserva el header original como []string para iteraciones
//     deterministas.
//   - Los lookups por PK se construyen en `BuildIndexes`.
package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Row es un registro genérico (columna→valor).
type Row map[string]string

// Table es una tabla cargada desde CSV.
type Table struct {
	Name   string
	Header []string
	Rows   []Row
}

// Store es el contenedor de todas las tablas del día + lookups precomputados.
type Store struct {
	Tables map[string]*Table

	// Lookups por PK natural. Se construyen en BuildIndexes().
	Kostst       map[string]Row // KST_ID -> row
	Liefer       map[string]Row // LF_ID -> row
	Warengruppe  map[string]Row // WGR_ID -> row
	Vpckeinh     map[string]Row // VPK_ID -> row
	Artikel      map[string]Row // ART_ID -> row
	Lieferschein map[string]Row // LFS_ID -> row
	Inventur     map[string]Row // INV_ID -> row

	// Lookups secundarios.
	LieferposByLFS map[string][]Row // LFS_ID -> rows de LIEFERPOS
	InvposartByINV map[string][]Row // INV_ID -> rows de INVPOSART
}

// NewStore crea una store vacía.
func NewStore() *Store {
	return &Store{Tables: make(map[string]*Table)}
}

// AddTable registra una tabla con su header y filas.
func (s *Store) AddTable(name string, header []string, rows []Row) {
	s.Tables[name] = &Table{Name: name, Header: header, Rows: rows}
}

// BuildIndexes construye todos los lookups en función de las tablas cargadas.
// Se llama una sola vez después del bulk load.
func (s *Store) BuildIndexes() {
	s.Kostst = indexBy(s.Tables["KOSTST"], "KST_ID")
	s.Liefer = indexBy(s.Tables["LIEFER"], "LF_ID")
	s.Warengruppe = indexBy(s.Tables["WARENGRUPPE"], "WGR_ID")
	s.Vpckeinh = indexBy(s.Tables["VPCKEINH"], "VPK_ID")
	s.Artikel = indexBy(s.Tables["ARTIKEL"], "ART_ID")
	s.Lieferschein = indexBy(s.Tables["LIEFERSCHEIN"], "LFS_ID")
	s.Inventur = indexBy(s.Tables["INVENTUR"], "INV_ID")

	s.LieferposByLFS = groupBy(s.Tables["LIEFERPOS"], "LFS_ID")
	s.InvposartByINV = groupBy(s.Tables["INVPOSART"], "INV_ID")
}

// indexBy crea un mapa key→row con la primera fila ganadora ante duplicados.
func indexBy(t *Table, keyCol string) map[string]Row {
	out := make(map[string]Row)
	if t == nil {
		return out
	}
	for _, r := range t.Rows {
		k := r[keyCol]
		if k == "" {
			continue
		}
		if _, exists := out[k]; !exists {
			out[k] = r
		}
	}
	return out
}

// groupBy agrupa filas por valor de keyCol.
func groupBy(t *Table, keyCol string) map[string][]Row {
	out := make(map[string][]Row)
	if t == nil {
		return out
	}
	for _, r := range t.Rows {
		k := r[keyCol]
		if k == "" {
			continue
		}
		out[k] = append(out[k], r)
	}
	return out
}

// SaveSnapshot escribe un JSON con todas las tablas para diagnóstico.
// Es opcional; el pipeline puede omitirlo si keep_db_after_run = false.
func (s *Store) SaveSnapshot(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	type snapshot struct {
		Tables []*Table `json:"tables"`
	}
	tables := make([]*Table, 0, len(s.Tables))
	for _, t := range s.Tables {
		tables = append(tables, t)
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	b, err := json.MarshalIndent(snapshot{Tables: tables}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return os.WriteFile(path, b, 0o644)
}

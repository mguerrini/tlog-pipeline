// Package timeutil tiene helpers para parseo y formato de fechas en los
// formatos que usa el pipeline (AAAAMMDD para nombres de archivo y carpeta,
// y YYYY-MM-DD HH:MM:SS para campos XML).
package timeutil

import (
	"fmt"
	"strings"
	"time"
)

const (
	LayoutCompact = "20060102"
	LayoutISODate = "2006-01-02"
	LayoutDateTimeNoTZ = "2006-01-02 15:04:05"
)

// ParseDay acepta "AAAAMMDD" o "AAAA-MM-DD" y devuelve la fecha (UTC, hora 0).
func ParseDay(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.ParseInLocation(LayoutCompact, s, time.UTC); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation(LayoutISODate, s, time.UTC); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("formato de fecha inválido: %q (esperado AAAAMMDD o AAAA-MM-DD)", s)
}

// FormatCompact devuelve "AAAAMMDD".
func FormatCompact(t time.Time) string { return t.Format(LayoutCompact) }

// FormatISODate devuelve "AAAA-MM-DD".
func FormatISODate(t time.Time) string { return t.Format(LayoutISODate) }

// ApplyOffset suma a t (medianoche del día) un offset HH:MM:SS y devuelve el
// timestamp resultante. Si offset no es parseable, devuelve t sin cambios y un error.
func ApplyOffset(day time.Time, hhmmss string) (time.Time, error) {
	hh, mm, ss, err := parseHHMMSS(hhmmss)
	if err != nil {
		return day, err
	}
	return day.Add(time.Duration(hh)*time.Hour +
		time.Duration(mm)*time.Minute +
		time.Duration(ss)*time.Second), nil
}

func parseHHMMSS(s string) (hh, mm, ss int, err error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("offset inválido %q (formato esperado HH:MM:SS)", s)
	}
	if _, err = fmt.Sscanf(s, "%d:%d:%d", &hh, &mm, &ss); err != nil {
		return 0, 0, 0, fmt.Errorf("offset inválido %q: %w", s, err)
	}
	return hh, mm, ss, nil
}

// FormatDateTimeNoTZ devuelve "AAAA-MM-DD HH:MM:SS".
func FormatDateTimeNoTZ(t time.Time) string { return t.Format(LayoutDateTimeNoTZ) }

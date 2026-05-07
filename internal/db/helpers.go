package db

import (
	"strconv"
	"strings"
)

// IsNull devuelve true si el valor crudo del CSV se considera nulo.
// Por convención del proyecto: string vacío y "NULL" (case-insensitive).
func IsNull(s string) bool {
	t := strings.TrimSpace(s)
	return t == "" || strings.EqualFold(t, "NULL")
}

// AsInt parsea un entero. Devuelve 0 y false si no se puede.
func AsInt(s string) (int64, bool) {
	if IsNull(s) {
		return 0, false
	}
	// Algunos CSV traen "1.0" donde se espera entero
	if i, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return int64(f), true
	}
	return 0, false
}

// AsFloat parsea un float. Devuelve 0 y false si no se puede.
func AsFloat(s string) (float64, bool) {
	if IsNull(s) {
		return 0, false
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return f, true
	}
	return 0, false
}

// MustAsInt es AsInt forzando 0 si falla. Útil cuando el campo se considera
// "vacío = 0" (ej. flags).
func MustAsInt(s string) int64 {
	v, _ := AsInt(s)
	return v
}

// MustAsFloat es AsFloat forzando 0 si falla.
func MustAsFloat(s string) float64 {
	v, _ := AsFloat(s)
	return v
}

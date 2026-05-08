// Package common reúne helpers compartidos por los mappers de TLOG:
// formato de fechas/decimales y helpers de escritura XML. La generación
// del SEQUENCENUMBER vive en internal/sequence.
package common

import (
	"fmt"
	"strings"
)

// FormatDecimal4 formatea un float con 4 decimales: "100.0000".
func FormatDecimal4(v float64) string {
	return fmt.Sprintf("%.4f", v)
}

// PadLeft rellena s a la izquierda con pad hasta length chars.
func PadLeft(s string, length int, pad rune) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat(string(pad), length-len(s)) + s
}

// FormatRetailStoreID toma un KST_CODE crudo y lo normaliza al formato APIES
// del XML: 5 dígitos rellenando con ceros a la izquierda. Si KST_CODE trae
// prefijos no numéricos (ej. "CC 31252"), se quedan los dígitos finales.
func FormatRetailStoreID(kstCode string) string {
	digits := extractTrailingDigits(kstCode)
	if digits == "" {
		// fallback: devolver tal cual padded
		return PadLeft(kstCode, 5, '0')
	}
	return PadLeft(digits, 5, '0')
}

func extractTrailingDigits(s string) string {
	out := ""
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if c >= '0' && c <= '9' {
			out = string(c) + out
		} else if out != "" {
			break
		}
	}
	return out
}

// FormatDetSeq formatea el DET_SEQUENCENUMBER (3 dígitos, 1..999).
func FormatDetSeq(n int) string {
	return fmt.Sprintf("%d", n) // sin pad: el doc dice "numérico de 1 a 999"
}

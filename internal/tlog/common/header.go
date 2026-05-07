// Package common reúne helpers compartidos por los mappers de TLOG:
// formato de fechas/decimales, generación de SequenceNumber, helpers de
// escritura XML.
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

// BuildSequenceNumber12 construye un SEQUENCENUMBER tipo SU (12 dígitos).
// Formato: "9" + APIES(5 dígitos sin pad) + secuencial(6 dígitos).
// La regla canónica del proyecto es: empieza por 9, longitud 12.
//
// Como el secuencial real lo arma Bridge en producción, acá simulamos:
// retailDigits es la APIES sin pad (ej "19" para "00019").
func BuildSequenceNumber12(retailStoreID string, seq int) string {
	digits := strings.TrimLeft(retailStoreID, "0")
	if digits == "" {
		digits = "0"
	}
	// "9" + APIES (5) + seq (6)  → 12 dígitos
	apies := PadLeft(digits, 5, '0')
	return fmt.Sprintf("9%s%06d", apies, seq)
}

// BuildSequenceNumber11 construye un SEQUENCENUMBER tipo Bridge (11 dígitos).
// Formato: "9" + APIES(5) + secuencial(5) → total 11.
func BuildSequenceNumber11(retailStoreID string, seq int) string {
	digits := strings.TrimLeft(retailStoreID, "0")
	if digits == "" {
		digits = "0"
	}
	apies := PadLeft(digits, 5, '0')
	return fmt.Sprintf("9%s%05d", apies, seq)
}

// FormatDetSeq formatea el DET_SEQUENCENUMBER (3 dígitos, 1..999).
func FormatDetSeq(n int) string {
	return fmt.Sprintf("%d", n) // sin pad: el doc dice "numérico de 1 a 999"
}

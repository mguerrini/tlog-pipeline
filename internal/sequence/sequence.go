// Package sequence implementa el algoritmo de construcción y descomposición
// del campo SEQUENCENUMBER usado en los TLOG OCPRA.
//
// Formato (12 dígitos):
//
//	[9:1][AAMMDD:6][DOC:1][CONTADOR:4]
//
// Donde DOC es el índice del tipo de archivo (0..7) y CONTADOR es un
// secuencial por (día × tipo) que arranca en 0 y se incrementa para cada
// documento adicional del mismo tipo en el mismo día.
//
// Ver docs/SEQUENCENUMBER.md para la especificación completa.
package sequence

import (
	"errors"
	"fmt"
	"time"
)

// DocumentNumber es el índice del tipo de archivo TLOG (0..7).
type DocumentNumber int

const (
	DocReception   DocumentNumber = 0
	DocReturn      DocumentNumber = 1
	DocTransfer    DocumentNumber = 2
	DocAdjustment  DocumentNumber = 3
	DocCount       DocumentNumber = 4
	DocFiscalDocFC DocumentNumber = 5
	DocFiscalDocNC DocumentNumber = 6
	DocCierre      DocumentNumber = 7
)

// String devuelve el nombre legible del tipo de documento.
func (d DocumentNumber) String() string {
	switch d {
	case DocReception:
		return "Reception"
	case DocReturn:
		return "Return"
	case DocTransfer:
		return "Transfer"
	case DocAdjustment:
		return "Adjustment"
	case DocCount:
		return "Count"
	case DocFiscalDocFC:
		return "FiscalDocFC"
	case DocFiscalDocNC:
		return "FiscalDocNC"
	case DocCierre:
		return "Cierre"
	default:
		return fmt.Sprintf("DocumentNumber(%d)", int(d))
	}
}

const (
	prefix         = '9'
	totalLen       = 12
	counterMaxPlus = 10000 // contador 4 dígitos: 0..9999
)

// Decoded contiene los componentes lógicos extraídos de un SEQUENCENUMBER.
type Decoded struct {
	Year           int
	Month          int
	Day            int
	DocumentNumber DocumentNumber
	Counter        int
}

// BusinessDayDate reconstruye la fecha (medianoche UTC).
func (d Decoded) BusinessDayDate() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

// Build construye el SEQUENCENUMBER con el formato 9AAMMDD<DOC><CONTADOR4>.
//
// counter es el secuencial por (día × tipo) que arranca en 0. El llamador
// debe pasar 0 para el primer documento del tipo en el día, 1 para el
// segundo, etc.
func Build(businessDayDate time.Time, doc DocumentNumber, counter int) (string, error) {
	if doc < DocReception || doc > DocCierre {
		return "", fmt.Errorf("sequence: documentNumber fuera de rango: %d", int(doc))
	}
	if counter < 0 || counter >= counterMaxPlus {
		return "", fmt.Errorf("sequence: contador fuera de rango [0,%d]: %d", counterMaxPlus-1, counter)
	}
	aa := businessDayDate.Year() % 100
	mm := int(businessDayDate.Month())
	dd := businessDayDate.Day()
	return fmt.Sprintf("%c%02d%02d%02d%d%04d", prefix, aa, mm, dd, int(doc), counter), nil
}

// Decode descompone un SEQUENCENUMBER en sus componentes lógicos.
// yearWindowStart resuelve la ambigüedad del tramo AA (usar 2000 para
// asumir siglo XXI; usar 0 para el default 2000+AA).
func Decode(seq string, yearWindowStart int) (Decoded, error) {
	var d Decoded
	if !isAllDigits(seq) {
		return d, errors.New("sequence: contiene caracteres no numéricos")
	}
	if len(seq) != totalLen {
		return d, fmt.Errorf("sequence: longitud inválida %d (esperado %d)", len(seq), totalLen)
	}
	if seq[0] != prefix {
		return d, fmt.Errorf("sequence: prefijo inválido %q (esperado %q)", seq[0], prefix)
	}
	aa, _ := atoiDigits(seq[1:3])
	mm, _ := atoiDigits(seq[3:5])
	dd, _ := atoiDigits(seq[5:7])
	docDigit, _ := atoiDigits(seq[7:8])
	counter, _ := atoiDigits(seq[8:12])

	if docDigit > 7 {
		return d, fmt.Errorf("sequence: documentNumber fuera de rango: %d", docDigit)
	}

	d.Year = expandAA(aa, yearWindowStart)
	d.Month = mm
	d.Day = dd
	d.DocumentNumber = DocumentNumber(docDigit)
	d.Counter = counter

	if mm < 1 || mm > 12 {
		return d, fmt.Errorf("sequence: mes inválido %d", mm)
	}
	if dd < 1 || dd > 31 {
		return d, fmt.Errorf("sequence: día inválido %d", dd)
	}
	// Validar fecha real (ej: 31 de febrero).
	check := time.Date(d.Year, time.Month(mm), dd, 0, 0, 0, 0, time.UTC)
	if check.Year() != d.Year || int(check.Month()) != mm || check.Day() != dd {
		return d, fmt.Errorf("sequence: fecha inválida %04d-%02d-%02d", d.Year, mm, dd)
	}
	return d, nil
}

// expandAA resuelve el año completo a partir del tramo AA usando una ventana
// móvil. Si yearWindowStart==0 se asume siglo XXI (2000+AA).
func expandAA(aa, yearWindowStart int) int {
	if yearWindowStart == 0 {
		return 2000 + aa
	}
	startCentury := (yearWindowStart / 100) * 100
	candidate := startCentury + aa
	if candidate < yearWindowStart {
		candidate += 100
	}
	return candidate
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func atoiDigits(s string) (int, error) {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("sequence: dígito inválido %q", c)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
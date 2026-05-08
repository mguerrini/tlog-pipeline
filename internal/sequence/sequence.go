// Package sequence implementa el algoritmo de construcción y descomposición
// del campo SEQUENCENUMBER usado en los TLOG OCPRA.
//
// Formato:
//
//	SU      (12 díg): [RETAILID:6][AA:2][NUMERO_ARCHIVO:4]   (Reception..FiscalDocNC)
//	Bridge  (11 díg): [RETAILID:5][AA:2][NUMERO_ARCHIVO:4]   (Cierre)
//
// Donde NUMERO_ARCHIVO = (DDD-1)*8 + DocumentNumber, con DDD = día del año
// (1..366) y DocumentNumber el índice del tipo de archivo (0..7).
//
// Ver docs/SEQUENCENUMBER.md para la especificación completa.
package sequence

import (
	"errors"
	"fmt"
	"strings"
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

// Origin distingue los dos formatos de SEQUENCENUMBER.
type Origin int

const (
	OriginSU     Origin = iota // 12 dígitos (Reception..FiscalDocNC)
	OriginBridge               // 11 dígitos (Cierre)
)

const (
	retailWidthSU     = 6
	retailWidthBridge = 5
)

// Decoded contiene los componentes lógicos extraídos de un SEQUENCENUMBER.
type Decoded struct {
	Origin         Origin
	RetailID       string
	Year           int
	DayOfYear      int
	DocumentNumber DocumentNumber
	NumeroArchivo  int
}

// BusinessDayDate reconstruye la fecha (medianoche UTC) a partir de Year y
// DayOfYear.
func (d Decoded) BusinessDayDate() time.Time {
	return time.Date(d.Year, time.January, d.DayOfYear, 0, 0, 0, 0, time.UTC)
}

// Build construye el SEQUENCENUMBER. El origen (SU vs Bridge) se infiere
// del DocumentNumber: Cierre→Bridge (11 díg), resto→SU (12 díg).
func Build(retailID string, businessDayDate time.Time, doc DocumentNumber) (string, error) {
	if doc < DocReception || doc > DocCierre {
		return "", fmt.Errorf("sequence: documentNumber fuera de rango: %d", int(doc))
	}
	retailWidth := retailWidthSU
	if doc == DocCierre {
		retailWidth = retailWidthBridge
	}
	if !isAllDigits(retailID) {
		return "", fmt.Errorf("sequence: retailID no es numérico: %q", retailID)
	}
	if len(retailID) > retailWidth {
		return "", fmt.Errorf("sequence: retailID %q excede el ancho %d del tramo", retailID, retailWidth)
	}
	retailPadded := padLeftZero(retailID, retailWidth)
	aa := businessDayDate.Year() % 100
	ddd := businessDayDate.YearDay() // 1..366
	numeroArchivo := (ddd-1)*8 + int(doc)
	return fmt.Sprintf("%s%02d%04d", retailPadded, aa, numeroArchivo), nil
}

// Decode descompone un SEQUENCENUMBER en sus componentes lógicos.
// yearWindowStart resuelve la ambigüedad del tramo AA (usar 2000 para
// asumir siglo XXI; usar 0 para el default 2000+AA).
func Decode(seq string, yearWindowStart int) (Decoded, error) {
	var d Decoded
	if !isAllDigits(seq) {
		return d, errors.New("sequence: contiene caracteres no numéricos")
	}
	var retailWidth int
	switch len(seq) {
	case 12:
		d.Origin = OriginSU
		retailWidth = retailWidthSU
	case 11:
		d.Origin = OriginBridge
		retailWidth = retailWidthBridge
	default:
		return d, fmt.Errorf("sequence: longitud inválida %d (esperado 11 ó 12)", len(seq))
	}
	d.RetailID = seq[:retailWidth]
	aa, err := atoiDigits(seq[retailWidth : retailWidth+2])
	if err != nil {
		return d, err
	}
	numero, err := atoiDigits(seq[retailWidth+2:])
	if err != nil {
		return d, err
	}
	d.NumeroArchivo = numero
	d.DayOfYear = numero/8 + 1
	d.DocumentNumber = DocumentNumber(numero % 8)

	if d.DocumentNumber == DocCierre && d.Origin != OriginBridge {
		return d, fmt.Errorf("sequence: documentNumber=Cierre incompatible con origen SU (12 díg)")
	}
	if d.DocumentNumber != DocCierre && d.Origin != OriginSU {
		return d, fmt.Errorf("sequence: documentNumber=%s incompatible con origen Bridge (11 díg)", d.DocumentNumber)
	}

	d.Year = expandAA(aa, yearWindowStart)

	if d.DayOfYear < 1 || d.DayOfYear > 366 {
		return d, fmt.Errorf("sequence: dayOfYear %d fuera de [1,366]", d.DayOfYear)
	}
	if d.DayOfYear == 366 && !isLeapYear(d.Year) {
		return d, fmt.Errorf("sequence: dayOfYear=366 no válido en año no bisiesto %d", d.Year)
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

func isLeapYear(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
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

func padLeftZero(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
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
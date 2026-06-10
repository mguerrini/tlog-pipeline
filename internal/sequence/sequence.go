// Package sequence implementa el algoritmo de construcción y descomposición
// del campo SEQUENCENUMBER usado en los TLOG OCPRA.
//
// Formato (13 dígitos):
//
//	[9:1][AAMMDD:6][DOC:2][CONTADOR:4]
//
// Donde DOC es el índice del tipo de archivo (2 dígitos, 00..99) y CONTADOR es
// un secuencial por (día × tipo) que arranca en 0 y se incrementa para cada
// documento adicional del mismo tipo en el mismo día.
//
// Ver docs/SEQUENCENUMBER.md para la especificación completa.
package sequence

import (
	"fmt"
	"time"
)

// DocumentNumber es el índice del tipo de archivo TLOG (0..7).
type DocumentNumber int

const (
	DocReception           DocumentNumber = 0
	DocReturn              DocumentNumber = 1
	DocTransfer            DocumentNumber = 2
	DocFiscalDocFC         DocumentNumber = 5
	DocFiscalDocNC         DocumentNumber = 6
	DocCierre              DocumentNumber = 7
	DocAdjustmentVerbrauch DocumentNumber = 3
	DocAdjustmentInventur  DocumentNumber = 3
	DocCountVerbrauch      DocumentNumber = 4
	DocCountInventur       DocumentNumber = 4
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
	case DocAdjustmentVerbrauch: // = DocAdjustmentInventur = 3
		return "Adjustment"
	case DocFiscalDocFC:
		return "FiscalDocFC"
	case DocFiscalDocNC:
		return "FiscalDocNC"
	case DocCierre:
		return "Cierre"
	case DocCountVerbrauch: // = DocCountInventur = 4
		return "Count"
	default:
		return fmt.Sprintf("DocumentNumber(%d)", int(d))
	}
}

const (
	prefix         = '9'
	totalLen       = 13
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
	if doc < 0 || doc > 9 {
		return "", fmt.Errorf("sequence: documentNumber fuera de rango: %d", int(doc))
	}
	if counter < 0 || counter >= counterMaxPlus {
		return "", fmt.Errorf("sequence: contador fuera de rango [0,%d]: %d", counterMaxPlus-1, counter)
	}
	aa := businessDayDate.Year() % 100
	mm := int(businessDayDate.Month())
	dd := businessDayDate.Day()
	return fmt.Sprintf("%c%02d%02d%02d%01d%04d", prefix, aa, mm, dd, int(doc), counter), nil
}

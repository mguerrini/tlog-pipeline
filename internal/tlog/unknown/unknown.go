// Package unknown encapsula el manejo de campos sin mapeo definido. La regla
// del proyecto es: cuando no se sabe cómo mapear un campo, el valor del XML
// debe ser:
//
//	"[UNKNOWN] - {valor xml ejemplo} - {dudas/opciones}"
package unknown

import "fmt"

// Emit devuelve la cadena con el formato canónico [UNKNOWN].
//
//	xmlValue: el texto que figura en el XML real (puede estar vacío)
//	doubts:   notas / preguntas para el equipo de negocio
func Emit(xmlValue, doubts string) string {
	if xmlValue == "" {
		xmlValue = "(vacío)"
	}
	if doubts == "" {
		doubts = "Validar con negocio"
	}
	return fmt.Sprintf("[UNKNOWN] - {%s} - {%s}", xmlValue, doubts)
}

package tlogsql

// mapVrtIDToAdjType convierte VRT_ID al tipo de ajuste TLOG.
// Tabla de traducción pendiente de validación con OCPRA (UNKNOWN A DEFINIR).
func mapVrtIDToAdjType(vrtID string) string {
	switch vrtID {
	case "1":
		return "UNJUSTIFIED_DEPLETIONS"
	case "2":
		return "JUSTIFIED_ADJUSTMENTS"
	case "3":
		return "UNJUSTIFIED_ADJUSTMENTS"
	case "4":
		return "JUSTIFIED_DEPLETIONS"
	default:
		return "CORRECTIVE_ADJUSTMENT"
	}
}

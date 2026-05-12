package common

import "time"

// HeaderCtx contiene los datos comunes inyectados al header de cada TLOG.
// Todos los mappers reciben un HeaderCtx + la store + el retailID a procesar.
type HeaderCtx struct {
	BusinessDay   time.Time
	BeginDateTime time.Time
	EndDateTime   time.Time
	OperatorID    string
	RetailStoreID string // formato "00019" (5 dígitos)
	WorkstationID string // siempre "0"
	Period        string // siempre "0"
	Subperiod     string // siempre "0"
}

// FormatBusinessDayDate devuelve "YYYY-MM-DD" del BusinessDay.
func (h *HeaderCtx) FormatBusinessDayDate() string {
	return h.BusinessDay.Format("2006-01-02")
}

// FormatBeginDateTime devuelve "YYYY-MM-DD HH:MM:SS".
func (h *HeaderCtx) FormatBeginDateTime() string {
	return h.BeginDateTime.Format("2006-01-02 15:04:05")
}

// FormatEndDateTime devuelve "YYYY-MM-DD HH:MM:SS".
func (h *HeaderCtx) FormatEndDateTime() string {
	return h.EndDateTime.Format("2006-01-02 15:04:05")
}

// FormatARTimestamp devuelve el formato "YYYY-MM-DD HH:MM:SS.000 ART"
// usado en CreateDateTimestamp / LastUpdateDate / ExpectedDeliveryDate /
// ReceiptDate de los TLOG de Inventory.
func (h *HeaderCtx) FormatARTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

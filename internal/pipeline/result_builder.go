package pipeline

import "time"

// resultBuilder facilita construir un StepResult desde un step.
type resultBuilder struct {
	r *StepResult
}

// NewResult inicia un builder con StartedAt = now.
func NewResult() *resultBuilder {
	return &resultBuilder{r: &StepResult{
		StartedAt: time.Now(),
		Meta:      make(StepMeta),
	}}
}

// Meta expone el mapa de meta del resultado para escritura directa.
var _ = (*resultBuilder)(nil) // evitar lint "unused"

// OK finaliza con status OK.
func (b *resultBuilder) OK() *StepResult {
	b.r.Status = StatusOK
	b.r.FinishedAt = time.Now()
	return b.r
}

// Skip finaliza con status Skipped.
func (b *resultBuilder) Skip(reason string) *StepResult {
	b.r.Status = StatusSkipped
	b.r.FinishedAt = time.Now()
	b.r.Meta["reason"] = reason
	return b.r
}

// Fail finaliza con status Failed.
func (b *resultBuilder) Fail(err error) *StepResult {
	b.r.Status = StatusFailed
	b.r.FinishedAt = time.Now()
	b.r.Err = err
	return b.r
}

// Meta es un shorthand para b.r.Meta.
func (b *resultBuilder) SetMeta(key string, val any) *resultBuilder {
	b.r.Meta[key] = val
	return b
}

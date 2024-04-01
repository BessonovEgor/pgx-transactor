package espool

// Tracer represents tracer behavior
//
//go:generate mockery --name=Tracer --case underscore --output=./mocks
type Tracer interface {
	TraceData(request string) func()
}

type NilTracer struct{}

// NewNilTracer creates a new nil tracer instance.
func NewNilTracer() *NilTracer {
	return &NilTracer{}
}

// TraceData default trace function.
func (NilTracer) TraceData(_ string) func() {
	return func() {}
}

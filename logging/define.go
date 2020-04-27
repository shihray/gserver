package log

// A SpanID refers to a single span.
type TraceSpan interface {

	// Trace is the root ID of the tree that contains all of the spans
	// related to this one.
	TraceId() string

	// Span is an ID that probabilistically uniquely identifies this
	// span.
	SpanId() string
}

type TraceSpanImp struct {
	Trace string `json:"Trace"`
	Span  string `json:"Span"`
}

func (this *TraceSpanImp) TraceId() string {
	return this.Trace
}

func (this *TraceSpanImp) SpanId() string {
	return this.Span
}

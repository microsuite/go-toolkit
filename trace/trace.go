package trace

import "context"

type traceIDKey struct{}
type traceInfoKey struct{}

// WithTraceID returns a context with the trace ID set.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// GetTraceID returns the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

// WithTraceInfo returns a context with the trace info set.
func WithTraceInfo(ctx context.Context, traceInfo any) context.Context {
	return context.WithValue(ctx, traceInfoKey{}, traceInfo)
}

// GetTraceInfo returns the trace info from the context.
func GetTraceInfo(ctx context.Context) any {
	return ctx.Value(traceInfoKey{})
}

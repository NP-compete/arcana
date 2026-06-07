package tracing

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func StartLLMSpan(ctx context.Context, name, model string, promptTokens int) (context.Context, trace.Span) {
	tracer := Tracer("arcana-engine")
	ctx, span := tracer.Start(ctx, name, trace.WithAttributes(
		attribute.String("llm.model", model),
		attribute.Int("llm.prompt_tokens", promptTokens),
		attribute.String("llm.provider", "anthropic"),
	))
	return ctx, span
}

func EndLLMSpan(span trace.Span, completionTokens int, costUSD float64) {
	span.SetAttributes(
		attribute.Int("llm.completion_tokens", completionTokens),
		attribute.Int("llm.total_tokens", completionTokens),
		attribute.Float64("llm.cost_usd", costUSD),
	)
	span.End()
}

func StartToolSpan(ctx context.Context, toolName string) (context.Context, trace.Span) {
	tracer := Tracer("arcana-engine")
	ctx, span := tracer.Start(ctx, "tool:"+toolName, trace.WithAttributes(
		attribute.String("tool.name", toolName),
	))
	return ctx, span
}

func EndToolSpan(span trace.Span, resultSize int, err error) {
	span.SetAttributes(attribute.Int("tool.result_size", resultSize))
	if err != nil {
		span.SetAttributes(attribute.String("tool.error", err.Error()))
	}
	span.End()
}

func StartGuardrailSpan(ctx context.Context, direction string) (context.Context, trace.Span) {
	tracer := Tracer("arcana-ward")
	ctx, span := tracer.Start(ctx, "guardrail:"+direction, trace.WithAttributes(
		attribute.String("guardrail.direction", direction),
	))
	return ctx, span
}

func EndGuardrailSpan(span trace.Span, verdict string, layers int) {
	span.SetAttributes(
		attribute.String("guardrail.verdict", verdict),
		attribute.Int("guardrail.layers_checked", layers),
	)
	span.End()
}

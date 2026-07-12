package tracing

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// headerCarrier adapts amqp.Table to OTel's TextMapCarrier so W3C trace
// context rides in message headers across the broker.
type headerCarrier amqp.Table

func (hc headerCarrier) Get(key string) string {
	if v, ok := hc[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (hc headerCarrier) Set(key, val string) { hc[key] = val }

func (hc headerCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range hc {
		keys = append(keys, k)
	}
	return keys
}

// StartPublishSpan opens a producer span and returns headers carrying the
// trace context, to be set on the outgoing amqp.Publishing. Call end with
// the publish result.
func StartPublishSpan(ctx context.Context, exchange, routingKey string) (headers amqp.Table, end func(error)) {
	ctx, span := otel.Tracer("shop-api/amqp").Start(ctx, "publish "+exchange+"/"+routingKey,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", exchange),
			attribute.String("messaging.rabbitmq.destination.routing_key", routingKey),
		),
	)
	headers = amqp.Table{}
	otel.GetTextMapPropagator().Inject(ctx, headerCarrier(headers))
	return headers, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// StartConsumeSpan extracts the trace context from delivery headers and opens
// a consumer span, so broker-delivered work joins the trace that published it.
// The returned ctx carries the span for the handler; call end with the final
// handling result.
func StartConsumeSpan(ctx context.Context, queue, routingKey string, headers amqp.Table) (context.Context, func(error)) {
	parent := otel.GetTextMapPropagator().Extract(ctx, headerCarrier(headers))
	ctx, span := otel.Tracer("shop-api/amqp").Start(parent, "consume "+queue,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination.name", queue),
			attribute.String("messaging.rabbitmq.destination.routing_key", routingKey),
		),
	)
	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

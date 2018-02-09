import event
import itertools
from spanid import SpanID, Annotation
from basictracer import BasicTracer, SpanRecorder

def create_new_tracer(collector, sampler=None):
    """create_new_tracer creates a new appdash opentracing tracer using an Appdash collector.
    """
    return BasicTracer(recorder=AppdashRecorder(collector), sampler=sampler)

class AppdashRecorder(SpanRecorder):
    """AppdashRecorder collects and records spans to a remote Appdash collector.
    """
    def __init__(self, collector):
        self._collector = collector

    def record_span(self, span):
        if not span.context.sampled:
            return
        span_id = SpanID()
        span_id.trace = span.context.trace_id
        span_id.span = span.context.span_id
        if span.parent_id is not None:
            span_id.parent = span.parent_id

        self._collector.collect(span_id,
                *event.MarshalEvent(event.SpanNameEvent(span.operation_name)))

        approx_endtime = span.start_time + span.duration
        self._collector.collect(span_id,
                *event.MarshalEvent(event.TimespanEvent(span.start_time, approx_endtime)))

        if span.tags is not None:
            for key in span.tags:
                self._collector.collect(span_id, Annotation(key, span.tags[key]))

        if span.context.baggage is not None:
            for key in span.context.baggage:
                self._collector.collect(span_id, Annotation(key, span.contex.baggage[key]))


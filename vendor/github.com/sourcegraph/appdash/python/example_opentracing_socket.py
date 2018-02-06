#!/usr/bin/env python

# Imports
import time
import appdash

# Appdash: Socket Collector
from appdash.sockcollector import RemoteCollector

# Create a remote appdash collector.
collector = RemoteCollector(debug=True)
collector.connect(host="localhost", port=7701)

# Create a tracer
tracer = appdash.create_new_tracer(collector)

for i in range(0, 7):
    # Generate a few spans with some annotations.
    span = None
    # Name the span.
    if i == 0:
        span = tracer.start_span("Request")
    else:
        span = tracer.start_span("SQL Query")

    span.set_tag("query", "SELECT * FROM table_name;")
    span.set_tag("foo", "bar")

    if i % 2 == 0:
        span.log_event("Hello world!")

    child_span = tracer.start_span("child", child_of=span)
    child_span.finish()

    span.finish(finish_time=time.time()+2)

# Close the collector's connection.
collector.close()

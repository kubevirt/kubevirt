# PerfScale Thresholds
PerfScale thresholds are per release thresholds used by CI to evaluate the code
base for performance and scale.  The list of thresholds is extensible but it
must meet certain criteria.

### Adding Thresholds
In order to add a threshold, the threshold must be:
  - Based on a metric exported to Prometheus
  - Testable
  - Consistent with other thresholds
  - Precise and well-defined

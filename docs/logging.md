# Logging

For logging use the `Log` struct provided in the `log` package (see `pkg/log`).
That struct offers several functions for providing logs with different log levels, verbosity and content.

## Log Levels and Verbosity

The Log struct offers several functions for logging with a decent log level: `Info(msg)`, `Warning(msg)`, `Error(msg)`, `Critical(msg)`.
These functions also exist in a second flavor, where you can use a formatted string with additional arguments.

Additionally you can set the verbosity with `V(verbosity)`, which takes effect on info level messages.

Please check the [Kubernetes Logging Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md) for the meaning of the different log levels and verbosity.

During runtime you will see:

- all info logs with a verbosity equal or lower to verbosity set by the `-v` command line flag
- all warning, error and critical logs

## Default values

If you don't set a verbosity for your info log statements, they will use the default value of `2`.

Also, if you don't provide a `-v` command line flag, it will use a default of `2`.

## Enhancing logs

You can enhance the log statements with some helper functions:

- `Object(o)`: `o` has to be a Kubernetes resource, this will log the name, namespace, kind and uuid of the resource
- `With(...keyvals)`: logs the given key / value pairs
- `Reason(err)`: short for `With("reason", err)`
- `Key(name, kind)`: short for `With("name", name, "kind", kind)`, where given name can be in format `namespace/name`
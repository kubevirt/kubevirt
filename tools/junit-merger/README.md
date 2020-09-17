= JUnit Merger =

This tool helps merging test results from parallel and serial executions to provide a common overview.
The parallel and the serial execution skip tests which the other one executes. As a result naively merging
the junit files leads to an insane amount of skipped tests. One location where this occurs is in Deck from prow.

The tool will do the following things which need to be extra pointed out:
 * Skipped tests are de-duplicated and their execution time gets merged
 * The tool fails if some tests are run in more than one provided junit file

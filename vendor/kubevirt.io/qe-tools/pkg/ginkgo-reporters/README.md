# Ginkgo reporters

These reporters will build the xunit xml


## Polarion reporter
This reporter fills in the xunit file the needed fields in order to upload it into Polarion as test run

#### Required parameters:
- --polarion-execution=true to enable the reporter
- --project-id="QE" will be set under 'properties'
- --polarion-custom-plannedin="QE_1_0" will be set under 'properties'
- --test-tier="tier1" will be set under 'properties'

#### Optional parameters:
- --polarion-report-file the output file will be generated under working directory, the default is polarion_results.xml

### Usage

Include the reporter in the tests entry point
```
if ginkgo_reporters.Polarion.Run {
		reporters = append(reporters, &ginkgo_reporters.Polarion)
	}
```

when executing the tests, in addition to your regular execution parameters,
add the reporter parameters as specified above

``` bash
go test YOUR_PARAMS --polarion-execution=true --project-id="QE" --polarion-custom-plannedin="QE_1_0" --test-tier="tier1" --polarion-report-file="polarion.xml"
```
Will generate `polarion.xml` file under the work directory that can be imported into polarion.

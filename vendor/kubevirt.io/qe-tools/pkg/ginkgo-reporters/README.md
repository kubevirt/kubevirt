# Ginkgo reporters

These reporters will build the xunit xml


## Polarion reporter
This reporter fills in the xunit file the needed fields in order to upload it into Polarion as test run

#### Required parameters:
- --polarion-execution=true to enable the reporter
- --project-id="QE" will be set under 'properties'
- --polarion-custom-plannedin="QE_1_0" will be set under 'properties'

#### Optional parameters:
- --polarion-report-file the output file will be generated under working directory, the default is polarion_results.xml
- --test-suite-params="OS=EL8 Storage=NFS Arch=x86" will be set under 'properties' and the values will get concatenated to the test run name 
- --test-id-prefix="PREFIX" will set "PREFIX" for each test ID in test properties, if this parameter is not passed, the project ID parameter is set to be that prefix by default
- --test-run-template="Existing template name" will create the test run from an existing template
- --test-run-title="Title" will set the test run title

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
go test YOUR_PARAMS --polarion-execution=true --project-id="QE" --polarion-custom-plannedin="QE_1_0" --polarion-report-file="polarion.xml"
```
Will generate `polarion.xml` file under the work directory that can be imported into polarion.

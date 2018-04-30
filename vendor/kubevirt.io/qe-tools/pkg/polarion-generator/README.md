# Polarion test cases generator

This tool will parse tests files in ginkgo format and extract
- Title - generated from concatenation of `Describe`, `Context`, `When`, `Specify` and `It`
- Description - generated from concatenation of `Describe`, `Context`, `When`, `Specify` and `It`
- Steps - generated from `By`
- Additional custom fields

### Usage
```bash
make
polarion-generator --tests-dir=tests/ --output-file=polarion.xml --project-id=QE
```
It will generate `polarion.xml` file under the work directory that can be imported into polarion.

### Limitations

Because generator use static analysis of AST, it creates number of limitations
- can not parse `By` in methods outside of main test `Describe` scope
- can not parse calls to methods under the `By`, for example
`By(fmt.Sprintf("%s step", "test"))` will not generate test step
- it will not parse steps from method, if the method was define after the call

### Additional custom fields for a test

You can automatically generate additional test custom fields like `importance` or `positive`,
you just need to create relevant polarion comment under test case.
```
...
It("should work", func() {
    // +polarion:caseimportance=critical
    // +polarion:caseposneg=positive
    ...
})
```

Custom fields

Name | Supported Values
--- | --- 
caseimportance | critical, high, medium, low
caseposneg | positive, negative

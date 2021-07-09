# GOSEC static analysis 
## Background
Gosec (https://github.com/securego/gosec) is a static code analysis tool that scans the code to find security related issues.

Gosec executes a set of validation tests/rules and produces a report listing all the detected potential violations. The set of rules can be configured as well as include and exclude directories. 

## Running gosec in your local environment
To perform gosec scanning over your code run:
```
    make gosec
```
This command will execute all the tests. If you want to run a specific gosec test you can use the `GOSEC` environment variable: E.g.,  `GOSEC="G601`"

## Handle false positive warnings
As a static analysis tool, gosec may produce false positive warnings - identify a potential risk that is not relevant/applicable in the specific KubeVirt code. 
In extreme cases you can annotate the code with a special comment:
```
// #nosec <description>
```
This will instruct gosec to suppress the warning.
The developer is encouraged to avoid this annotation whenever possible as this will **silence all warnings** in the line below the comment, even if the code changes and another, true positive issue is created because of that.

### How to fix false positive warnnings
Define a new “safe” function that encapsulates the risky function or code
The safe function name should give good indication on when this function should be used. Whenever possible, the function should validate in-code that the condition really applies.
Annotate the call with a `//#nosec ` comment only within this unsafe function implementation

### Example  
The  rule  “G304:  Potential  file  inclusion  via  variable”  warns  about  the  risk  of  unsafe  path  injection  by  an  attacker.  For  example,  when  using  `ioutil.ReadFile(fileName)`  if  an  attacker  can  can inject/change  `fileName`,  then  using  a  combination  of  `/../`  (e.g.,  `fileName="/safe/path/../../private/path"`)  he  might  be  able  to  access  private  files.

Let's look at the following 
```go
func callerExample(){
    ...    
    exampleFunc("/fixed/static/safe/path/file.txt")
    ...
}

func exampleFunc(fileName string){
    ...    
    ioutil.ReadFile(fileName)
    ...
}
```

**Bad fix: silencing the specific warning**
You should avoid annotating the specific line that generated the "false positive" warning  as illustrated here:
```go
func exampleFunc(fileName string){
    ...   
    // #nosec: fileName is a fixed static path and can't be injected 
    ioutil.ReadFile(fileName)
    ...
}
```
The problem is that because the warning was silenced we won't get warning in case in the future the  `callerExample()` function will be changed and pass a filename that is injectable.  
  
**Recommended fix**  
Define  a  new  "safe”  version  of  `ioutil.ReadFile()`  and use it instead of the original function. The new function should validate  that the file path  is  not  risky  (as  much  as  possible)  and  then  call  the  "unsafe"  original  `ioutil.ReadFile()`.  Only  this  call  within  the  "safe"  function  should  be  annotated  with  `//  #nosec` .

```go
func exampleFunc(fileName string){
    ...    
    ValidatePathAndReadFile(fileName)
    ...
}

func ValidatePathAndReadFile(filename  string)  ([]byte,  error) {

    if !string.HasPrefix(filepath.Clean(filename), "/safe/basepath") {
       return nil, fmt.Errorf("unsafe filename")
    }

    // #nosec filename is known to be safe now
    return ioutil.ReadFile(filename)
}
```

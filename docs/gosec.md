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
The developer is encourage to avoid such annotation when possible as this will silence the warnnings in the line below the comment, even if the code changes and another, true positive issue is created because of that.

### Fixing guidelines
Define a new “safe” function that encapsulate the risky function or code
The safe function name should give good indication on when this function should be used. Whenever possible, the function should validate in-code that the condition really applies.
Annotate the call with a `//#nosec ` comment only within this unsafe function implementation

### Example  
The rule “G304: Potential file inclusion via variable” warns about the risk of unsafe path injection by an attacker. For example, when using `ioutil.ReadFile(fileName)` if an attacker can obtian access to inject/change `fileName`, then using a combination of `/../` (e.g., `fileName="/safe/path/../../private/path"`) he might be able to access private files.

**Suggested fix** 
Define a new "safe” version of `ioutil.ReadFile()` that function will validate that `fileName` is not risky (as much as possible) and then call the "unsafe" original `ioutil.ReadFile()`. Only this call within the "safe" function should be annotated with `// #nosec `.

```
func ValidatePathAndReadFile(filename string) ([]byte, error)
		....
	 	 // Validate that the path is not risky, for example by using filepath.Clean(), detect ".." in filename, etc..
		
		....
    	 // #nosec using exec.Command only for non injectable parameters
		 return		 ioutil.ReadFile(filename)
}
```

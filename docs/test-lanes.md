#TEST LANES LABELS

The GPU tests can be run by calling the file automation/test.sh with $TARGET that contains the work "gpu". The Regex is shown below:  
```$TARGET =~ gpu.* ```

For these tests to pass, the TARGET must have atleast one node on the cluster that contains an Nvidia GPU.
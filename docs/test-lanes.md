#TEST LANES LABELS

The GPU tests can be run by calling the file automation/test.sh with $TARGET that contains the word "gpu". The test lane is shown below:  
```$TARGET =~ gpu.* ```

For these tests to pass, the TARGET must have atleast one node on the cluster that contains an Nvidia GPU.

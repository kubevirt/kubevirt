BAZEL_PID=$(bazel info | grep server_pid | cut -d " " -f 2)
while kill -0 $BAZEL_PID 2>/dev/null; do sleep 1; done
# Might not be necessary, just to be sure that exec shutdowns always succeed
# and are not killed by docker.
sleep 1

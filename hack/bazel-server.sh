# Execute unit tests without relying on Bazel.
if [ "$KUBEVIRT_NO_BAZEL" = true ]; then
    sleep infinity
else
    BAZEL_PID=$(bazel info | grep server_pid | cut -d " " -f 2)

    # On developer machines, shut down the bazel server on network changes
    # (e.g. VPN toggle, WiFi reconnect). A stale server won't recover on
    # its own; tearing it down lets hack/dockerized start a fresh one on
    # the next invocation. Skipped in CI where networking is stable.
    if [ "$CI" != "true" ]; then
        (
            initial=$(cat /proc/net/route 2>/dev/null)
            while kill -0 $BAZEL_PID 2>/dev/null; do
                sleep 5
                current=$(cat /proc/net/route 2>/dev/null)
                if [ "$initial" != "$current" ]; then
                    echo "Network change detected, shutting down bazel server..."
                    bazel shutdown
                    break
                fi
            done
        ) &
    fi

    while kill -0 $BAZEL_PID 2>/dev/null; do sleep 1; done
    # Might not be necessary, just to be sure that exec shutdowns always succeed
    # and are not killed by docker.
    sleep 1
fi

DIRS=$(find pkg/*/ -type d)
for dir in $DIRS; do
    # Let's make sure we don't break build by bootstrapping tests in empty
    # directories.
    GOFILES=$(find $dir -maxdepth 1 -type f -name *_test.go | wc -l | xargs)
    if [ "x${GOFILES}" != "x0" ]; then
        # Since ginkgo bootstrap doesn't take path, we have to hop in the
        # target package first.
        (cd $dir && ginkgo bootstrap || :)
    fi
done

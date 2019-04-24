gocli_image="kubevirtci/gocli@sha256:fa7f615a1b07925b27027c57bf09bba0e9874ca92e4f67559556950665598c49"
gocli="docker run --net=host --privileged --rm -v /var/run/docker.sock:/var/run/docker.sock $gocli_image"
gocli_interactive="docker run --net=host --privileged --rm -it -v /var/run/docker.sock:/var/run/docker.sock $gocli_image"

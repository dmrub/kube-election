#!/bin/bash

THIS_DIR="$(dirname "$(readlink -f "$BASH_SOURCE")")"

docker run --rm -v "$THIS_DIR":/go/src/k8s.io/contrib/election \
       -w /go/src/k8s.io/contrib/election golang:1.6 \
       bash -c "rm -f ./server && make server && chown $EUID ./server" && \
    echo "Successfully built election server" || \
        exit

PREFIX=${PREFIX:-leader-elector}
TAG=${TAG:-0.6}
IMAGE_TAG=${PREFIX}:${TAG}

docker build -t ${IMAGE_TAG} "$THIS_DIR" && \
    echo "Successfully built docker image $IMAGE_TAG"

# go get github.com/tools/godep && godep restore -v
# go get github.com/tools/godep
# godep restore -v

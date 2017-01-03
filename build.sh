#!/bin/bash

THIS_DIR="$(dirname "$(readlink -f "$BASH_SOURCE")")"

cleanup() {
    docker rm leader-elector-data
}

trap cleanup INT EXIT

if [ "$1" == "--dind" ]; then
    docker rm leader-elector-data || /bin/true
    docker create -v /go/src/k8s.io/contrib/election --name leader-elector-data golang:1.6 /bin/true && \
        docker cp "$THIS_DIR/." leader-elector-data:/go/src/k8s.io/contrib/election && \
        docker run --rm --volumes-from leader-elector-data -w /go/src/k8s.io/contrib/election golang:1.6 \
               /bin/bash -c "rm -f ./server && make server && chown $EUID ./server" && \
        docker cp leader-elector-data:/go/src/k8s.io/contrib/election/server "$THIS_DIR/" && \
        echo "Successfully built election server" || \
            exit 1
else
    docker run --rm -v "$THIS_DIR":/go/src/k8s.io/contrib/election \
           -w /go/src/k8s.io/contrib/election golang:1.6 \
           bash -c "rm -f ./server && make server && chown $EUID ./server" && \
        echo "Successfully built election server" || \
            exit 1
fi

PREFIX=${PREFIX:-leader-elector}
TAG=${TAG:-0.6}
IMAGE_TAG=${PREFIX}:${TAG}

docker build -t ${IMAGE_TAG} "$THIS_DIR" && \
    echo "Successfully built docker image $IMAGE_TAG"

# go get github.com/tools/godep && godep restore -v
# go get github.com/tools/godep
# godep restore -v

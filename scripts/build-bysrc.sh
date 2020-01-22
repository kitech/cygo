#!/bin/bash

set -e

export GOPATH=$HOME
export GO111MODULE=off

go get -v github.com/thoas/go-funk
go get -v github.com/twmb/algoimpl/go/graph
go get -v golang.org/x/tools/go/ast/astutil
go get -v golang.org/x/tools/go/packages
go get -v github.com/kitech/goplusplus
cp -a $GOPATH/src/github.com/kitech/goplusplus $GOPATH/src/gopp
ln -sv $PWD/go $GOPATH/src

cd bysrc
export CGO_ENABLED=1
go env
go version

go build -v
ls -lh

./bysrc ./tpkgs/cgo1
./utests.sh
for f in `ls ./tpkgs/`; do
    echo "$PWD ./bysrc ./tpkgs/$f"
    # ./bysrc "./tpkgs/$f/"
done

cd -


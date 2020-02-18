#!/bin/bash

set -e

export GOPATH=$HOME
export GO111MODULE=""
export CGO_ENABLED=1

go get -v github.com/thoas/go-funk
go get -v github.com/twmb/algoimpl/go/graph
#go get -v golang.org/x/tools/go/ast/astutil
#go get -v golang.org/x/tools/go/packages
go get -v github.com/smacker/go-tree-sitter
go get -v github.com/xlab/c-for-go
go get -v modernc.org/ir
go get -v modernc.org/token
go get -v github.com/kitech/goplusplus
cp -a $GOPATH/src/github.com/kitech/goplusplus $GOPATH/src/gopp
#ln -sv $PWD $GOPATH/src/
cp -a $PWD $GOPATH/src/
ln -sv $PWD/xgo $GOPATH/src/

cd $GOPATH/src/cxrt/bysrc
pwd

go env
go version
ls

go build -v -a
ls -lh
# build flag -mod=vendor only valid when using modules
#go build -v -mod=vendor
#ls -lh


./bysrc ./tpkgs/cgo1
./utests.sh
for f in `ls ./tpkgs/`; do
    echo "$PWD ./bysrc ./tpkgs/$f"
    # ./bysrc "./tpkgs/$f/"
done

cd -


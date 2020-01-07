#!/bin/bash

git clone https://github.com/ivmai/bdwgc
ls
wget https://github.com/ivmai/libatomic_ops/releases/download/v7.6.10/libatomic_ops-7.6.10.tar.gz
tar xvf libatomic_ops-7.6.10.tar.gz
mv libatomic_ops-7.6.10 bdwgc/libatomic_ops

cd bdwgc
./autogen.sh
./configure --prefix=/usr
make && sudo make install
cd -


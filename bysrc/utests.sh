#!/bin/bash

files=$(ls -d tpkgs/* | grep -v ".go")
for pkg in $files; do
    echo "$PWD ./bysrc ./$pkg"
    ./bysrc ./$pkg/
    #make
    ret=$?
    if [[ $ret != 0 ]]; then
        echo "$PWD $pkg error"
        # break;
    fi
done

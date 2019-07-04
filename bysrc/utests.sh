#!/bin/sh

files=$(ls -d tpkgs/* | grep -v ".go")
for pkg in $files; do
    echo "$pkg"
    ./bysrc $pkg/
    make
    ret=$?
    if [[ $ret != 0 ]]; then
        echo "$pkg error"
        break;
    fi
done

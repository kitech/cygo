#!/bin/bash

files=$(ls -d tpkgs/* | grep -v ".go")

totcnt=0
for pkg in $files; do
    totcnt=$(($totcnt + 1))
done
cnter=0
for pkg in $files; do
    cnter=$(($cnter + 1))
    echo "[$cnter/$totcnt] $PWD ./bysrc ./$pkg"
    ./bysrc ./$pkg/
    #make
    ret=$?
    if [[ $ret != 0 ]]; then
        echo "$PWD $pkg error"
        # break;
    fi
done

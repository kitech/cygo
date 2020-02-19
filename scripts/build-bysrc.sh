#!/bin/bash

set -e

myexe=$0
if [[ $RUNNER_OS == "macOS" ]]; then
    selfdir=$(dirname $(readlink $myexe))
    $selfidr/build-bysrc-macos.sh
elif [[ $RUNNER_OS == "Windows" ]]; then
    # should be mingw env that can use pacman
    $selfdir/build-bysrc-win.sh
else
    selfdir=$(dirname $(readlink -f $myexe))
    $selfdir/build-bysrc-linux.sh
fi


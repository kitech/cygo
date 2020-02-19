#!/bin/bash

# RUNNER_OS=macOS
# RUNNER_OS=Windows
# RUNNER_OS=Linux

myexe=$0
if [[ $RUNNER_OS == "macOS" ]]; then
    selfdir=$(dirname $(readlink $myexe))
    brew install tcc curl
    brew install bdw-gc
    brew install libevent
elif [[ $RUNNER_OS == "Windows" ]]; then
    # should be mingw env that can use pacman
    pacman -S --no-confirm tcc curl
    pacman -S --no-confirm bdw-gc
    pacman -S --no-confirm libevent
else
    sudo apt install -y libxcb1-dev libxcb-xkb-dev libx11-xcb-dev libxcb-cursor-dev libxcb-render0-dev libxkbcommon-x11-dev libxkbcommon-dev libdbus-1-dev libcurl4-openssl-dev
    sudo apt install -y tcc libtcc-dev
    sudo apt remove -y libgc1c2

    selfdir=$(dirname $(readlink -f $myexe))
    $selfdir/build-libgc-8.0.4x.sh
fi

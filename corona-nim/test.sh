# have .gdbinit, direct rand `gdb` or `gdb ./corona`, both ok

set -x
while true; do
    # ./goro1
    gdb ./corona
    ret=$?
    echo "ret=$ret"
    if [[ "$ret" != "0" ]]; then
        break
    fi
    #sleep 1
done

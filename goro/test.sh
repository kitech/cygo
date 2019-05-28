# have .gdbinit, direct rand `gdb` or `gdb ./goro1`, both ok

set -x
while true; do
    # ./goro1
    gdb ./goro1
    ret=$?
    echo "ret=$ret"
    if [[ "$ret" != "0" ]]; then
        break
    fi
    #sleep 1
done

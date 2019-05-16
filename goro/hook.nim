{.compile:"hook.c"}
{.passc:"-D_GNU_SOURCE".}

proc initHook() {.importc.}
initHook()

### hooked transfer to nim scope


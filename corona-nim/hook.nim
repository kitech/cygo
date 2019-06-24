{.compile:"../corona-c/hook.c"}
{.compile:"../corona-c/hookcb.c"}
{.passc:"-D_GNU_SOURCE".}
#{.passc:"-DLIBGO_SYS_Linux".}
{.passc:"-I../corona-c/cltc/include".}
{.passl:"-L../corona-c/cltc/lib -lcollectc"}

proc initHook() {.importc.}
initHook()

### hooked transfer to nim scope


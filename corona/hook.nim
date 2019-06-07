{.compile:"../noro/hook.c"}
{.compile:"../noro/hookcb.c"}
{.passc:"-D_GNU_SOURCE".}
#{.passc:"-DLIBGO_SYS_Linux".}
{.passc:"-I../noro/cltc/include".}
{.passl:"-L../noro/cltc/lib -lcollectc"}

proc initHook() {.importc.}
initHook()

### hooked transfer to nim scope


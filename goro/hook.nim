{.compile:"hook.c"}
{.compile:"hookcb.c"}
{.passc:"-D_GNU_SOURCE".}
{.passc:"-I../noro/cltc/include".}
{.passl:"-L../noro/cltc/lib -lcollectc"}

proc initHook() {.importc.}
initHook()

### hooked transfer to nim scope


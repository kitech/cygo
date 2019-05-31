
# 主要用来编译其他的c文件
# 像是个工程文件了

# 这几个暂时放在对应的封装文件
# {.compile:"../noro/coro.c".}
# {.compile:"../noro/corowp.c".}
# {.compile:"../noro/hook.c".}
# {.compile:"../noro/hookcb.c".}

import os

# absolute path in cflags and ldflags
const abssrcdir = currentSourcePath().splitFile()[0]
const abscflags = "-I " & abssrcdir & "/../noro " &
    " -I " & abssrcdir & "/../noro/include" &
    " -I " & abssrcdir & "/../noro/cltc/include"
const absldflags = "-L " & abssrcdir & "/../bdwgc/.libs" &
    " -L " & abssrcdir & "/../noro/cltc/lib"
{.passc: abscflags .}
{.passl: absldflags .}
{.passc:"-DGC_THREADS".}
{.passl:"-lgc -lpthread".}

{.compile:"../noro/noro.c".}
{.compile:"../noro/norogc.c".}
{.compile:"../noro/noro_util.c".}

{.compile:"../noro/rxilog.c".}
{.compile:"../noro/atomic.c".}
{.compile:"../noro/queue.c".}
{.compile:"../noro/chan.c".}
{.compile:"../noro/hchan.c".}
{.compile:"../noro/hselect.c".}
#{.compile:"../noro/netpoller_ev.c".}
#{.passl:"-lev"}
{.compile:"../noro/netpoller_event.c".}
{.passl:"-levent -levent_pthreads"}
{.passc:"-Wall -std=c11"} # 不管用啊


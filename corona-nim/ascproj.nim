
# 主要用来编译其他的c文件
# 像是个工程文件了

# 这几个暂时放在对应的封装文件
# {.compile:"../corona-c/coro.c".}
# {.compile:"../corona-c/corowp.c".}
# {.compile:"../corona-c/hook.c".}
# {.compile:"../corona-c/hookcb.c".}

import os

# absolute path in cflags and ldflags
const abssrcdir = currentSourcePath().splitFile()[0]
const abscflags = "-I " & abssrcdir & "/../corona-c " &
    " -I " & abssrcdir & "/../bdwgc/include" &
    " -I " & abssrcdir & "/../cltc/include"
const absldflags = "-L " & abssrcdir & "/../bdwgc/.libs" &
    " -L " & abssrcdir & "/../cltc/lib"
{.passc: abscflags .}
{.passl: absldflags .}
{.passc:"-DGC_THREADS".}
{.passl:"-lgc -lpthread".}

{.compile:"../corona-c/corona.c".}
{.compile:"../corona-c/coronagc.c".}
{.compile:"../corona-c/corona_util.c".}
{.compile:"../corona-c/datstu.c".}

{.compile:"../corona-c/rxilog.c".}
{.compile:"../corona-c/futex.c".}
{.compile:"../corona-c/atomic.c".}
{.compile:"../corona-c/szqueue.c".}
{.compile:"../corona-c/chan.c".}
{.compile:"../corona-c/hchan.c".}
{.compile:"../corona-c/hselect.c".}
#{.compile:"../corona-c/netpoller_ev.c".}
#{.passl:"-lev"}
{.compile:"../corona-c/netpoller_event.c".}
{.passl:"-levent -levent_pthreads"}
{.passc:"-Wall -std=c11"} # 不管用啊

#{.passc:"-fstack-usage -finstrument-functions".}
#{.passl:"-Wl,--export-dynamic".}

{.compile:"../corona-c/functrace.c".}

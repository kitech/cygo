
# 主要用来编译其他的c文件
# 像是个工程文件了

{.compile:"../noro/rxilog.c".}
{.compile:"../noro/atomic.c".}
{.compile:"../noro/queue.c".}
{.compile:"../noro/chan.c".}
{.compile:"../noro/hchan.c".}
#{.compile:"../noro/netpoller_ev.c".}
#{.passl:"-lev"}
{.compile:"../noro/netpoller_event.c".}
{.passl:"-levent -levent_pthreads"}
{.passc:"-Wall -std=c11"} # 不管用啊


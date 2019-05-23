
# 主要用来编译其他的c文件

#{.compile:"../noro/netpoller_ev.c".}
#{.passl:"-lev"}
{.compile:"../noro/netpoller_event.c".}
{.passl:"-levent -levent_pthreads"}
{.passc:"-Wall -std=c11"} # 不管用啊



# 主要用来编译其他的c文件

{.compile:"../noro/netpoller.c".}
{.passl:"-lev"}
{.passc:"-Wall -std=c11"} # 不管用啊


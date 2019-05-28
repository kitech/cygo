
### .gdbinit

首先会加载 $HOME/.gdbinit，可以在这设置

比如：添加set auto-load safe-path /

然后在当前目录的 $PWD/.gdbinit，设置当前项目需要的命令

然后执行 gdb

### smash stack
这可能是coro_create没加锁造成的，加了锁之后好像没出现过了。



import macros
import threadpool

template println(v: varargs[untyped]): untyped = echo(v)

proc gettid() : int = getThreadId()

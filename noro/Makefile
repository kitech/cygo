# -DCORO_PTHREAD -DCORO_ASM -DCORO_UCONTEXT
# PTHREAD本来就不是这个目的，应该一般不用。
# lin上能用的也UCONTEXT/SJLJ/ASM
# win上能用的是啥
# GC_THREADS not GC_PTHREADS
CFLAGS=-g -O0 -D_GNU_SOURCE -DCORO_STACKALLOC -DCORO_UCONTEXT -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK -I. -I../goro -I./cltc/include -I./include -DGC_THREADS
LDFLAGS=-L./cltc/lib -lcollectc -L ../bdwgc/.libs -lgc -lpthread -ldl

all:
	gcc ${CFLAGS} -o coro.o -c ../goro/coro.c
	gcc ${CFLAGS} -o corowp.o -c ../goro/corowp.c
	gcc ${CFLAGS} -o hook.o -c ../goro/hook.c
	gcc ${CFLAGS} -o hookcb.o -c ../goro/hookcb.c
	gcc ${CFLAGS} -o noro.o -c noro.c
	gcc ${CFLAGS} -o main.o -c main.c
	gcc -o noro main.o noro.o hookcb.o hook.o corowp.o coro.o ${LDFLAGS}

clean:
	rm -f noro *.o

# some tricks
# gdb: handle SIGXCPU SIG33 SIG35 SIGPWR nostop noprint


# -DCORO_ASM -DCORO_UCONTEXT -DCORO_PTHREAD
# CORO_PTHREAD本来就不是这个目的，应该一般不用(在本项目中不要再用，与mutex hook冲突)。
# lin上能用的也UCONTEXT/SJLJ/ASM
# win上能用的是啥
# GC_THREADS not GC_PTHREADS
# -finstrument-functions
# TODO -fstack-usage not work on macos
# TODO ucontex hard depcreated on macos
MAC_CFLAGS= # -D_XOPEN_SOURCE
LNX_CFLAGS=-fstack-usage
CFLAGS=-g -O0 -std=c11 -D_GNU_SOURCE -DNRDEBUG -DCORO_STACKALLOC -DCORO_UCONTEXT -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK -I. -I../3rdparty/cltc/src -DGC_THREADS ${MAC_CFLAGS} # -finstrument-functions

# -I../bdwgc/include
# -L ../bdwgc/.libs
LDFLAGS=-levent -levent_pthreads -lgc -lpthread -ldl -Wl,--export-dynamic

dotos:
	gcc ${CFLAGS} -o coro.o -c ./coro.c
	gcc ${CFLAGS} -o corowp.o -c ./corowp.c
	gcc ${CFLAGS} -o hook.o -c ./hook.c
	gcc ${CFLAGS} -o hookcb.o -c ./hookcb.c
	gcc ${CFLAGS} -o futex.o -c ./futex.c
	gcc ${CFLAGS} -o corona_util.o -c corona_util.c
	gcc ${CFLAGS} -o rxilog.o -c rxilog.c
	gcc ${CFLAGS} -o atomic.o -c atomic.c
	gcc ${CFLAGS} -o szqueue.o -c szqueue.c
	gcc ${CFLAGS} -o chan.o -c chan.c
	gcc ${CFLAGS} -o hchan.o -c hchan.c
	gcc ${CFLAGS} -o hselect.o -c hselect.c
	#gcc ${CFLAGS} -o netpoller_ev.o -c netpoller_ev.c
	#gcc ${CFLAGS} -o netpoller_event.o -c netpoller_event.c
	gcc ${CFLAGS} -o netpoller_event.o -c netpoller_epoll.c
	gcc ${CFLAGS} -o coronagc.o -c coronagc.c
	gcc ${CFLAGS} -o corona.o -c corona.c
	gcc ${CFLAGS} -o main.o -c main.c
	gcc ${CFLAGS} -o functrace.o -c functrace.c

all: dotos
	gcc -o corona functrace.o main.o corona.o coronagc.o netpoller_event.o hselect.o hchan.o chan.o szqueue.o atomic.o rxilog.o corona_util.o futex.o hookcb.o hook.o corowp.o coro.o ${LDFLAGS}

lcrn: dotos
	ar rcs libcorona.a functrace.o main.o corona.o coronagc.o netpoller_event.o hselect.o hchan.o chan.o szqueue.o atomic.o rxilog.o corona_util.o futex.o hookcb.o hook.o corowp.o coro.o

clean:
	rm -f corona *.o *.su

# some tricks
# gdb: handle SIGXCPU SIG33 SIG35 SIGPWR nostop noprint
# gdb> up/down // call stack up/down
# watch var
# step/next/continue/delete
# list

# https://github.com/ivmai/bdwgc/issuees/154#issuecomment-293088265
# https://github.com/ivmai/bdwgc/issues/152
# ./configure --enable-static=no --enable-shared=yes --enable-cplusplus --enable-gc-debug --enable-gc-assertion --disable-parallel-mark --enable-munmap

# real stack corrupt debug process blog
# https://blogs.unity3d.com/2016/04/25/debugging-memory-corruption-who-the-hell-writes-2-into-my-stack-2/


# corona-c/tests — standalone test suite
# mirrors the root cmake build: libevent netpoller backend, same flags
.DEFAULT_GOAL := all
CC=gcc
ROOT=../..
SRCDIR=..
CFLAGS=-g -O0 -std=c11 -D_GNU_SOURCE -D_XOPEN_SOURCE=600 -DNRDEBUG -DCORO_STACKALLOC -DCORO_UCONTEXT \
  -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK -DGC_THREADS -DLOG_USE_COLOR \
  -I$(SRCDIR) -I$(SRCDIR)/zigsrc -I$(ROOT) -I$(ROOT)/src -I. -I/opt/local/include \
  -I$(ROOT)/3rdparty/cltc/src -I$(ROOT)/3rdparty/cltc/src/include -I$(ROOT)/3rdparty
ifeq ($(shell uname -s),Darwin)
CFLAGS += -D_DARWIN_C_SOURCE
endif
LDFLAGS=-levent -levent_pthreads -lgc -lpthread -ldl -lsigsegv  -L/opt/local/lib

LIBSRCS = coro.c corowp.c iohook.c iohookcb.c iohook2.c futex.c corona_util.c \
  rxilog.c atomic.c datstu.c szqueue.c chan.c hchan.c hselect.c netpoller_event.c \
  coronagc.c corona.c functrace.c creflect.o
LIBOBJS = $(patsubst %.c,%.o,$(filter %.c,$(LIBSRCS)))

creflect.o: $(ROOT)/src/creflect.c
	$(CC) $(CFLAGS) -o $@ -c $<

# Zig C-ABI demo object (same rule as cxrt.cmake)
# -fno-stack-check: clang/ld link path has no Zig compiler-rt (ziglang/zig#6817)
ZIGOBJ = crn_zig_demo.o
ZIG_EXTRA = -fno-stack-check
ifeq ($(shell uname -s),Darwin)
ZIG_EXTRA += -target $(shell uname -m | sed 's/arm64/aarch64/')-macos.11.0
endif
$(ZIGOBJ): $(SRCDIR)/zigsrc/corona_zig.zig
	zig build-obj -fPIC -O Debug -lc $(ZIG_EXTRA) -femit-bin=$@ $<

CLTCDIR=$(ROOT)/3rdparty/cltc/src
CLTCOBJS = array.o hashtable.o treetable.o queue.o deque.o pqueue.o common.o hashset.o

TESTS = t01_spawn t02_pingpong t03_buffered t04_close t05_select t06_timer \
  t07_mutex t08_net t09_stress t10_hammer t11_gcstack t12_sched t13_handshake t14_stats \
  t15_zigdemo t16_zarray

all: $(LIBOBJS) $(CLTCOBJS) creflect.o $(ZIGOBJ) $(TESTS)

$(LIBOBJS): $(wildcard $(SRCDIR)/*.h) t00_common.h

%.o: $(SRCDIR)/%.c
	$(CC) $(CFLAGS) -o $@ -c $<

$(CLTCOBJS): %.o: $(CLTCDIR)/%.c
	$(CC) $(CFLAGS) -o $@ -c $<

$(TESTS): %: %.c $(LIBOBJS) $(CLTCOBJS) creflect.o $(ZIGOBJ) t00_common.h
	$(CC) $(CFLAGS) -o $@ $< $(LIBOBJS) $(CLTCOBJS) creflect.o $(ZIGOBJ) $(LDFLAGS)

clean:
	rm -f $(LIBOBJS) $(CLTCOBJS) $(ZIGOBJ) $(TESTS)

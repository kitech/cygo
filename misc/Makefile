
all: dm1

dm0:
	gcc cogc.c -o cogc0 -g -O0 -L ../bdwgc/.libs -lgc -lpthread -I ../bdwgc/include

dm1:
	gcc cogc1.c ../goro/coro.c -o cogc1 -g -O0 -L ../bdwgc/.libs -lgc -lpthread -I ../bdwgc/include -I ../goro -D_GNU_SOURCE -DCORO_STACKALLOC -DCORO_UCONTEXT -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK


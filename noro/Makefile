CFLAGS=-g -O0 -D_GNU_SOURCE -DCORO_PTHREAD -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK -I. -I../goro -I./cltc/include
LDFLAGS=-L./cltc/lib -lcollectc -lgc -lpthread -ldl

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


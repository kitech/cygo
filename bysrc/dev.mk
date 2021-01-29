# -DIOHOOK_XLIB
# -lX11
all:
	gcc -g -O0 -fPIC -DCORO_ASM -D_GNU_SOURCE -DGC_THREADS -DNRDEBUG -DIOHOOK_XLIB -lX11 -lgc -ldl -latomic -lpthread opkgs/foo.c
	ls -lh a.out

all:
	gcc -g -O0 -fPIC -DCORO_ASM -D_GNU_SOURCE -DGC_THREADS -DNRDEBUG -lgc -ldl -latomic -lpthread opkgs/foo.c
	ls -lh a.out

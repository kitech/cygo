


dmb: cor
	gcc -g -O2 -std=c11 -o cxrtbase.o -c include/cxrtbase.c -I./include
	gcc -g -O2 -std=c11 -o foo.o -c bysrc/opkgs/foo.c -I./include
	g++ -g -O2 -std=c++11 -o dm0 foo.o cxrtbase.o routine.o -L./libgo -llibgo -ldl -lpthread

cor:
	g++ -g -O2 -std=c++11 -o routine.o -c include/routine.cpp -I./libgo/libgo

run:
	LD_LIBRARY_PATH=./libgo ./dm0

clean:
	rm -f *.o


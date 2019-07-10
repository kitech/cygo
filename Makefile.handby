


dmb: # cor
	gcc -g -O0 -std=c11 -D_GNU_SOURCE -o cxrtbase.o -c include/cxrtbase.c -I./include
	gcc -g -O0 -std=c11 -o foo.o -c bysrc/opkgs/foo.c -I./include
	g++ -g -O0 -std=c++11 -o dm0 foo.o cxrtbase.o routine.o \
			-L./libgo -llibgo -lgc -lcord -lgccpp -ldl -lpthread

cor:
	# cmake -DCMAKE_BUILD_TYPE=debug -DBUILD_DYNAMIC=on .
	g++ -g -O0 -std=c++11 -o routine.o -c include/routine.cpp -I./libgo/libgo

run:
	LD_LIBRARY_PATH=./libgo ./dm0

clean:
	rm -f *.o


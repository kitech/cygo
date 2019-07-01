#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>  /* for write(), also available on Windows */

extern "C" void* emulate_cc_new(size_t len) { \
    void *p = malloc(len);
    if (p == 0) {
        /* Don't use stdio (e.g. fputs), because that may want to allocate more
         * memory.
         */
        (void)!write(2, "out of memory\n", 14);
        abort();
    }
    return p;
}
extern "C" void emulate_cc_delete(void* p) {
    if (p != 0)
        free(p);
}


void* operator new  (size_t len) __attribute__((alias("emulate_cc_new")));
void* operator new[](size_t len) __attribute__((alias("emulate_cc_new")));
void  operator delete  (void* p)   __attribute__((alias("emulate_cc_delete")));
void  operator delete[](void* p)   __attribute__((alias("emulate_cc_delete")));
void* __cxa_pure_virtual = 0;

// dynamic_cast
void* __gxx_personality_v0 = 0;

// tests
class TestCppMinrt {
public:
    int a;
    char* p;
    double d;

    TestCppMinrt();
    void hehe();
};

TestCppMinrt::TestCppMinrt() {
}
void TestCppMinrt::hehe() {
}

extern "C"
void* mynewtcm() {
    TestCppMinrt* v = new TestCppMinrt();
    auto i = 3;
    printf("%p\n", v);
    return v;
}

// readme
// http://ptspts.blogspot.com/2010/12/how-to-write-c-program-without-libstdc.html
//
// g++ -fno-rtti -fno-exceptions -o bar.o bar.cpp
// gcc -o bar bar.o ...


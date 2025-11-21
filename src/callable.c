

#include <stdio.h>
#include <stdlib.h>


// can be a free func, or a method with this
// no any captures now
typedef struct {
    void* isclos;
    void* fnptr;
    int  ismth;
    void* fnobj; // can be null, but still use it
} callable;

callable* new_callable(void* fnptr, int ismth, void* fnobj) {
    callable* clos = malloc(sizeof(callable));
    clos->isclos = 0x1;
    clos->fnptr = fnptr;
    clos->ismth = ismth;
    clos->fnobj = fnobj;
    return clos;
}

#define callable_call_noret(anyfn, args...)     \
    {                                           \
        callable* clos = anyfn;                                     \
        void (*fnptr)() = clos->isclos == 0x1 ? clos->fnptr : clos; \
        if (clos->isclos==(void*)1) {                               \
            if (clos->ismth == 1) {                                 \
                fnptr(clos->fnobj, "closobj with mth");             \
            }else{                                                  \
                fnptr("closobj");                                   \
            }                                                       \
        }else{                                                      \
            fnptr("barefn");                                        \
        }                                                           \
    }

void call(void* anyfn) {
    callable* clos = anyfn;
    void (*fnptr)() = clos->isclos == 0x1 ? clos->fnptr : clos;
    if (clos->isclos==(void*)1) {
        if (clos->ismth == 1) {
            fnptr(clos->fnobj, "closobj with mth");
        }else{
            fnptr("closobj");
        }
    }else{
        fnptr("barefn");
    }
}

static void foo(char* from) {
    printf("called hhh %s\n", from);
}

int main_callable_demo(int argc, char**argv) {

    void* nonclosfn = foo;
    void* closfn = new_callable(foo, 0, NULL);

    call(nonclosfn);
    printf("=====\n");
    call(closfn);

    callable_call_noret(nonclosfn, "hhh");

    return 0;
}

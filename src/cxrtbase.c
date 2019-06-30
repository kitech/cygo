
#include <assert.h>
#include <stdarg.h>

#include "cxrtbase.h"

extern void GC_allow_register_threads();
static void cxrt_init_gc_env() {
    GC_set_free_space_divisor(50); // default 3
    GC_INIT();
    GC_allow_register_threads();
}

extern void cxrt_init_routine_env();
void cxrt_init_routine_env() {
    // assert(1==2);
}

void cxrt_init_env() {
    cxrt_init_gc_env();
    cxrt_init_routine_env();
}

void println(const char* fmt, ...) {
    va_list arg;
    int done;

    va_start (arg, fmt);
    done = vprintf (fmt, arg);
    va_end (arg);

    printf("\n");
}
void println2(const char* filename, int lineno, const char* funcname, const char* fmt, ...) {
    const char* fbname = strrchr(filename, '/');
    if (fbname != nilptr) { fbname = fbname + 1; }
    else { fbname = filename; }

    printf("%s:%d:%s ", fbname, lineno, funcname);

    va_list arg;
    int done;

    va_start (arg, fmt);
    done = vprintf (fmt, arg);
    va_end (arg);

    printf("\n");
}
void panic(cxstring*s) {
    if (s != nilptr) {
        printf("%.*s", s->len, s->ptr);
    }else{
        printf("<%p>", s);
    }
    memcpy((void*)0x1, "abc", 3);
}
void panicln(cxstring*s) {
    cxstring* lr = cxstring_new_cstr("\n");
    if (s != nilptr) {
        s = cxstring_add(s, lr);
    } else{
        s = lr;
    }
    panic(s);
}

#include <unistd.h>
#include <sys/syscall.h>

pid_t gettid() {
#ifdef SYS_gettid
    pid_t tid = syscall(SYS_gettid);
    return tid;
#else
#error "SYS_gettid unavailable on this system"
    return 0;
#endif
}

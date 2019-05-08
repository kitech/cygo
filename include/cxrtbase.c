#include "cxrtbase.h"

#include <stdarg.h>

static void cxrt_init_gc_env() { GC_INIT(); }

extern void cxrt_init_routine_env();
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

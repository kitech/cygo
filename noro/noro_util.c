#include <stdlib.h>
#include <stdio.h>
#include <stdarg.h>

#include <unistd.h>
#include <sys/syscall.h>
#include <threads.h>

#include <noro_util.h>

pid_t gettid() {
#ifdef SYS_gettid
    pid_t tid = syscall(SYS_gettid);
    return tid;
#else
#error "SYS_gettid unavailable on this system"
    return 0;
#endif
}

int (array_randcmp) (const void*a, const void*b) {
    int n = rand() % 3;
    return n-1;
}

static mtx_t loglk;
void loglock() {
        mtx_lock(&loglk);
}
void logunlock() {
     mtx_unlock(&loglk);
}

void noro_simlog(int level, const char *filename, int line, const char* funcname, const char *fmt, ...) {
    char* fbname = strrchr(filename, '/');
    if (fbname != NULL) fbname ++;
    struct timeval ltv = {0};
    gettimeofday(&ltv, 0);
    loglock();
    fprintf(stderr, "%ld.%ld %s:%d %s: ", ltv.tv_sec, ltv.tv_usec, fbname, line, funcname);

    va_list args;
    va_start(args, fmt);
    vfprintf(stderr, fmt, args);
    va_end(args);
    fflush(stderr);
    logunlock();
}

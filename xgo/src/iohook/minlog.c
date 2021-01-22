#include "minlog.h"

#include <stdio.h>
#include <string.h>
#include <stdarg.h>
#include <sys/time.h>
#include <unistd.h>
#include <pthread.h>

void __attribute__((no_instrument_function))
crn_simlog(int level, const char *filename, int line, const char* funcname, const char *fmt, ...) {
    // if (level > loglvl) return;
    static __thread char obuf[612] = {0};
    const char* fbname = strrchr(filename, '/');
    fbname = fbname != NULL ? (fbname++) : filename;
    struct timeval ltv = {0};
    gettimeofday(&ltv, 0);
    // crn_loglock();
    int len = snprintf(obuf, sizeof(obuf)-1, "%ld.%ld %s:%d %s: ",
                       ltv.tv_sec, ltv.tv_usec, fbname, line, funcname);

    va_list args;
    va_start(args, fmt);
    len += vsnprintf(obuf+len, sizeof(obuf)-len-1, fmt, args);
    va_end(args);
    obuf[len] = '\0';
    // fprintf(stderr, "%s", buf);
    // fflush(stderr);
    write(STDERR_FILENO, obuf, len);
    // crn_logunlock();
}

#ifndef _NORO_UTIL_H_
#define _NORO_UTIL_H_

#include <sys/time.h>
#include <unistd.h>

pid_t gettid();

int (array_randcmp) (const void*a, const void*b);

void loglock();
void logunlock();

void noro_simlog(int level, const char *filename, int line, const char* funcname, const char *fmt, ...);
void noro_simlog2(int level, const char *filename, int line, const char* funcname, const char *fmt, ...);

#ifdef NRDEBUG
#define SHOWLOG 1
#else
#define SHOWLOG 0
#endif

#define linfo3(fmt, ...)                                                 \
    if (SHOWLOG) {                                                      \
        const char* filename = __FILE__; char* fbname = strrchr(filename, '/'); \
        if (fbname != NULL) fbname ++;                                  \
        struct timeval ltv = {0}; gettimeofday(&ltv, 0);                \
        loglock();                                                      \
        fprintf(stderr, "%ld.%ld %s:%d %s: ", ltv.tv_sec, ltv.tv_usec, fbname, __LINE__, __FUNCTION__); \
        fprintf(stderr, fmt, __VA_ARGS__);                              \
        fflush(stderr); logunlock();                                    \
    }

#define linfo(fmt, ...)                                                 \
    if (SHOWLOG) { noro_simlog(0, __FILE__, __LINE__, __FUNCTION__, fmt, __VA_ARGS__); }

#define linfo2(fmt, ...)                                                 \
    if (SHOWLOG) { noro_simlog2(0, __FILE__, __LINE__, __FUNCTION__, fmt, __VA_ARGS__); }

#endif


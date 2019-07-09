#ifndef _NORO_UTIL_H_
#define _NORO_UTIL_H_

#include <sys/time.h>
#include <unistd.h>

pid_t gettid();

int (array_randcmp) (const void*a, const void*b);

typedef struct rtsettings {
    int loglevel;
    int maxprocs;
    int gcpercent;
    int gctrace;
    int dbggc;
    int dbgsched;
    int dbgchan;
    int dbghook;
    int dbgpoller;
    int dbgthread;
} rtsettings;
extern rtsettings* rtsets;
void crn_loglvl_forenv();


void crn_loglock();
void crn_logunlock();

void crn_simlog(int level, const char *filename, int line, const char* funcname, const char *fmt, ...);
void crn_simlog2(int level, const char *filename, int line, const char* funcname, const char *fmt, ...);


#define LOGLVL_FATAL 0
#define LOGLVL_ERROR 1
#define LOGLVL_WARN 2
#define LOGLVL_INFO 3
#define LOGLVL_DEBUG 4
#define LOGLVL_VERBOSE 5
#define LOGLVL_TRACE 6

#ifdef NRDEBUG
#define SHOWLOG 1
#else
#define SHOWLOG 0
#endif


#define linfo(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_INFO, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define linfo2(fmt, ...)                                                \
    if (SHOWLOG) { crn_simlog2(LOGLVL_INFO, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lfatal(fmt, ...)                                                \
    if (SHOWLOG) { crn_simlog(LOGLVL_FATAL, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lerror(fmt, ...)                                                \
    if (SHOWLOG) { crn_simlog(LOGLVL_ERROR, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lwarn(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_WARN, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define ldebug(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_DEBUG, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lverb(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_VERBOSE, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define ltrace(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_TRACE, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }

// depcreated
#define linfo333(fmt, ...)                                                \
    if (SHOWLOG) {                                                      \
        const char* filename = __FILE__; char* fbname = strrchr(filename, '/'); \
        if (fbname != NULL) fbname ++;                                  \
        struct timeval ltv = {0}; gettimeofday(&ltv, 0);                \
        loglock();                                                      \
        fprintf(stderr, "%ld.%ld %s:%d %s: ", ltv.tv_sec, ltv.tv_usec, fbname, __LINE__, __func__); \
        fprintf(stderr, fmt, __VA_ARGS__);                              \
        fflush(stderr); logunlock();                                    \
    }

#endif


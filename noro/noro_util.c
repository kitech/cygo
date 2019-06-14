#include <stdlib.h>
#include <stdio.h>
#include <stdarg.h>
#include <string.h>
#include <assert.h>

#include <unistd.h>
#include <sys/syscall.h>
#include <threads.h>

#include <yieldtypes.h>
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
    // mtx_lock(&loglk);
}
void logunlock() {
    // mtx_unlock(&loglk);
    // mtx_trylock(&loglk);
}

void noro_simlog(int level, const char *filename, int line, const char* funcname, const char *fmt, ...) {
    static __thread char buf[512] = {0};
    char* fbname = strrchr(filename, '/');
    if (fbname != NULL) fbname ++;
    struct timeval ltv = {0};
    gettimeofday(&ltv, 0);
    loglock();
    int len = snprintf(buf, sizeof(buf)-1, "%ld.%ld %s:%d %s: ", ltv.tv_sec, ltv.tv_usec, fbname, line, funcname);
    // fprintf(stderr, "%ld.%ld %s:%d %s: ", ltv.tv_sec, ltv.tv_usec, fbname, line, funcname);

    va_list args;
    va_start(args, fmt);
    len += vsnprintf(buf+len, sizeof(buf)-len-1, fmt, args);
    // vfprintf(stderr, fmt, args);
    va_end(args);
    buf[len] = '\0';
    fprintf(stderr, "%s", buf);
    fflush(stderr);
    logunlock();
}

// nolock version, used when stopped the world
void noro_simlog2(int level, const char *filename, int line, const char* funcname, const char *fmt, ...) {
    char* fbname = strrchr(filename, '/');
    if (fbname != NULL) fbname ++;
    struct timeval ltv = {0};
    gettimeofday(&ltv, 0);
    // loglock();
    fprintf(stderr, "%ld.%ld %s:%d %s: ", ltv.tv_sec, ltv.tv_usec, fbname, line, funcname);

    va_list args;
    va_start(args, fmt);
    vfprintf(stderr, fmt, args);
    va_end(args);
    fflush(stderr);
    // logunlock();
}


const char* yield_type_name(int ytype) {
    switch (ytype) {
    case YIELD_TYPE_NONE:
        return "none";
    case YIELD_TYPE_CHAN_SEND:
        return "chansend";
    case YIELD_TYPE_CHAN_RECV:
        return "chanrecv";
    case YIELD_TYPE_CHAN_SELECT:
        return "chanselect";
    case YIELD_TYPE_CHAN_SELECT_NOCASE:
        return "chanselectnocase";
    case YIELD_TYPE_CONNECT:
        return "connect";
    case YIELD_TYPE_READ:
        return "read";
    case YIELD_TYPE_READV:
        return "readv";
    case YIELD_TYPE_RECV:
        return "recv";
    case YIELD_TYPE_RECVFROM:
        return "recvfrom";
    case YIELD_TYPE_RECVMSG:
        return "recvmsg";
    case YIELD_TYPE_RECVMSG_TIMEOUT:
        return "recvmsgtimeo";
    case YIELD_TYPE_WRITE:
        return "write";
    case YIELD_TYPE_WRITEV:
        return "writev";
    case YIELD_TYPE_SEND:
        return "send";
    case YIELD_TYPE_SENDTO:
        return "sendto";
    case YIELD_TYPE_SENDMSG:
        return "sendmsg";

    case YIELD_TYPE_POLL:
        return "poll";
    case YIELD_TYPE_UUPOLL:
        return "uupoll";
    case YIELD_TYPE_SELECT:
        return "select";
    case YIELD_TYPE_ACCEPT:
        return "accept";

    case YIELD_TYPE_SLEEP:
        return "sleep";
    case YIELD_TYPE_MSLEEP:
        return "msleep";
    case YIELD_TYPE_USLEEP:
        return "usleep";
    case YIELD_TYPE_NANOSLEEP:
        return "nanosleep";

    case YIELD_TYPE_GETHOSTBYNAMER:
        return "gethostbynamer";
    case YIELD_TYPE_GETHOSTBYNAME2R:
        return "gethostbyname2r";
    case YIELD_TYPE_GETHOSTBYADDR:
        return "gethostbyaddr";
    case YIELD_TYPE_MAX:
        return "max";
    default:
        return "unknown";
    }
}

typedef enum grstate {nostack=0, runnable, executing, waiting, finished, } grstate;
const char* grstate2str(grstate s) {
    switch (s) {
    case nostack: return "nostack";
    case runnable: return "runnable";
    case executing: return "executing";
    case waiting: return "waiting";
    case finished: return "finished";
    default:
        assert(s >= nostack && s <= finished);
    }
}

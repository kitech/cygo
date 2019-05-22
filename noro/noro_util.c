#include <unistd.h>
#include <sys/syscall.h>

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

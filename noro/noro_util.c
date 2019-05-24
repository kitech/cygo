#include <unistd.h>
#include <sys/syscall.h>
#include <stdlib.h>

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

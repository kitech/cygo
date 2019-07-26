// libc non syscall hook

#include "hook2.h"
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <dlfcn.h>

extern void crn_pre_gclock_proc(const char* funcname);
extern void crn_post_gclock_proc(const char* funcname);

void initHook2();

getaddrinfo_t getaddrinfo_f;
pthread_create_t pthread_create_f;

#include "coronapriv.h"

// conflict with GC_pthread_create
int pthread_create_wip(pthread_t *thread, const pthread_attr_t *attr,
                   void *(*start_routine) (void *), void *arg) {
    if (!pthread_create_f) initHook2();
    int rv = pthread_create_f(thread, attr, start_routine, arg);
    return rv;
}


int getaddrinfo_wip(const char *node, const char *service,
                const struct addrinfo *hints,
                struct addrinfo **res)
{
    if (!getaddrinfo_f) initHook2();
    crn_pre_gclock_proc(__func__);
    int rv = getaddrinfo_f(node, service, hints, res);
    crn_post_gclock_proc(__func__);
    return rv;
}


static int doInitHook2() {
    if (getaddrinfo_f) return 0;
    getaddrinfo_f = (getaddrinfo_t)dlsym(RTLD_NEXT, "getaddrinfo");

    return 1;
}

static int isInit2 = 0;
void initHook2()
{
    isInit2 = doInitHook2();
    (void)isInit2;
}

#ifdef STANDALONE_HOOK
void main() {
    int a = socket(1, 1,1);
    printf("a=%d\n", a);
}
#endif

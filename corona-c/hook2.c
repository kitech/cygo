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
    linfo("%d %p\n", thread, pthread_create_f);
    int rv = 0;
    rv = pthread_create_f(thread, attr, start_routine, arg);
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

int getaddrinfo(const char *node, const char *service,
                    const struct addrinfo *hints,
                    struct addrinfo **res)
{
    if (!getaddrinfo_f) initHook2();
    if (!crn_in_procer()) return getaddrinfo_f(node, service, hints, res);
    // linfo("%s **res=%d\n", node, res);

    extern void* netpoller_dnsresolv(const char* hostname, int ytype, fiber* gr, void** out, int* errcode);

    fiber* gr = crn_fiber_getcur();
    const char* hostname = node;
    // struct addrinfo **res2 = calloc(1, sizeof(void*));
    struct addrinfo *oldres = *res;
    struct addrinfo *res2 = nilptr;
    int errcode = 0;
    void* retptr = netpoller_dnsresolv(hostname, YIELD_TYPE_GETADDRINFO, gr, (void**)&res2, &errcode);
    // If we can answer the request immediately (with an error or not!), then we invoke cb immediately and return NULL.
    int imdret = retptr == nilptr;
    // assert(retptr != nilptr && res2 == nilptr);
    if (imdret) {
        if (errcode != 0) {
            assert(res2 == nilptr);
        }else{
            assert(res2 != nilptr);
        }
        *res = res2;
        return errcode;
    }else{
        assert(res2 == nilptr);
    }

    crn_procer_yield((long)node, YIELD_TYPE_GETADDRINFO);

    linfo("%p %d %p %s %s\n", oldres, res2 != nilptr, res2, node, service);
    if (res2 != nilptr) {
        *res = res2;
    }
    return errcode;
}

#include <arpa/inet.h>
static char* lookup_host_impl2 (const char *host)
{
    struct addrinfo hints, *res;
    int errcode;
    char addrstr[100];
    void *ptr;

    memset (&hints, 0, sizeof (hints));
    hints.ai_family = PF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_flags |= AI_CANONNAME;

    errcode = getaddrinfo (host, NULL, &hints, &res);
    if (errcode != 0) {
        perror ("getaddrinfo");
        return 0;
    }

    char* ipaddr = crn_gc_malloc(120);
    linfo ("Host: %s %p\n", host, res);
    while (res) {
        inet_ntop (res->ai_family, res->ai_addr->sa_data, addrstr, 100);

        switch (res->ai_family) {
        case AF_INET:
            ptr = &((struct sockaddr_in *) res->ai_addr)->sin_addr;
            break;
        case AF_INET6:
            ptr = &((struct sockaddr_in6 *) res->ai_addr)->sin6_addr;
            break;
        }
        inet_ntop (res->ai_family, ptr, addrstr, 100);
        linfo ("IPv%d address: %s (%s)\n", res->ai_family == PF_INET6 ? 6 : 4,
                addrstr, res->ai_canonname);

        memcpy(ipaddr, addrstr, strlen(addrstr));
        res = res->ai_next;
    }
    freeaddrinfo(res);

    return ipaddr;
}

char* crn_getaddrinfo(const char* hostname) {
    return lookup_host_impl2 (hostname);
}

static int doInitHook2() {
    if (getaddrinfo_f) return 0;
    getaddrinfo_f = (getaddrinfo_t)dlsym(RTLD_NEXT, "getaddrinfo");
    pthread_create_f = (pthread_create_t)dlsym(RTLD_NEXT, "pthread_create");

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

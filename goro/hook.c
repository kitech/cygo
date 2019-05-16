#include "hook.h"
#include <stdlib.h>
#include <dlfcn.h>
#include <fcntl.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/ioctl.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <assert.h>
#include <stdarg.h>
#include <poll.h>
#if defined(LIBGO_SYS_Linux)
# include <sys/epoll.h>
#elif defined(LIBGO_SYS_FreeBSD)
# include <sys/event.h>
# include <sys/time.h>
#endif

#if defined(LIBGO_SYS_Linux)
# define ATTRIBUTE_WEAK __attribute__((weak))
#elif defined(LIBGO_SYS_FreeBSD)
# define ATTRIBUTE_WEAK __attribute__((weak_import))
#endif

pipe_t pipe_f = NULL;
socket_t socket_f = NULL;
socketpair_t socketpair_f = NULL;
connect_t connect_f = NULL;
read_t read_f = NULL;
readv_t readv_f = NULL;
recv_t recv_f = NULL;
recvfrom_t recvfrom_f = NULL;
recvmsg_t recvmsg_f = NULL;
write_t write_f = NULL;
writev_t writev_f = NULL;
send_t send_f = NULL;
sendto_t sendto_f = NULL;
sendmsg_t sendmsg_f = NULL;
poll_t poll_f = NULL;
select_t select_f = NULL;
accept_t accept_f = NULL;
sleep_t sleep_f = NULL;
usleep_t usleep_f = NULL;
nanosleep_t nanosleep_f = NULL;
close_t close_f = NULL;
fcntl_t fcntl_f = NULL;
ioctl_t ioctl_f = NULL;
getsockopt_t getsockopt_f = NULL;
setsockopt_t setsockopt_f = NULL;
dup_t dup_f = NULL;
dup2_t dup2_f = NULL;
dup3_t dup3_f = NULL;
fclose_t fclose_f = NULL;
#if defined(LIBGO_SYS_Linux)
pipe2_t pipe2_f = NULL;
gethostbyname_r_t gethostbyname_r_f = NULL;
gethostbyname2_r_t gethostbyname2_r_f = NULL;
gethostbyaddr_r_t gethostbyaddr_r_f = NULL;
epoll_wait_t epoll_wait_f = NULL;
#elif defined(LIBGO_SYS_FreeBSD)
#endif

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d %s: ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;

#include "hookcb.h"

int pipe(int pipefd[2])
{
    if (!socket_f) initHook();
    linfo("%d\n", pipefd[0]);

    int rv = pipe_f(pipefd);
    if (rv == 0) {
        hookcb_oncreate(pipefd[0], FDISPIPE, false, 0,0,0);
        hookcb_oncreate(pipefd[1], FDISPIPE, false, 0,0,0);
    }
    return rv;
}
#if defined(LIBGO_SYS_Linux)
int pipe2(int pipefd[2], int flags)
{
    if (!socket_f) initHook();
    linfo("%d\n", flags);

    int rv = pipe2_f(pipefd, flags);
    if (rv == 0) {
        hookcb_oncreate(pipefd[0], FDISPIPE, !!(flags&O_NONBLOCK), 0,0,0);
        hookcb_oncreate(pipefd[1], FDISPIPE, !!(flags&O_NONBLOCK), 0,0,0);
    }
    return rv;
}
#endif
int socket(int domain, int type, int protocol)
{
    if (!socket_f) initHook();
    printf("socket_f=%p\n",socket_f);

    int sock = socket_f(domain, type, protocol);
    if (sock >= 0) {
        hookcb_oncreate(sock, FDISSOCKET, false, domain, type, protocol);
    }
    linfo("task(%s) hook socket, returns %d.\n", "", sock);
    return sock;
}

int socketpair(int domain, int type, int protocol, int sv[2])
{
    if (!socketpair_f) initHook();
    linfo("%d\n", type);

    int rv = socketpair_f(domain, type, protocol, sv);
    if (rv == 0) {
        hookcb_oncreate(sv[0], FDISSOCKET, false, domain, type, protocol);
        hookcb_oncreate(sv[1], FDISSOCKET, false, domain, type, protocol);
    }
    return rv;
}

int connect(int fd, const struct sockaddr *addr, socklen_t addrlen)
{
    if (!connect_f) initHook();
    linfo("%d\n", fd);
}

int accept(int sockfd, struct sockaddr *addr, socklen_t *addrlen)
{
    if (!accept_f) initHook();
    linfo("%d\n", sockfd);
}

ssize_t read(int fd, void *buf, size_t count)
{
    if (!read_f) initHook();
    linfo("%d\n", fd);
    {
        return read_f(fd, buf, count);
    }
    return 0;
}

ssize_t readv(int fd, const struct iovec *iov, int iovcnt)
{
    if (!readv_f) initHook();
    linfo("%d\n", fd);
}

ssize_t recv(int sockfd, void *buf, size_t len, int flags)
{
    if (!recv_f) initHook();
    linfo("%d\n", sockfd);
}

ssize_t recvfrom(int sockfd, void *buf, size_t len, int flags,
        struct sockaddr *src_addr, socklen_t *addrlen)
{
    if (!recvfrom_f) initHook();
    linfo("%d\n", sockfd);
}

ssize_t recvmsg(int sockfd, struct msghdr *msg, int flags)
{
    if (!recvmsg_f) initHook();
    linfo("%d\n", sockfd);
}

ssize_t write(int fd, const void *buf, size_t count)
{
    if (!write_f) initHook();
    linfo("%d %d\n", fd, count);

    {
        return write_f(fd, buf, count);
    }
    return 0;
}

ssize_t writev(int fd, const struct iovec *iov, int iovcnt)
{
    if (!writev_f) initHook();
    linfo("%d\n", fd);
}

ssize_t send(int sockfd, const void *buf, size_t len, int flags)
{
    if (!send_f) initHook();
    linfo("%d\n", sockfd);
}

ssize_t sendto(int sockfd, const void *buf, size_t len, int flags,
        const struct sockaddr *dest_addr, socklen_t addrlen)
{
    if (!sendto_f) initHook();
    linfo("%d\n", sockfd);
}

ssize_t sendmsg(int sockfd, const struct msghdr *msg, int flags)
{
    if (!sendmsg_f) initHook();
    linfo("%d\n", sockfd);
}

int poll_wip(struct pollfd *fds, nfds_t nfds, int timeout)
{
    if (!poll_f) initHook();
    linfo("%d\n", nfds);
}

// ---------------------------------------------------------------------------
// ------ for dns syscall
int __poll(struct pollfd *fds, nfds_t nfds, int timeout)
{
    if (!poll_f) initHook();
    linfo("%d\n", nfds);
}


#if defined(LIBGO_SYS_Linux)
struct hostent* gethostbyname(const char* name)
{
    linfo("%d\n", 1);
    return NULL;
}
int gethostbyname_r(const char *__restrict name,
			    struct hostent *__restrict __result_buf,
			    char *__restrict __buf, size_t __buflen,
			    struct hostent **__restrict __result,
			    int *__restrict __h_errnop)
{
    if (!gethostbyname_r_f) initHook();
    linfo("%d\n", __buflen);
}

struct hostent* gethostbyname2(const char* name, int af)
{
    linfo("%d\n", af);
    return NULL;
}
int gethostbyname2_r(const char *name, int af,
        struct hostent *ret, char *buf, size_t buflen,
        struct hostent **result, int *h_errnop)
{
    if (!gethostbyname2_r_f) initHook();
    linfo("%d\n", af);
}

struct hostent *gethostbyaddr(const void *addr, socklen_t len, int type)
{
    linfo("%d\n", type);
    return NULL;

}
int gethostbyaddr_r(const void *addr, socklen_t len, int type,
        struct hostent *ret, char *buf, size_t buflen,
        struct hostent **result, int *h_errnop)
{
    if (!gethostbyaddr_r_f) initHook();
    linfo("%d\n", type);
}
#endif

// ---------------------------------------------------------------------------

int select_wip(int nfds, fd_set *readfds, fd_set *writefds,
        fd_set *exceptfds, struct timeval *timeout)
{
    if (!select_f) initHook();
    linfo("%d\n", nfds);
}

unsigned int sleep(unsigned int seconds)
{
    if (!sleep_f) initHook();
    linfo("%d\n", seconds);

    {
        sleep_f(seconds);
    }
    return 0;
}

int usleep(useconds_t usec)
{
    if (!usleep_f) initHook();
    linfo("%d\n", usec);

    {
        usleep_f(usec);
    }
    return 0;
}

int nanosleep(const struct timespec *req, struct timespec *rem)
{
    if (!nanosleep_f) initHook();

    {
        return nanosleep_f(req, rem);
    }
    return 0;
}

int close(int fd)
{
    if (!close_f) initHook();
    linfo("%d\n", fd);

    hookcb_onclose(fd);
    {
        return close_f(fd);
    }
    return 0;
}

int __close(int fd)
{
    if (!close_f) initHook();
    linfo("%d\n", fd);

    hookcb_onclose(fd);
    {
        return close_f(fd);
    }
    return 0;
}

int fcntl_wip(int __fd, int __cmd, ...)
{
    if (!fcntl_f) initHook();
    linfo("%d\n", __fd);
    {
        return fcntl_f(__fd, __cmd);
    }
    return 0;
}

int ioctl_wip(int fd, unsigned long int request, ...)
{
    if (!ioctl_f) initHook();
    linfo("%d\n", fd);
}

int getsockopt(int sockfd, int level, int optname, void *optval, socklen_t *optlen)
{
    if (!getsockopt_f) initHook();
    linfo("%d\n", sockfd);
}
int setsockopt(int sockfd, int level, int optname, const void *optval, socklen_t optlen)
{
    if (!setsockopt_f) initHook();
    linfo("%d\n", sockfd);
}

int dup(int oldfd)
{
    if (!dup_f) initHook();
    linfo("%d\n", oldfd);
}
// TODO: support FD_CLOEXEC
int dup2(int oldfd, int newfd)
{
    if (!dup2_f) initHook();
    linfo("%d\n", newfd);
}
// TODO: support FD_CLOEXEC
int dup3(int oldfd, int newfd, int flags)
{
    if (!dup3_f) initHook();
    linfo("%d\n", flags);
}

int fclose(FILE* fp)
{
    if (!fclose_f) initHook();
    int fd = fileno(fp);
    linfo("%p, %d\n", fp, fd);

    hookcb_onclose(fd);
    {
        return fclose_f(fp);
    }
    return 0;
}

#if defined(LIBGO_SYS_Linux)
/*
int epoll_wait(int epfd, struct epoll_event *events, int maxevents, int timeout)
{
    if (!epoll_wait_f) initHook();
    return libgo_epoll_wait(epfd, events, maxevents, timeout);
}
*/
#elif defined(LIBGO_SYS_FreeBSD)
#endif

#if defined(LIBGO_SYS_Linux)
ATTRIBUTE_WEAK extern int __pipe(int pipefd[2]);
ATTRIBUTE_WEAK extern int __pipe2(int pipefd[2], int flags);
ATTRIBUTE_WEAK extern int __socket(int domain, int type, int protocol);
ATTRIBUTE_WEAK extern int __socketpair(int domain, int type, int protocol, int sv[2]);
ATTRIBUTE_WEAK extern int __connect(int fd, const struct sockaddr *addr, socklen_t addrlen);
ATTRIBUTE_WEAK extern ssize_t __read(int fd, void *buf, size_t count);
ATTRIBUTE_WEAK extern ssize_t __readv(int fd, const struct iovec *iov, int iovcnt);
ATTRIBUTE_WEAK extern ssize_t __recv(int sockfd, void *buf, size_t len, int flags);
ATTRIBUTE_WEAK extern ssize_t __recvfrom(int sockfd, void *buf, size_t len, int flags,
        struct sockaddr *src_addr, socklen_t *addrlen);
ATTRIBUTE_WEAK extern ssize_t __recvmsg(int sockfd, struct msghdr *msg, int flags);
ATTRIBUTE_WEAK extern ssize_t __write(int fd, const void *buf, size_t count);
ATTRIBUTE_WEAK extern ssize_t __writev(int fd, const struct iovec *iov, int iovcnt);
ATTRIBUTE_WEAK extern ssize_t __send(int sockfd, const void *buf, size_t len, int flags);
ATTRIBUTE_WEAK extern ssize_t __sendto(int sockfd, const void *buf, size_t len, int flags,
        const struct sockaddr *dest_addr, socklen_t addrlen);
ATTRIBUTE_WEAK extern ssize_t __sendmsg(int sockfd, const struct msghdr *msg, int flags);
ATTRIBUTE_WEAK extern int __libc_accept(int sockfd, struct sockaddr *addr, socklen_t *addrlen);
ATTRIBUTE_WEAK extern int __libc_poll(struct pollfd *fds, nfds_t nfds, int timeout);
ATTRIBUTE_WEAK extern int __select(int nfds, fd_set *readfds, fd_set *writefds,
                          fd_set *exceptfds, struct timeval *timeout);
ATTRIBUTE_WEAK extern unsigned int __sleep(unsigned int seconds);
ATTRIBUTE_WEAK extern int __nanosleep(const struct timespec *req, struct timespec *rem);
ATTRIBUTE_WEAK extern int __libc_close(int);
ATTRIBUTE_WEAK extern int __fcntl(int __fd, int __cmd, ...);
ATTRIBUTE_WEAK extern int __ioctl(int fd, unsigned long int request, ...);
ATTRIBUTE_WEAK extern int __getsockopt(int sockfd, int level, int optname,
        void *optval, socklen_t *optlen);
ATTRIBUTE_WEAK extern int __setsockopt(int sockfd, int level, int optname,
        const void *optval, socklen_t optlen);
ATTRIBUTE_WEAK extern int __dup(int);
ATTRIBUTE_WEAK extern int __dup2(int, int);
ATTRIBUTE_WEAK extern int __dup3(int, int, int);
ATTRIBUTE_WEAK extern int __usleep(useconds_t usec);
ATTRIBUTE_WEAK extern int __new_fclose(FILE *fp);
#if defined(LIBGO_SYS_Linux)
ATTRIBUTE_WEAK extern int __gethostbyname_r(const char *__restrict __name,
			    struct hostent *__restrict __result_buf,
			    char *__restrict __buf, size_t __buflen,
			    struct hostent **__restrict __result,
			    int *__restrict __h_errnop);
ATTRIBUTE_WEAK extern int __gethostbyname2_r(const char *name, int af,
        struct hostent *ret, char *buf, size_t buflen,
        struct hostent **result, int *h_errnop);
ATTRIBUTE_WEAK extern int __gethostbyaddr_r(const void *addr, socklen_t len, int type,
        struct hostent *ret, char *buf, size_t buflen,
        struct hostent **result, int *h_errnop);
ATTRIBUTE_WEAK extern int __epoll_wait_nocancel(int epfd, struct epoll_event *events,
        int maxevents, int timeout);
#elif defined(LIBGO_SYS_FreeBSD)
#endif

// 某些版本libc.a中没有__usleep.
ATTRIBUTE_WEAK int __usleep(useconds_t usec)
{
    struct timespec req = {usec / 1000000, usec * 1000};
    return __nanosleep(&req, NULL);
}
#endif


static int doInitHook()
{
    connect_f = (connect_t)dlsym(RTLD_NEXT, "connect");
    printf("%s:%d, doInitHook %p\n", __FILE__, __LINE__, connect_f);

    if (connect_f) {
        pipe_f = (pipe_t)dlsym(RTLD_NEXT, "pipe");
        socket_f = (socket_t)dlsym(RTLD_NEXT, "socket");
        socketpair_f = (socketpair_t)dlsym(RTLD_NEXT, "socketpair");
        connect_f = (connect_t)dlsym(RTLD_NEXT, "connect");
        read_f = (read_t)dlsym(RTLD_NEXT, "read");
        readv_f = (readv_t)dlsym(RTLD_NEXT, "readv");
        recv_f = (recv_t)dlsym(RTLD_NEXT, "recv");
        recvfrom_f = (recvfrom_t)dlsym(RTLD_NEXT, "recvfrom");
        recvmsg_f = (recvmsg_t)dlsym(RTLD_NEXT, "recvmsg");
        write_f = (write_t)dlsym(RTLD_NEXT, "write");
        writev_f = (writev_t)dlsym(RTLD_NEXT, "writev");
        send_f = (send_t)dlsym(RTLD_NEXT, "send");
        sendto_f = (sendto_t)dlsym(RTLD_NEXT, "sendto");
        sendmsg_f = (sendmsg_t)dlsym(RTLD_NEXT, "sendmsg");
        accept_f = (accept_t)dlsym(RTLD_NEXT, "accept");
        poll_f = (poll_t)dlsym(RTLD_NEXT, "poll");
        select_f = (select_t)dlsym(RTLD_NEXT, "select");
        sleep_f = (sleep_t)dlsym(RTLD_NEXT, "sleep");
        usleep_f = (usleep_t)dlsym(RTLD_NEXT, "usleep");
        nanosleep_f = (nanosleep_t)dlsym(RTLD_NEXT, "nanosleep");
        close_f = (close_t)dlsym(RTLD_NEXT, "close");
        fcntl_f = (fcntl_t)dlsym(RTLD_NEXT, "fcntl");
        ioctl_f = (ioctl_t)dlsym(RTLD_NEXT, "ioctl");
        getsockopt_f = (getsockopt_t)dlsym(RTLD_NEXT, "getsockopt");
        setsockopt_f = (setsockopt_t)dlsym(RTLD_NEXT, "setsockopt");
        dup_f = (dup_t)dlsym(RTLD_NEXT, "dup");
        dup2_f = (dup2_t)dlsym(RTLD_NEXT, "dup2");
        dup3_f = (dup3_t)dlsym(RTLD_NEXT, "dup3");
        fclose_f = (fclose_t)dlsym(RTLD_NEXT, "fclose");
#if defined(LIBGO_SYS_Linux)
        pipe2_f = (pipe2_t)dlsym(RTLD_NEXT, "pipe2");
        gethostbyname_r_f = (gethostbyname_r_t)dlsym(RTLD_NEXT, "gethostbyname_r");
        gethostbyname2_r_f = (gethostbyname2_r_t)dlsym(RTLD_NEXT, "gethostbyname2_r");
        gethostbyaddr_r_f = (gethostbyaddr_r_t)dlsym(RTLD_NEXT, "gethostbyaddr_r");
        epoll_wait_f = (epoll_wait_t)dlsym(RTLD_NEXT, "epoll_wait");
#elif defined(LIBGO_SYS_FreeBSD)
#endif
    } else {
#if defined(LIBGO_SYS_Linux)
        pipe_f = &__pipe;
//        printf("use static hook. pipe_f=%p\n", (void*)pipe_f);
        socket_f = &__socket;
        socketpair_f = &__socketpair;
        connect_f = &__connect;
        read_f = &__read;
        readv_f = &__readv;
        recv_f = &__recv;
        recvfrom_f = &__recvfrom;
        recvmsg_f = &__recvmsg;
        write_f = &__write;
        writev_f = &__writev;
        send_f = &__send;
        sendto_f = &__sendto;
        sendmsg_f = &__sendmsg;
        accept_f = &__libc_accept;
        poll_f = &__libc_poll;
        select_f = &__select;
        sleep_f = &__sleep;
        usleep_f = &__usleep;
        nanosleep_f = &__nanosleep;
        close_f = &__libc_close;
        fcntl_f = &__fcntl;
        ioctl_f = &__ioctl;
        getsockopt_f = &__getsockopt;
        setsockopt_f = &__setsockopt;
        dup_f = &__dup;
        dup2_f = &__dup2;
        dup3_f = &__dup3;
        fclose_f = &__new_fclose;
#if defined(LIBGO_SYS_Linux)
        pipe2_f = &__pipe2;
        gethostbyname_r_f = &__gethostbyname_r;
        gethostbyname2_r_f = &__gethostbyname2_r;
        gethostbyaddr_r_f = &__gethostbyaddr_r;
        epoll_wait_f = &__epoll_wait_nocancel;
#elif defined(LIBGO_SYS_FreeBSD)
#endif
#endif
    }

    if (!pipe_f || !socket_f || !socketpair_f ||
            !connect_f || !read_f || !write_f || !readv_f || !writev_f || !send_f
            || !sendto_f || !sendmsg_f || !accept_f || !poll_f || !select_f
            || !sleep_f|| !usleep_f || !nanosleep_f || !close_f || !fcntl_f || !setsockopt_f
            || !getsockopt_f || !dup_f || !dup2_f || !fclose_f
#if defined(LIBGO_SYS_Linux)
            || !pipe2_f
            || !gethostbyname_r_f
            || !gethostbyname2_r_f
            || !gethostbyaddr_r_f
            || !epoll_wait_f
#elif defined(LIBGO_SYS_FreeBSD)
#endif
            // 老版本linux中没有dup3, 无需校验
            // || !dup3_f
            )
    {
        fprintf(stderr, "Hook syscall failed. Please don't remove libc.a when static-link.\n");
        exit(1);
    }
    return 0;
}

static int isInit = 0;
void initHook()
{
    isInit = doInitHook();
    (void)isInit;
}

#ifdef STANDALONE_HOOK
void main() {
    int a = socket(1, 1,1);
    printf("a=%d\n", a);
}
#endif


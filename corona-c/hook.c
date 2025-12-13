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
#include <errno.h>
#include <stdarg.h>
#include <poll.h>
#if defined(LIBGO_SYS_Linux)
# include <sys/epoll.h>
# include <sys/inotify.h>
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
ppoll_t ppoll_f = NULL;
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
fopen_t fopen_f = NULL;
open_t open_f = NULL;
open64_t open64_f = NULL;
creat_t creat_f = NULL;
openat_t openat_f = NULL;
fdopen_t fdopen_f = NULL;
eventfd_t eventfd_f = NULL;
pmutex_lock_t pmutex_lock_f = NULL;
pmutex_trylock_t pmutex_trylock_f = NULL;
pmutex_unlock_t pmutex_unlock_f = NULL;
pcond_timedwait_t pcond_timedwait_f = NULL;
pcond_wait_t pcond_wait_f = NULL;
pcond_signal_t pcond_signal_f = NULL;
pcond_broadcast_t pcond_broadcast_f = NULL;

#if defined(LIBGO_SYS_Linux)
inotify_init_t inotify_init_f = NULL;
inotify_init1_t inotify_init1_f = NULL;

pipe2_t pipe2_f = NULL;
gethostbyname_r_t gethostbyname_r_f = NULL;
gethostbyname2_r_t gethostbyname2_r_f = NULL;
gethostbyaddr_r_t gethostbyaddr_r_f = NULL;
epoll_wait_t epoll_wait_f = NULL;
#elif defined(LIBGO_SYS_FreeBSD)
#endif


// #include "hookcb.h"
#include "coronapriv.h"

int pipe(int pipefd[2])
{
    if (!socket_f) initHook();
    linfo("%d %p\n", pipefd[0], pipefd);

    int rv = pipe_f(pipefd);
    if (rv == 0) {
        if (crn_in_procer()) {
            hookcb_oncreate(pipefd[0], FDISPIPE, false, 0,0,0);
            hookcb_oncreate(pipefd[1], FDISPIPE, false, 0,0,0);
        }
    }
    return rv;
}
int __pipe(int pipefd[2]) {
    linfo("%d\n", 11233456);
    assert(1==2);
}
#ifndef _GNU_SOURCE
#define _GNU_SOURCE 1
#endif

#if defined(LIBGO_SYS_Linux)
// why not hooked of glib2 use?
int pipe2(int pipefd[2], int flags)
{
    // if (crn_in_procer())
    if (!pipe2_f) initHook();
    // linfo("%d\n", flags);

    int rv = pipe2_f(pipefd, flags);
    if (rv == 0) {
        hookcb_oncreate(pipefd[0], FDISPIPE, !!(flags&O_NONBLOCK), 0,0,0);
        hookcb_oncreate(pipefd[1], FDISPIPE, !!(flags&O_NONBLOCK), 0,0,0);
    }
    return rv;
}
int __pipe2(int pipefd[2], int flags) {
    linfo("%d\n", flags);
    assert(1==2);
}

int inotify_init() {
    if (!inotify_init_f) initHook();
    // linfo("%d\n", 0);
    int rv = inotify_init_f();
    if (rv > 0) {
        hookcb_oncreate(rv, FDISPIPE, 0, 0,0,0);
    }
    return rv;
}
int inotify_init1(int flags) {
    if (!inotify_init1_f) initHook();
    int rv = inotify_init1_f(flags);
    // linfo("%d fd=%d %d %d\n", flags, rv, IN_NONBLOCK, IN_CLOEXEC);
    if (rv > 0) {
        hookcb_oncreate(rv, FDISPIPE, !!(flags&IN_NONBLOCK), 0,0,0);
    }
    return rv;
}
#endif

int socket(int domain, int type, int protocol)
{
    // if (!crn_in_procer()) return;
    if (!socket_f) initHook();
    // linfo("socket_f=%p\n", socket_f);

    int sock = socket_f(domain, type, protocol);
    if (sock >= 0) {
        hookcb_oncreate(sock, FDISSOCKET, false, domain, type, protocol);
        // linfo("task(%s) hook socket, returns %d nb %d.\n", "", sock, fd_is_nonblocking(sock));
        // linfo("domain=%d type=%s(%d) socket=%d\n", domain, type==SOCK_STREAM ? "tcp" : "what", type, sock);
    }

    return sock;
}

int socketpair(int domain, int type, int protocol, int sv[2])
{
    if (!socketpair_f) initHook();
    if (!crn_in_procer()) return socketpair_f(domain, type, protocol, sv);
    // linfo("%d\n", type);

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
    if (!crn_in_procer()) return connect_f(fd, addr, addrlen);
    // linfo("%d\n", fd);

    time_t btime = time(0);
    for (int i = 0;; i++) {
        char buf[200] = {0};
        struct sockaddr_in* sa = (struct sockaddr_in*) addr;
        // linfo("what addr %s\n", inet_ntop(AF_INET, &sa->sin_addr.s_addr, buf, 200));
        // linfo("blocking?? fd=%d %d\n", fd, fd_is_nonblocking(fd));
        if (!fd_is_nonblocking(fd)) {
            lwarn("why fd is blocking mode %d?\n", fd);
            // memcpy(1, 2, 3);
            // hookcb_fd_set_nonblocking(fd, 1);
        }
        assert(fd_is_nonblocking(fd) == 1);
        int rv = connect_f(fd, addr, addrlen);
        // linfo("what addr %s\n", inet_ntop(AF_INET, &sa->sin_addr.s_addr, buf, 200));
        int eno = rv < 0 ? errno : 0;
        if (rv >= 0) {
            // linfo("connect ok %d %d, %d, %d\n", fd, errno, time(0)-btime, i);
            return rv;
        }
        if (eno != EINPROGRESS && eno != EALREADY) {
            linfo("Unknown %d %d %d %d %s\n", fd, rv, eno, i, strerror(eno));
            return rv;
        }
        // linfo("yield %d %d %d\n", fd, rv, eno);
        crn_procer_yield(fd, YIELD_TYPE_CONNECT);
    }
    assert(1==2); // unreachable
}

int accept(int sockfd, struct sockaddr *addr, socklen_t *addrlen)
{
    if (!accept_f) initHook();
    if (!crn_in_procer()) return accept_f(sockfd, addr, addrlen);

    // linfo("%d fdnb=%d\n", sockfd, fd_is_nonblocking(sockfd));
    while(1){
        int rv = accept_f(sockfd, addr, addrlen);
        int eno = rv < 0 ? errno : 0;
        if (rv >= 0) {
            hookcb_oncreate(rv, FDISSOCKET, false, AF_INET, SOCK_STREAM, 0);
            linfo("%d fdnb=%d newfd=%d newnb=%d\n", sockfd, fd_is_nonblocking(sockfd),
                  rv, fd_is_nonblocking(rv));
            return rv;
        }
        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d err=%d eno=%d err=%s\n", sockfd, rv, errno, strerror(errno));
            return rv;
        }
        crn_procer_yield(sockfd, YIELD_TYPE_ACCEPT);
    }
    assert(1==2); // unreachable
}

ssize_t read(int fd, void *buf, size_t count)
{
    if (!read_f) initHook();
    if (!crn_in_procer()) return read_f(fd, buf, count);

    // linfo("%d fdnb=%d bufsz=%d\n", fd, fd_is_nonblocking(fd), count);
    fdcontext* ctx = hookcb_get_fdcontext(fd);
    if (ctx == nilptr) {
        linfo("wtf, why not found fd %d\n", fd);
        assert(1==2);
    }
    bool isfile = fdcontext_is_file(ctx);
    if (fd_is_nonblocking(fd) == 0 && !isfile) {
        linfo("%d fdnb=%d bufsz=%d\n", fd, fd_is_nonblocking(fd), count);
        assert(fd_is_nonblocking(fd) == 1);
    }
    //
    while (1){
        ssize_t rv = read_f(fd, buf, count);
        int eno = rv < 0 ? errno : 0;
        if (rv >= 0) {
            return rv;
        }
        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d err=%d eno=%d err=%s\n", fd, rv, errno, strerror(errno));
            return rv;
        }
        bool inpoll = hookcb_getin_poll(fd, true);
        // linfo("read yeild %d n %d nb %d inpoll %d\n", fd, count, fd_is_nonblocking(fd), inpoll);
        if (inpoll) { // dont yeild inpoll fd read
            // hookcb_setin_poll(fd, false, true); // cannot clear flag, or unexpected yeild/suspend
            return rv;
        }
        crn_procer_yield(fd, YIELD_TYPE_READ);
    }
    assert(1==2); // unreachable
}

ssize_t readv(int fd, const struct iovec *iov, int iovcnt)
{
    if (!readv_f) initHook();
    if (!crn_in_procer()) return readv_f(fd, iov, iovcnt);
    linfo("%d\n", fd);
    assert(1==2);
}

int fd_is_valid(int fd)
{
    return fcntl_f(fd, F_GETFD) != -1 || errno != EBADF;
}
ssize_t recv(int sockfd, void *buf, size_t len, int flags)
{
    if (!recv_f) initHook();
    // linfo("%d %d %d\n", sockfd, len, flags);
    while (1) {
        ssize_t rv = recv_f(sockfd, buf, len, flags);
        int eno = rv < 0 ? errno : 0;
        if (rv == 0) {
            hookcb_onclose(sockfd);
        }
        if (rv >= 0) {
            return rv;
        }
        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d err=%d eno=%d err=%s\n", sockfd, rv, errno, strerror(errno));
            return rv;
        }
        int fdvalid = fd_is_valid(sockfd);
        if (fdvalid != 1) {
            linfo("invalid fd=%d val=%d\n", sockfd, fdvalid);
            assert(fd_is_valid(sockfd) == 1);
        }
        crn_procer_yield(sockfd, YIELD_TYPE_RECV);
    }
    assert(1==2); // unreachable
}

ssize_t recvfrom(int sockfd, void *buf, size_t len, int flags,
        struct sockaddr *src_addr, socklen_t *addrlen)
{
    if (!recvfrom_f) initHook();
    if (!crn_in_procer()) return recvfrom_f(sockfd, buf, len, flags, src_addr, addrlen);

    struct timeval btv = {0};
    struct timeval etv = {0};
    gettimeofday(&btv, 0);
    // linfo("%d %ld.%ld\n", sockfd, btv.tv_sec, btv.tv_usec);
    while (1){
        ssize_t rv = recvfrom_f(sockfd, buf, len, flags, src_addr, addrlen);
        int eno = rv < 0 ? errno : 0;
        gettimeofday(&etv, 0);
        // linfo("%d %ld.%ld\n", sockfd, etv.tv_sec, etv.tv_usec);
        if (rv >= 0) {
            return rv;
        }
        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d rv=%d eno=%d err=%s\n", sockfd, rv, eno, strerror(eno));
            return rv;
        }
        crn_procer_yield(sockfd, YIELD_TYPE_RECVFROM);
    }
    assert(1==2); // unreachable
}

ssize_t recvmsg(int sockfd, struct msghdr *msg, int flags)
{
    if (!recvmsg_f) initHook();
    if (!crn_in_procer()) return recvmsg_f(sockfd, msg, flags);

    // linfo("%d fdnb=%d flags=%d\n", sockfd, fd_is_nonblocking(sockfd), flags);
    assert(fd_is_nonblocking(sockfd)==1);
    time_t btime = time(0);
    for (int i = 0; ; i ++){
        ssize_t rv = recvmsg_f(sockfd, msg, flags);
        int eno = rv < 0 ? errno : 0;
        if (rv >= 0) {
            return rv;
        }

        fdcontext* fdctx = hookcb_get_fdcontext(sockfd);
        bool isudp = fdcontext_is_socket(fdctx) && !fdcontext_is_tcpsocket(fdctx);
        bool isnb = fd_is_nonblocking(sockfd);
        // linfo("fd=%d isudp=%d rv=%d eno=%d err=%s\n", sockfd, isudp, rv, eno, strerror(eno));

        if (eno != EINPROGRESS && eno != EAGAIN && eno != EWOULDBLOCK) {
            linfo("fd=%d fdnb=%d rv=%d eno=%d err=%s\n", sockfd, isnb, rv, eno, strerror(eno));
            return rv;
        }
        // hotfix xlib fd nowait
        if (eno == EAGAIN && fdcontext_get_fdtype(fdctx) == FDXLIB) { return rv; }
        if (isudp) {
            time_t dtime = time(0) - btime;
            if (i > 0 && dtime >= 5) {
                linfo("timedout udpfd=%d, %ld\n", sockfd, dtime);
                return 0; // timedout
            }

            int optval = 0;
            socklen_t optlen = 0;
            int rv2 = getsockopt(sockfd, SOL_SOCKET, SO_ERROR, &optval, &optlen);
            // linfo("opt rv2=%d, len=%d val=%d\n", rv2, optlen, optval);
            int rv3 = getsockopt(sockfd, SOL_SOCKET, SO_RCVTIMEO, &optval, &optlen);
            // linfo("opt rv3=%d, len=%d val=%d\n", rv3, optlen, optval);

            long timeoms = optval > 0 ? optval : 5678;
            // default udp timeout 5s
            long tfds[2] = {0};
            int ytypes[2] = {0};
            tfds[0] = sockfd;
            ytypes[0] = YIELD_TYPE_RECVMSG;
            tfds[1] = timeoms;
            ytypes[1] = YIELD_TYPE_MSLEEP;
            crn_procer_yield_multi(YIELD_TYPE_RECVMSG_TIMEOUT, 2, tfds, ytypes);
        }else{
            // linfo("recvmsg yeild fd=%d isudp=%d rv=%d eno=%d err=%s\n", sockfd, isudp, rv, eno, strerror(eno));
            // assert(1==2);
            crn_procer_yield(sockfd, YIELD_TYPE_RECVMSG);
        }
    }
    assert(1==2); // unreachable
}

ssize_t write(int fd, const void *buf, size_t count)
{
    if (!write_f) initHook();
    if (!crn_in_procer()) return write_f(fd, buf, count);
    if (fd == 1 || fd == 2) return write_f(fd, buf, count);
    // linfo("%d %d\n", fd, count);

    while(1){
        ssize_t rv = write_f(fd, buf, count);
        int eno = rv < 0 ? errno : 0;
        // linfo("wrote fd=%d rv=%d\n", fd, rv);
        if (rv >= 0) {
            return rv;
        }
        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d rv=%d eno=%d err=%s\n", fd, rv, eno, strerror(eno));
            return rv;
        }

        bool inpoll = hookcb_getin_poll(fd, false);
        linfo("write yeild %d n %d nb %d inpoll %d\n", fd, count, fd_is_nonblocking(fd), inpoll);
        crn_procer_yield(fd, YIELD_TYPE_WRITE);
    }
    assert(1==2); // unreachable
}

ssize_t writev(int fd, const struct iovec *iov, int iovcnt)
{
    if (!writev_f) initHook();
    if (!crn_in_procer()) return writev_f(fd, iov, iovcnt);

    int totlen = 0;
    for (int i = 0; i < iovcnt; i++) { totlen += iov[i].iov_len; }
    // linfo("%d %d %d\n", fd, iovcnt, totlen);

    assert(fd_is_nonblocking(fd) == 1);
    while (true) {
        ssize_t rv = writev_f(fd, iov, iovcnt);
        int eno = rv < 0 ? errno : 0;
        if (rv >= 0) {
            assert(rv == totlen);
            return rv;
        }
        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d rv=%d eno=%d err=%s\n", fd, rv, eno, strerror(eno));
            return rv;
        }
        // linfo("writev yield fd=%d rv=%d len=%d\n", fd, rv, totlen);
        crn_procer_yield(fd, YIELD_TYPE_WRITEV);
    }
    assert(1==2);
}

ssize_t send(int sockfd, const void *buf, size_t len, int flags)
{
    if (!send_f) initHook();
    if (!crn_in_procer()) return send_f(sockfd, buf, len, flags);
    // linfo("%d %d %d fdnb=%d\n", sockfd, len, flags, fd_is_nonblocking(sockfd));

    int msgnosig = 0;
#ifdef __APPLE__
#elif _WIN32
#else
    msgnosig = MSG_NOSIGNAL;
#endif

    flags |= msgnosig; // fix SIGPIPE and exit with errro code 141
    while (true) {
        ssize_t rv = send_f(sockfd, buf, len, flags);
        int eno = rv < 0 ? errno : 0;
        if (rv >= 0) {
            assert(rv == len);
            return rv;
        }

        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d rv=%d eno=%d err=%s\n", sockfd, rv, eno, strerror(eno));
            return rv;
        }
        crn_procer_yield(sockfd, YIELD_TYPE_SEND);
    }
    assert(1==2); // unreachable
}

ssize_t sendto(int sockfd, const void *buf, size_t len, int flags,
        const struct sockaddr *dest_addr, socklen_t addrlen)
{
    if (!sendto_f) initHook();
    if (!crn_in_procer()) return sendto_f(sockfd, buf, len, flags, dest_addr, addrlen);
    // linfo("%d %p\n", sockfd, crn_fiber_getcur());

    while(1) {
        int rv = sendto_f(sockfd, buf, len, flags, dest_addr, addrlen);
        if (rv >= 0) {
            return rv;
        }
        linfo("%d len %d %d %s\n", rv, len, errno, strerror(errno));
        return rv;
    }

    assert(1==2);
    return -1;
}

ssize_t sendmsg(int sockfd, const struct msghdr *msg, int flags)
{
    if (!sendmsg_f) initHook();
    if (!crn_in_procer()) return sendmsg_f(sockfd, msg, flags);

    // linfo("%d fdnb=%d\n", sockfd, fd_is_nonblocking(sockfd));
    while (1){
        ssize_t rv = sendmsg_f(sockfd, msg, flags);
        int eno = rv < 0 ? errno : 0;
        if (rv >= 0) {
            return rv;
        }
        if (eno != EINPROGRESS && eno != EAGAIN) {
            linfo("fd=%d rv=%d eno=%d err=%s\n", sockfd, rv, eno, strerror(eno));
            return rv;
        }
        crn_procer_yield(sockfd, YIELD_TYPE_SENDMSG);
    }
    assert(1==2); // unreachable
}


static int getiocinq(int fd)  {
    // fix macos
#ifndef TIOCINQ
#ifdef FIONREAD
#define TIOCINQ FIONREAD
#else
#define TIOCINQ 0x541B
#endif
#endif

    int val = 0;
    int rv = ioctl_f(fd, TIOCINQ, &val);
    if (rv == -1) {
        //         vpp.prtcerr('getiocinq $fd')
    }
    if (rv == -1) {
        // return 0
    }
    // assert rv != -1
    return val;
}


// ---------------------------------------------------------------------------
// ------ for dns syscall
int __poll(struct pollfd fds[], nfds_t nfds, int timeout)
{
    if (!poll_f) initHook();
    if (!crn_in_procer()) return poll_f(fds, nfds, timeout);

    // linfo("%d fd0=%d timeo=%d\n", nfds, fds[0].fd, timeout);
    if (timeout == 0) {  // non-block
        int rv = poll_f(fds, nfds, timeout);
        return rv;
    }

    int nevts = 0;
    for (int i = 0; i < nfds; i ++) {
        if (fds[i].events & POLLIN) { nevts += 1; }
        if (fds[i].events & POLLOUT) { nevts += 1; }
        if (fds[i].events & POLLERR) {  }
        if ((POLLIN | POLLOUT) == fds[i].events ||
            POLLIN == fds[i].events || POLLOUT == fds[i].events) {
        }else{
            linfo("not supported poll event set %d %d\n", POLLIN | POLLOUT, fds[i].events);
        }
        if (fd_is_nonblocking(fds[i].fd) == 0) {
            linfo("blocking socket found %d %d\n", i, fds[i].fd);
        }
    }
    long tfds[nevts+1];
    int tytypes[nevts+1];
    for (int i = 0, j = 0; i < nfds; i ++) {
        if (fds[i].events & POLLIN) {
            tfds[j] = fds[i].fd;
            tytypes[j] = YIELD_TYPE_READ;
            j++;
        }
        if (fds[i].events & POLLOUT) {
            tfds[j] = fds[i].fd;
            tytypes[j] = YIELD_TYPE_WRITE;
            j++;
        }
        // linfo("poll fd %d read %d write %d\n", fds[i].fd, fds[i].events & POLLIN, fds[i].events & POLLOUT);
    }
    int ynfds = nevts;
    if (timeout > 0) {
        tfds[ynfds] = timeout;
        tytypes[ynfds] = YIELD_TYPE_MSLEEP;
        ynfds += 1;
        // linfo("timeout set %d nfds=%d nevts=%d ynfds=%d\n", timeout, nfds, nevts, ynfds);
    }

    for (int i = 0; ; i++) {
        for (int i = 0; i < nfds; i ++) {
            if (fds[i].events&POLLIN) {
                hookcb_setin_poll(fds[i].fd, false, true);
            }
            if (fds[i].events&POLLOUT) {
                hookcb_setin_poll(fds[i].fd, false, false);
            }
        }
        int rv = poll_f(fds, nfds, 0);
        int eno = rv < 0 ? errno : 0;
        int qval = getiocinq(fds[0].fd);
        // linfo("i=%d %d fd0=%d timeo=%d rv=%d qval=%d\n", i, nfds, fds[0].fd, timeout, rv, qval);
        if (rv > 0) {
            // when clear
            for (int i = 0; i < nfds; i ++) {
                if (fds[i].events&POLLIN) {
                    hookcb_setin_poll(fds[i].fd, true, true);
                }
                if (fds[i].events&POLLOUT) {
                    hookcb_setin_poll(fds[i].fd, true, false);
                }
            }
            return rv;
        }
        if (timeout > 0 && i > 0) {
            return 0;
        }
        if (rv < 0) {
            if (eno != EINPROGRESS && eno != EAGAIN) {
                linfo("rv=%d eno=%d err=%s\n", rv, eno, strerror(eno));
                return rv;
            }
        }

        // linfo("poll yeild %d timeo %d rv %d\n", i, timeout, rv);
        int fixyn = i == 0 ? ynfds : (ynfds-1);
        crn_procer_yield_multi(YIELD_TYPE_UUPOLL, fixyn, tfds, tytypes);
    }
    assert(1==2);
}
int poll(struct pollfd fds[], nfds_t nfds, int timeout)
{
    return __poll(fds, nfds, timeout);
}

int ppoll_wip(struct pollfd *fds, nfds_t nfds,
          const struct timespec *tmo_p, const sigset_t *sigmask)
{
    linfo("%d fd0=%d timeo=%d\n", nfds, fds[0].fd, tmo_p);
    assert(1==2);
    if (!crn_in_procer()) return ppoll_f(fds, nfds, tmo_p, sigmask);
    if (!ppoll_f) initHook();
    linfo("%d fd0=%d timeo=%d\n", nfds, fds[0].fd, tmo_p);
    assert(1==2);
}

#if defined(LIBGO_SYS_Linux)
struct hostent* gethostbyname(const char* name)
{
    if (!gethostbyname_r_f) initHook();
    // linfo("%s\n", name);
    if (!crn_in_procer()) {
        static __thread struct hostent host_ = {0};
        static __thread char buf[4096] = {0};
        struct hostent* host = &host_;
        struct hostent* result = 0;
        int herrno = 0;
        int rv = gethostbyname_r(name, host, buf, sizeof(buf), &result, &herrno);
        if (rv == 0 && result == host) {
            return host;
        }
        return 0;
    }

    // below should be fiber vars, not thread vars
    static __thread int bufkey;
    static __thread int reskey;
    struct hostent* result = nilptr;
    struct hostent *host;
    char *buf;
    int herrno = 0;

    buf = crn_fiber_getspec(&bufkey);
    if (buf == nilptr) {
        buf = crn_gc_malloc(5096);
        crn_fiber_setspec(&bufkey, buf);
    }
    host = crn_fiber_getspec(&reskey);
    if (host == nilptr) {
        host = crn_gc_malloc(sizeof(struct hostent));
        crn_fiber_setspec(&reskey, host);
    }
    assert(buf != nilptr); assert(host != nilptr);

    int rv = -1;
    rv = gethostbyname_r(name, host, &buf[0], 4096, &result, &herrno);
    int eno = rv;
    if (rv == 0 && host == result) {
        return host;
    }
    if (rv == ERANGE && herrno == NETDB_INTERNAL) {
        linfo("bufsz too small %s\n", name);
        assert(1==2);
    }
    // linfo("host=%p, result=%p\n", host, result);
    if (eno != EINPROGRESS && eno != EAGAIN) {
        linfo("rv=%d eno=%d err=%s\n", rv, eno, strerror(eno));
    }

    // linfo("rv=%d eno=%d err=%s\n", rv, herrno, strerror(herrno));
    // linfo("rv=%d eno=%d err=%s\n", rv, errno, strerror(errno));
    return 0;
}
int gethostbyname_r(const char *__restrict name,
			    struct hostent *__restrict __result_buf,
			    char *__restrict __buf, size_t __buflen,
			    struct hostent **__restrict __result,
			    int *__restrict __h_errnop)
{
    if (!gethostbyname_r_f) initHook();
    // linfo("%ld\n", __buflen);

    int rv = gethostbyname_r_f(name, __result_buf, __buf, __buflen, __result, __h_errnop);
    int eno = rv == 0 ? 0 : errno;
    // linfo("%s rv=%d eno=%d, err=%d\n", name, rv, eno, strerror(eno));
    // linfo("%s rv=%d eno=%d, err=%d\n", name, rv, *__h_errnop, strerror(*__h_errnop));
    return rv;
}

struct hostent* gethostbyname2(const char* name, int af)
{
    linfo("%d\n", af);
    assert(1==2);
    return NULL;
}
// why this call cannot hooked?
int gethostbyname2_r(const char *name, int af,
        struct hostent *ret, char *buf, size_t buflen,
        struct hostent **result, int *h_errnop)
{
    if (!gethostbyname2_r_f) initHook();
    linfo("%s %d\n", name, af);
    assert(1==2);
}

struct hostent *gethostbyaddr(const void *addr, socklen_t len, int type)
{
    linfo("%d\n", type);
    assert(1==2);
    return NULL;

}
int gethostbyaddr_r(const void *addr, socklen_t len, int type,
        struct hostent *ret, char *buf, size_t buflen,
        struct hostent **result, int *h_errnop)
{
    if (!gethostbyaddr_r_f) initHook();
    linfo("%d\n", type);
    assert(1==2);
}
#endif

// ---------------------------------------------------------------------------

int select(int nfds, fd_set *readfds, fd_set *writefds,
           fd_set *exceptfds, struct timeval *timeout)
{
    if (!select_f) initHook();
    linfo("%d\n", nfds);
    assert(1==2);
}

unsigned int sleep(unsigned int seconds)
{
    if (!sleep_f) initHook();
    if (!crn_in_procer()) return sleep_f(seconds);
    // linfo("%d\n", seconds);

    unsigned int leftsec = seconds;
    time_t btime = time(0);
    while(1){ // maybe resume by some previours timer???
        int rv = crn_procer_yield(leftsec, YIELD_TYPE_SLEEP);
        time_t etime = time(0);
        int dtime = etime-btime;
        if (dtime >= seconds) { return 0; }
        leftsec = seconds - dtime;
        // linfo("leftsec=%d dtime=%d etime=%d btime=%d\n", leftsec, etime-btime, etime, btime);
    }
}

int usleep(useconds_t usec)
{
    if (!crn_in_procer()) return usleep_f(usec);
    if (!usleep_f) initHook();
    // linfo("%d\n", usec);

    time_t btime = time(0);
    {
        int rv = crn_procer_yield(usec, YIELD_TYPE_USLEEP);
        return 0;
    }
}

int nanosleep(const struct timespec *req, struct timespec *rem)
{
    if (!crn_in_procer()) return nanosleep_f(req, rem);
    if (!nanosleep_f) initHook();
    // linfo("%d, %d\n", req->tv_sec, req->tv_nsec);
    {
        long ns = req->tv_sec * 1000000000 + req->tv_nsec;
        int rv = crn_procer_yield(ns, YIELD_TYPE_NANOSLEEP);
        return 0;
    }
}

int close(int fd)
{
    if (!close_f) initHook();
    // linfo("%d\n", fd);

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

int fcntl(int __fd, int __cmd, ...)
{
    if (!fcntl_f) initHook();
    // linfo("%d\n", __fd);

    va_list va;
    va_start(va, __cmd);

    switch (__cmd) {
        // int
        case F_DUPFD:
        case F_DUPFD_CLOEXEC:
            {
                // TODO: support FD_CLOEXEC
                int fd = va_arg(va, int);
                va_end(va);
                int newfd = fcntl_f(__fd, __cmd, fd);
                if (newfd < 0) return newfd;

                hookcb_ondup(__fd, newfd);
                return newfd;
            }

        // int
        case F_SETFD:
        case F_SETOWN:

#if defined(LIBGO_SYS_Linux)
        case F_SETSIG:
        case F_SETLEASE:
        case F_NOTIFY:
#endif

#if defined(F_SETPIPE_SZ)
        case F_SETPIPE_SZ:
#endif
            {
                int arg = va_arg(va, int);
                va_end(va);
                return fcntl_f(__fd, __cmd, arg);
            }

        // int
        case F_SETFL:
            {
                int flags = va_arg(va, int);
                va_end(va);

                bool isNonBlocking = !!(flags & O_NONBLOCK);
                hookcb_fd_set_nonblocking(__fd, isNonBlocking);
                return fcntl_f(__fd, __cmd, flags);
            }

        // struct flock*
        case F_GETLK:
        case F_SETLK:
        case F_SETLKW:
            {
                struct flock* arg = va_arg(va, struct flock*);
                va_end(va);
                return fcntl_f(__fd, __cmd, arg);
            }

        // struct f_owner_ex*
#if defined(LIBGO_SYS_Linux)
        case F_GETOWN_EX:
        case F_SETOWN_EX:
            {
                struct f_owner_exlock* arg = va_arg(va, struct f_owner_exlock*);
                va_end(va);
                return fcntl_f(__fd, __cmd, arg);
            }
#endif

        // void
        case F_GETFL:
            {
                va_end(va);
                return fcntl_f(__fd, __cmd);
            }

        // void
        case F_GETFD:
        case F_GETOWN:

#if defined(LIBGO_SYS_Linux)
        case F_GETSIG:
        case F_GETLEASE:
#endif

#if defined(F_GETPIPE_SZ)
        case F_GETPIPE_SZ:
#endif
        default:
            {
                va_end(va);
                return fcntl_f(__fd, __cmd);
            }
    }
    assert(1==2);
}

int ioctl(int fd, unsigned long int request, ...)
{
    if (!ioctl_f) initHook();
    // linfo("%d\n", fd);

    va_list va;
    va_start(va, request);
    void* arg = va_arg(va, void*);
    va_end(va);

    if (FIONBIO == request) {
        bool isNonBlocking = !!*(int*)arg;
        hookcb_fd_set_nonblocking(fd, isNonBlocking);
    }

    return ioctl_f(fd, request, arg);
}

int getsockopt(int sockfd, int level, int optname, void *optval, socklen_t *optlen)
{
    if (!getsockopt_f) initHook();
    // linfo("%d %d %d\n", sockfd, level, optname);
    {
        int rv = getsockopt_f(sockfd, level, optname, optval, optlen);
        // linfo("%d %d %d ret=%d optlen=%d\n", sockfd, level, optname, rv, *optlen);
        return rv;
    }
}
int setsockopt(int sockfd, int level, int optname, const void *optval, socklen_t optlen)
{
    if (!setsockopt_f) initHook();
    // linfo("%d %d %d\n", sockfd, level, optname);
    {
        int rv = setsockopt_f(sockfd, level, optname, optval, optlen);
        if (rv == 0 && level == SOL_SOCKET) {
            if (optname == SO_RCVTIMEO || optname == SO_SNDTIMEO) {
                linfo("what can i do %d\n", sockfd);
            }
        }
        return rv;
    }
    assert(1==2);
}

int dup(int oldfd)
{
    if (!dup_f) initHook();
    linfo("%d\n", oldfd);
    assert(1==2);
}
// TODO: support FD_CLOEXEC
int dup2(int oldfd, int newfd)
{
    if (!dup2_f) initHook();
    linfo("%d\n", newfd);
    assert(1==2);
}
// TODO: support FD_CLOEXEC
int dup3(int oldfd, int newfd, int flags)
{
    if (!dup3_f) initHook();
    linfo("%d\n", flags);
    assert(1==2);
}

int fclose(FILE* fp)
{
    if (!fclose_f) initHook();
    int fd = fileno(fp);
    // linfo("%p, %d\n", fp, fd);
    return fclose_f(fp);
}
FILE* fopen(const char *pathname, const char *mode)
{
    if (!fopen_f) initHook();
    // if (!crn_in_procer()) return fopen_f(fds, nfds, timeout);
    // linfo("%s %s\n", pathname, mode);

    FILE* fp = fopen_f(pathname, mode);
    // linfo("fopen fp=%p fnlen=%d %s\n", fp, strlen(pathname), pathname);
    if (fp == 0) { return 0; }
    int fd = fileno(fp);
    hookcb_oncreate(fd, FDISFILE, 0, 0,0,0);
    // linfo("%s %s %d fdnb=%d\n", pathname, mode, fd, fd_is_nonblocking(fd));
    return fp;
}
FILE * freopen (const char *filename, const char *opentype, FILE *stream) {
    assert(1==2);
}
FILE * freopen64 (const char *filename, const char *opentype, FILE *stream){
    assert(1==2);
}

int open(const char *pathname, int flags, ...) {
    if (!open_f) initHook();
    // if (!crn_in_procer()) return open_f(fds, nfds, timeout);
    // linfo("%s %d\n", pathname, flags);

    va_list ap;
    mode_t mode;
    va_start(ap, flags);
    mode = va_arg(ap, int);
    va_end(ap);

    int fd = open_f(pathname, flags, mode);
    // linfo("%s %d %d %s\n", pathname, 0, fd, strerror(errno));
    if (fd > 0) {
        hookcb_oncreate(fd, FDISFILE, 0, 0,0,0);
    }
    return fd;
}
int __open(const char *pathname, int flags, mode_t mode) {
    linfo("%s %d\n", pathname, mode);
    return open(pathname, flags, mode);
}
int open64 (const char *filename, int flags, ...) {
    if (!open64_f) initHook();
    // if (!crn_in_procer()) return open_f(fds, nfds, timeout);
    // linfo("%s %d\n", filename, flags);

    va_list ap;
    mode_t mode;
    va_start(ap, flags);
    mode = va_arg(ap, int);
    va_end(ap);

    int fd = open64_f(filename, flags, mode);
    // linfo("%s %d %d %s\n", filename, 0, fd, strerror(errno));
    if (fd > 0) {
        hookcb_oncreate(fd, FDISFILE, 0, 0,0,0);
    }
    return fd;
}

int creat(const char *pathname, mode_t mode) {
    if (!creat_f) initHook();
    // if (!crn_in_procer()) return open_f(fds, nfds, timeout);
    linfo("%s %d\n", pathname, mode);

    int rv = creat_f(pathname,  mode);
    linfo("%s %d %d %s\n", pathname, mode, rv, strerror(errno));
    if (rv > 0) {
        if (strstr(pathname, "fonts")) {
        }else{
            // memcpy(0x1, 0x2, 3);
        }
    }
    return rv;
}
int openat(int dirfd, const char *pathname, int flags, ...) {
    mode_t mode = 022;
    if (!openat_f) initHook();
    // if (!crn_in_procer()) return open_f(fds, nfds, timeout);
    linfo("%d %s %d\n", dirfd, pathname, mode);

    int rv = openat_f(dirfd, pathname, flags,  mode);
    linfo("%s %d %d %s\n", pathname, mode, rv, strerror(errno));
    if (rv > 0) {
        if (strstr(pathname, "fonts")) {
        }else{
            // memcpy(0x1, 0x2, 3);
        }
    }
    return rv;
}
FILE *fdopen(int fd, const char *mode) {
    if (!fdopen_f) initHook();
    // if (!crn_in_procer()) return open_f(fds, nfds, timeout);
    linfo("%d %s\n", fd, mode);
    assert(1==2);

    FILE* rv = fdopen_f(fd, mode);
    int fd2 = -1;
    if (rv != 0) {}
    linfo("%d %s %d %s\n", fd, mode, rv, strerror(errno));
    if (rv != 0) {
    }
    return rv;
}

int eventfd(unsigned int initval, int flags) {
    if (!eventfd_f) initHook();
    // if (!crn_in_procer()) return open_f(fds, nfds, timeout);
    // linfo("%d %d\n", initval, flags);
    if (GC_thread_is_registered() == 0) { // hotfix qt event dispatch thread
        linfo("%d %d not gcreg thread %p\n", initval, flags, pthread_self());
        struct GC_stack_base stkbase;
        GC_get_stack_base(&stkbase);
        GC_register_my_thread(&stkbase);
    }

    int rv = eventfd_f(initval, flags);
    // linfo("%d %d\n", initval, flags);
    int fd2 = -1;
    if (rv != 0) {}
    // linfo("%d %d %d %s\n", initval, flags, rv, strerror(errno));
    if (rv != 0) {
        hookcb_oncreate(rv, EVENTFD, 0, 0,0,0);
    }
    return rv;
}

#define CRN_HOOK_PTHREAD
#ifdef CRN_HOOK_PTHREAD
int pthread_create111(pthread_t *thread, const pthread_attr_t *attr,
                   void *(*start_routine) (void *), void *arg) {
    linfo("ooo %d\n", 123);
}
#endif

#define CRN_HOOK_MUTEX
#ifdef CRN_HOOK_MUTEX
int pthread_mutex_lock_wip(pthread_mutex_t *mutex)
{
    if (!pmutex_lock_f) initHook();
    ldebug("mtx=%p\n", mutex);
    assert(1==2);
}
int pthread_mutex_trylock_wip(pthread_mutex_t *mutex)
{
    if (!pmutex_trylock_f) initHook();
    ldebug("mtx=%p\n", mutex);
    assert(1==2);
}
int pthread_mutex_unlock_wip(pthread_mutex_t *mutex)
{
    if (!pmutex_unlock_f) initHook();
    ldebug("mtx=%p\n", mutex);
    assert(1==2);
}
int pthread_cond_timedwait_wip(pthread_cond_t *restrict cond,
                           pthread_mutex_t *restrict mutex,
                           const struct timespec *restrict abstime)
{
    if (!pcond_timedwait_f) initHook();
    ldebug("mtx=%p\n", mutex);
    assert(1==2);
}
int pthread_cond_wait_wip(pthread_cond_t *restrict cond,
                      pthread_mutex_t *restrict mutex)
{
    if (!pcond_wait_f) initHook();
    ldebug("mtx=%p\n", mutex);
    assert(1==2);
}
int pthread_cond_broadcast_wip(pthread_cond_t *cond)
{
    if (!pcond_broadcast_f) initHook();
    ldebug("mtx=%p\n", cond);
    assert(1==2);
}
int pthread_cond_signal_wip(pthread_cond_t *cond)
{
    if (!pcond_signal_f) initHook();
    ldebug("mtx=%p\n", cond);
    assert(1==2);
}
#endif

#define CRN_HOOK_EPOLL
#ifdef CRN_HOOK_EPOLL
#if defined(LIBGO_SYS_Linux)
// TODO conflict with libevent epoll_wait
int epoll_wait_wip(int epfd, struct epoll_event *events, int maxevents, int timeout)
{
    if (!epoll_wait_f) initHook();
    linfo("epfd=%d maxevents=%d timeout=%d\n", epfd, maxevents, timeout);
    {
        int rv = epoll_wait_f(epfd, events, maxevents, timeout);
        return rv;
    }
    // return libgo_epoll_wait(epfd, events, maxevents, timeout);
}
#endif
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
ATTRIBUTE_WEAK extern int __libc_ppoll(struct pollfd *fds, nfds_t nfds,
                          const struct timespec *tmo_p, const sigset_t *sigmask);
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
    if (connect_f) return 0;
    connect_f = (connect_t)dlsym(RTLD_NEXT, "connect");
    // linfo("%s:%d, doInitHook %p\n", __FILE__, __LINE__, connect_f);
    assert(connect_f != 0);

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
        ppoll_f = (ppoll_t)dlsym(RTLD_NEXT, "ppoll");
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
        fopen_f = (fopen_t)dlsym(RTLD_NEXT, "fopen");
        open_f = (open_t)dlsym(RTLD_NEXT, "open");
        open64_f = (open_t)dlsym(RTLD_NEXT, "open64");
        creat_f = (creat_t)dlsym(RTLD_NEXT, "creat");
        fdopen_f = (fdopen_t)dlsym(RTLD_NEXT, "fdopen");
        eventfd_f = (eventfd_t)dlsym(RTLD_NEXT, "eventfd");
        pmutex_lock_f = (pmutex_lock_t)dlsym(RTLD_NEXT, "pthread_mutex_lock");
        pmutex_trylock_f = (pmutex_trylock_t)dlsym(RTLD_NEXT, "pthread_mutex_trylock");
        pmutex_unlock_f = (pmutex_unlock_t)dlsym(RTLD_NEXT, "pthread_mutex_unlock");
        pcond_timedwait_f = (pcond_timedwait_t)dlsym(RTLD_NEXT, "pthread_cond_timedwait");
        pcond_wait_f = (pcond_wait_t)dlsym(RTLD_NEXT, "pthread_cond_wait");
        pcond_signal_f = (pcond_signal_t)dlsym(RTLD_NEXT, "pthread_cond_signal");
        pcond_broadcast_f = (pcond_broadcast_t)dlsym(RTLD_NEXT, "pthread_cond_broadcast");

#if defined(LIBGO_SYS_Linux)
        inotify_init_f = (inotify_init_t)dlsym(RTLD_NEXT, "inotify_init");
        inotify_init1_f = (inotify_init1_t)dlsym(RTLD_NEXT, "inotify_init1");
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
        ppoll_f = &__libc_ppoll;
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

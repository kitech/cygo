// auto generated with some change: gcc -E -DEV_COMPAT3=0 /usr/include/ev.h

typedef double ev_tstamp;

struct ev_loop;
        
enum {
  EV_UNDEF = (int)0xFFFFFFFF,
  EV_NONE = 0x00,
  EV_READ = 0x01,
  EV_WRITE = 0x02,
  EV__IOFDSET = 0x80,
  EV_IO = EV_READ,
  EV_TIMER = 0x00000100,

  EV_PERIODIC = 0x00000200,
  EV_SIGNAL = 0x00000400,
  EV_CHILD = 0x00000800,
  EV_STAT = 0x00001000,
  EV_IDLE = 0x00002000,
  EV_PREPARE = 0x00004000,
  EV_CHECK = 0x00008000,
  EV_EMBED = 0x00010000,
  EV_FORK = 0x00020000,
  EV_CLEANUP = 0x00040000,
  EV_ASYNC = 0x00080000,
  EV_CUSTOM = 0x01000000,
  EV_ERROR = (int)0x80000000
};

typedef struct ev_watcher {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_watcher *w, int revents);
} ev_watcher;

typedef struct ev_watcher_list {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_watcher_list *w, int revents);
  struct ev_watcher_list *next;
} ev_watcher_list;

typedef struct ev_watcher_time {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_watcher_time *w, int revents);
  ev_tstamp at;
} ev_watcher_time;

typedef struct ev_io {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_io *w, int revents);
  struct ev_watcher_list *next;

  int fd;
  int events;
} ev_io;

typedef struct ev_timer {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_timer *w, int revents);
  ev_tstamp at;

  ev_tstamp repeat;
} ev_timer;

typedef struct ev_periodic {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_periodic *w, int revents);
  ev_tstamp at;

  ev_tstamp offset;
  ev_tstamp interval;
  ev_tstamp (*reschedule_cb)(struct ev_periodic *w, ev_tstamp now);
} ev_periodic;

typedef struct ev_signal {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_signal *w, int revents);
  struct ev_watcher_list *next;

  int signum;
} ev_signal;

typedef struct ev_child {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_child *w, int revents);
  struct ev_watcher_list *next;

  int flags;
  int pid;
  int rpid;
  int rstatus;
} ev_child;

typedef struct stat ev_statdata;

typedef struct ev_stat {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_stat *w, int revents);
  struct ev_watcher_list *next;

  ev_timer timer;
  ev_tstamp interval;
  const char *path;
  ev_statdata prev;
  ev_statdata attr;

  int wd;
} ev_stat;

typedef struct ev_idle {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_idle *w, int revents);
} ev_idle;

typedef struct ev_prepare {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_prepare *w, int revents);
} ev_prepare;

typedef struct ev_check {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_check *w, int revents);
} ev_check;

typedef struct ev_fork {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_fork *w, int revents);
} ev_fork;

typedef struct ev_cleanup {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_cleanup *w, int revents);
} ev_cleanup;

typedef struct ev_embed {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_embed *w, int revents);

  ev_loop *other;
  ev_io io;
  ev_prepare prepare;
  ev_check check;
  ev_timer timer;
  ev_periodic periodic;
  ev_idle idle;
  ev_fork fork;

  ev_cleanup cleanup;

} ev_embed;

typedef struct ev_async {
  int active;
  int pending;
  int priority;
  void *data;
  void (*cb)(ev_loop *loop, struct ev_async *w, int revents);

  sig_atomic_t volatile sent;
} ev_async;

union ev_any_watcher {
  struct ev_watcher w;
  struct ev_watcher_list wl;

  struct ev_io io;
  struct ev_timer timer;
  struct ev_periodic periodic;
  struct ev_signal signal;
  struct ev_child child;

  struct ev_stat stat;

  struct ev_idle idle;

  struct ev_prepare prepare;
  struct ev_check check;

  struct ev_fork fork;

  struct ev_cleanup cleanup;

  struct ev_embed embed;

  struct ev_async async;
};

enum {

  EVFLAG_AUTO = 0x00000000U,

  EVFLAG_NOENV = 0x01000000U,
  EVFLAG_FORKCHECK = 0x02000000U,

  EVFLAG_NOINOTIFY = 0x00100000U,

  EVFLAG_SIGNALFD = 0x00200000U,
  EVFLAG_NOSIGMASK = 0x00400000U
};

enum {
  EVBACKEND_SELECT = 0x00000001U,
  EVBACKEND_POLL = 0x00000002U,
  EVBACKEND_EPOLL = 0x00000004U,
  EVBACKEND_KQUEUE = 0x00000008U,
  EVBACKEND_DEVPOLL = 0x00000010U,
  EVBACKEND_PORT = 0x00000020U,
  EVBACKEND_ALL = 0x0000003FU,
  EVBACKEND_MASK = 0x0000FFFFU
};

int ev_version_major(void);
int ev_version_minor(void);

unsigned int ev_supported_backends(void);
unsigned int ev_recommended_backends(void);
unsigned int ev_embeddable_backends(void);

ev_tstamp ev_time(void);
void ev_sleep(ev_tstamp delay);

void ev_set_allocator(void *(*cb)(void *ptr, long size));

void ev_set_syserr_cb(void (*cb)(const char *msg));

ev_loop *ev_default_loop(unsigned int flags);

static inline ev_loop *ev_default_loop_uc_(void) {
  ev_loop *ev_default_loop_ptr;

  return ev_default_loop_ptr;
}

static inline int ev_is_default_loop(ev_loop *loop) {
  return loop == ev_default_loop_uc_();
}

ev_loop *ev_loop_new(unsigned int flags);

ev_tstamp ev_now(ev_loop *loop);

void ev_loop_destroy(ev_loop *loop);

void ev_loop_fork(ev_loop *loop);

unsigned int ev_backend(ev_loop *loop);

void ev_now_update(ev_loop *loop);

enum { EVRUN_NOWAIT = 1, EVRUN_ONCE = 2 };

enum { EVBREAK_CANCEL = 0, EVBREAK_ONE = 1, EVBREAK_ALL = 2 };

int ev_run(ev_loop *loop, int flags);
void ev_break(ev_loop *loop, int how);

void ev_ref(ev_loop *loop);
void ev_unref(ev_loop *loop);

void ev_once(ev_loop *loop, int fd, int events, ev_tstamp timeout,
                    void (*cb)(int revents, void *arg), void *arg);

unsigned int ev_iteration(ev_loop *loop);
unsigned int ev_depth(ev_loop *loop);
void ev_verify(ev_loop *loop);

void ev_set_io_collect_interval(ev_loop *loop,
                                       ev_tstamp interval);
void ev_set_timeout_collect_interval(ev_loop *loop,
                                            ev_tstamp interval);

void ev_set_userdata(ev_loop *loop, void *data);
void *ev_userdata(ev_loop *loop);
typedef void (*ev_loop_callback)(ev_loop *loop);
void ev_set_invoke_pending_cb(ev_loop *loop,
                                     ev_loop_callback invoke_pending_cb);

void ev_set_loop_release_cb(ev_loop *loop,
                                   void (*release)(ev_loop *loop),
                                   void (*acquire)(ev_loop *loop));

unsigned int ev_pending_count(ev_loop *loop);
void ev_invoke_pending(ev_loop *loop);

void ev_suspend(ev_loop *loop);
void ev_resume(ev_loop *loop);

void ev_feed_event(ev_loop *loop, void *w, int revents);
void ev_feed_fd_event(ev_loop *loop, int fd, int revents);

void ev_feed_signal(int signum);
void ev_feed_signal_event(ev_loop *loop, int signum);

void ev_invoke(ev_loop *loop, void *w, int revents);
int ev_clear_pending(ev_loop *loop, void *w);

void ev_io_start(ev_loop *loop, ev_io *w);
void ev_io_stop(ev_loop *loop, ev_io *w);
void ev_io_set(ev_io *ev, int fd_, int events);

void ev_timer_start(ev_loop *loop, ev_timer *w);
void ev_timer_stop(ev_loop *loop, ev_timer *w);
void ev_timer_set(ev_timer *ev, ev_tstamp after, int repeat);

void ev_timer_again(ev_loop *loop, ev_timer *w);

ev_tstamp ev_timer_remaining(ev_loop *loop, ev_timer *w);

void ev_periodic_start(ev_loop *loop, ev_periodic *w);
void ev_periodic_stop(ev_loop *loop, ev_periodic *w);
void ev_periodic_again(ev_loop *loop, ev_periodic *w);

void ev_signal_start(ev_loop *loop, ev_signal *w);
void ev_signal_stop(ev_loop *loop, ev_signal *w);

void ev_child_start(ev_loop *loop, ev_child *w);
void ev_child_stop(ev_loop *loop, ev_child *w);

void ev_stat_start(ev_loop *loop, ev_stat *w);
void ev_stat_stop(ev_loop *loop, ev_stat *w);
void ev_stat_stat(ev_loop *loop, ev_stat *w);

void ev_idle_start(ev_loop *loop, ev_idle *w);
void ev_idle_stop(ev_loop *loop, ev_idle *w);

void ev_prepare_start(ev_loop *loop, ev_prepare *w);
void ev_prepare_stop(ev_loop *loop, ev_prepare *w);

void ev_check_start(ev_loop *loop, ev_check *w);
void ev_check_stop(ev_loop *loop, ev_check *w);

void ev_fork_start(ev_loop *loop, ev_fork *w);
void ev_fork_stop(ev_loop *loop, ev_fork *w);

void ev_cleanup_start(ev_loop *loop, ev_cleanup *w);
void ev_cleanup_stop(ev_loop *loop, ev_cleanup *w);

void ev_embed_start(ev_loop *loop, ev_embed *w);
void ev_embed_stop(ev_loop *loop, ev_embed *w);
void ev_embed_sweep(ev_loop *loop, ev_embed *w);

void ev_async_start(ev_loop *loop, ev_async *w);
void ev_async_stop(ev_loop *loop, ev_async *w);
void ev_async_send(ev_loop *loop, ev_async *w);

typedef struct ev_loop ev_loop;

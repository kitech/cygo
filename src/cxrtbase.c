
#include <assert.h>
#include <stdarg.h>
#include <execinfo.h>

#include "cxrtbase.h"

// corona
typedef struct corona corona;

extern corona* crn_init_and_wait_done();
extern void crn_post(void(*fn)(void*arg), void*arg);
/* extern void crn_sched(); */
extern void crn_set_finalizer(void*ptr, void(*fn)(void*));
/* typedef struct hchan hchan; */
/* extern hchan* hchan_new(int cap); */
/* extern int hchan_cap(hchan* hc); */
/* extern int hchan_len(hchan* hc); */
/* extern int hchan_send(hchan* hc, void* data); */
/* extern int hchan_recv(hchan* hc, void** pdata); */


extern void GC_allow_register_threads();
static void cxrt_init_gc_env() {
    GC_set_free_space_divisor(50); // default 3
    GC_INIT();
    GC_allow_register_threads();
}

void cxrt_init_routine_env() {
    // assert(1==2);
}

void cxrt_init_env() {
    // cxrt_init_gc_env();
    crn_init_and_wait_done();
}

// TODO simple demo fiber by pthread
typedef struct cxrt_fiber_args {
    void (*fn)(void*);
    void* arg;
} cxrt_fiber_args;
static void* cxrt_fiber_fwdfn(void* varg) {
    cxrt_fiber_args* arg = (cxrt_fiber_args*)varg;
    void (*fiber_fn)(void*) = arg->fn;
    void* fiber_arg = arg->arg;
    cxfree(varg);
    fiber_fn(fiber_arg);
    return nilptr;
}
static void cxrt_fiber_post_pth(void (*fn)(void*), void*arg) {
    pthread_t thr;
    cxrt_fiber_args* fwdarg = cxmalloc(sizeof(cxrt_fiber_args));
    fwdarg->fn = fn;
    fwdarg->arg = arg;
    pthread_create(&thr, nilptr, cxrt_fiber_fwdfn, fwdarg);
}
void cxrt_fiber_post(void (*fn)(void*), void*arg) {
    // cxrt_fiber_post_pth(fn, arg);
    crn_post(fn, arg);
}
void cxrt_set_finalizer(void* ptr,void (*fn) (void*)) {
    crn_set_finalizer(ptr, fn);
}
void* cxrt_chan_new(int sz) {
    /* void* ch = hchan_new(sz); */
    /* assert(ch != nilptr); */
    /* printf("cxrt_chan_new, %p\n", ch); */
    /* return ch; */
    return nilptr;
}
void cxrt_chan_send(void*ch, void*arg) {
    assert(ch != nilptr);
    // hchan_send(ch, arg);
}
void* cxrt_chan_recv(void*ch) {
    assert(ch != nilptr);
    // void* data = nilptr;
    // hchan_recv(ch, &data);
    // return data;
    return nilptr;
}

/////
void printlndep(const char* fmt, ...) {
    va_list arg;
    int done;

    va_start (arg, fmt);
    done = vprintf (fmt, arg);
    va_end (arg);

    printf("\n");
}
void println2(const char* filename, int lineno, const char* funcname, const char* fmt, ...) {
    static __thread char obuf[712] = {0};
    const char* fbname = strrchr(filename, '/');
    if (fbname != nilptr) { fbname = fbname + 1; }
    else { fbname = filename; }

    int len = snprintf(obuf, sizeof(obuf)-1, "%s:%d:%s ", fbname, lineno, funcname);

    va_list arg;
    va_start (arg, fmt);
    len += vsnprintf(obuf+len,sizeof(obuf)-len-1,fmt,arg);
    va_end (arg);
    obuf[len++] = '\n';

    write(STDERR_FILENO, obuf, len);
}

#define MAX_STACK_LEVELS  50
void panic(cxstring*s) {
    if (s != nilptr) {
        printf("%.*s", s->len, s->ptr);
    }else{
        printf("<%p>", s);
    }

    void *buffer[MAX_STACK_LEVELS];
    int levels = backtrace(buffer, MAX_STACK_LEVELS);
    // print to stderr (fd = 2), and remove this function from the trace
    backtrace_symbols_fd(buffer + 1, levels - 1, 2);

    memcpy((void*)0x1, "abc", 3);
    // abort();
    // raise(SIGABRT);
}
void panicln(cxstring*s) {
    cxstring* lr = cxstring_new_cstr("\n");
    if (s != nilptr) {
        s = cxstring_add(s, lr);
    } else{
        s = lr;
    }
    panic(s);
}

#include <unistd.h>
#include <sys/syscall.h>

pid_t cxgettid() {
#ifdef SYS_gettid
    pid_t tid = syscall(SYS_gettid);
    return tid;
#else
#error "SYS_gettid unavailable on this system"
    return 0;
#endif
}


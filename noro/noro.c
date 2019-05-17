
#include <unistd.h>
#include <stdio.h>
#include <assert.h>
#include <pthread.h>

#include <gc/gc.h>
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>

#include <noro.h>

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d:%s ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;

typedef struct coro_stack coro_stack;

typedef enum grstate {nostack=0, runnable, executing, waiting, finished, } grstate;
const int dftstksz = 128*1024;
const int dftstkusz = dftstksz/8; // unit size by sizeof(void*)

// 每个goroutine同时只能属于某一个machine
typedef struct goroutine {
    int id;
    coro_func fnproc;
    void* arg;
    coro_stack stack;
    coro_context coctx;
    coro_context coctx0;
    grstate state;
    int pkstate;
} goroutine;

typedef struct machine {
    int id;
    HashTable* ngrs; // id => goroutine* 新任务，未分配栈
    HashTable* grs;  // # grid => goroutine*
    goroutine* gr;   // 当前在执行的
    pthread_mutex_t pkmu; // pack lock
    pthread_cond_t pkcd;
    bool parking;
} machine;

typedef struct noro {
    int gridno;
    HashTableConf htconf;
    HashTable* mths; // thno => pthread_t*
    HashTable* mcs; // thno => machine*
    bool noroinited;
    pthread_mutex_t noroinitmu;
    pthread_cond_t noroinitcd;

    int eph; // epoll handler
} noro;


///
extern void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze);
extern void corowp_transfer(coro_context *prev, coro_context *next);
extern void corowp_destroy (coro_context *ctx);
extern int corowp_stack_alloc (coro_stack *stack, unsigned int size);
extern void corowp_stack_free(coro_stack* stack);
extern int gettid();

static noro* gnr__ = 0;
HashTableConf* noro_dft_htconf();
int noro_nxtid(noro*nr);


goroutine* noro_goroutine_new(int id, coro_func fn, void* arg) {
    goroutine* gr = (goroutine*)GC_malloc(sizeof(goroutine));
    gr->id = id;
    gr->fnproc = fn;
    gr->arg = arg;
    return gr;
}
// alloc stack and context
void noro_goroutine_new2(goroutine*gr) {
    corowp_create(&gr->coctx0, 0, 0, 0, 0);
    corowp_stack_alloc(&gr->stack, dftstkusz);
    gr->state = runnable;
    // 这一句会让fnproc直接执行，但是可能需要的是创建与执行分开。原来是针对-DCORO_PTHREAD
    // corowp_create(&gr->coctx, gr->fnproc, gr->arg, gr->stack.sptr, dftstksz);
}
void noro_goroutine_forward(goroutine* gr) {
    gr->fnproc(gr->arg);
    corowp_transfer(&gr->coctx, &gr->coctx0);
}
void noro_goroutine_run(goroutine* gr) {
    gr->state = executing;
    // 对-DCORO_PTHREAD来说，这句是真正开始执行
    corowp_create(&gr->coctx, noro_goroutine_forward, gr, gr->stack.sptr, gr->stack.ssze);
    // 对-DCORO_UCONTEXT/-DCORO_ASM等来说，这句是真正开始执行
    corowp_transfer(&gr->coctx0, &gr->coctx);
    // corowp_transfer(&gr->coctx, &gr->coctx0); // 这句要写在函数fnproc退出之前？
    gr->state = finished;
}

machine* noro_machine_new(int id) {
    machine* mc = (machine*)GC_malloc(sizeof(machine));
    mc->id = id;
    linfo("htconf=%o\n", noro_dft_htconf());
    hashtable_new_conf(noro_dft_htconf(), &mc->ngrs);
    hashtable_new_conf(noro_dft_htconf(), &mc->grs);
    return mc;
}
machine* noro_machine_get(int id) {
    machine* mc = 0;
    hashtable_get(gnr__->mcs, (void*)(uintptr_t)id, &mc);
    linfo("get mc %d=%p\n", id, mc);
    return mc;
}
void noro_machine_gradd(machine* mc, goroutine* gr) {
    hashtable_add(mc->grs, (void*)(uintptr_t)gr->id, gr);
}
goroutine* noro_machine_grdel(machine* mc, int id) {
    goroutine* gr = 0;
    hashtable_remove(mc->grs, (void*)(uintptr_t)id, (void**)&gr);
    assert(gr != 0);
    return gr;
}
void noro_machine_signal(machine* mc) {
    pthread_cond_signal(&mc->pkcd);
}

static __thread int gcurmc__ = 0; // thread local
static __thread int gcurgr__ = 0; // thread local
goroutine* noro_goroutine_getcur() {
    int grid = gcurgr__;
    machine* mc1 = noro_machine_get(1);
    goroutine* gr = 0;
    hashtable_get(mc1->grs, (void*)(uintptr_t)grid, (void**)gr);
    assert(gr != 0);
    return gr;
}

void noro_post(coro_func fn, void*arg) {
    int id = noro_nxtid(gnr__);
    goroutine* gr = noro_goroutine_new(id, fn, arg);
    machine* mc = noro_machine_get(1);
    linfo("mc=%p\n", mc);
    hashtable_add(mc->ngrs, (void*)(uintptr_t)id, gr);
    pthread_cond_signal(&mc->pkcd);
}

void noro_processor_setname(int id) {
    char buf[32] = {0};
    snprintf(buf, sizeof(buf), "noro_processor_%d", id);
    pthread_setname_np(pthread_self(), buf);
}
void* noro_processor_netpoller(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d, %d\n", mc->id, gettid());
    noro_processor_setname(mc->id);
    for (;;) {
        sleep(600);
    }
}

void* noro_processor0(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d %d\n", mc->id, gettid());
    noro_processor_setname(mc->id);
    gnr__->noroinited = true;
    pthread_cond_signal(&gnr__->noroinitcd);

    for (;;) {
        int newg = hashtable_size(mc->ngrs);
        if (newg == 0) {
            mc->parking = true;
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
            mc->parking = false;
            newg = hashtable_size(mc->ngrs);
        }

        linfo("newgr %d\n", newg);
        Array* arr1 = 0;
        hashtable_get_keys(mc->ngrs, &arr1);

        for (int i = 0; arr1 != 0 && i < array_size(arr1); i++) {
            void*key = array_peek_at(arr1, i);
            goroutine* gr = 0;
            hashtable_get(mc->ngrs, key, &gr); assert(gr != 0);
            noro_goroutine_new2(gr);
            linfo("process %d, %d\n", gr->id, dftstksz);
            hashtable_remove(mc->ngrs, key, 0);
            noro_machine_gradd(mc, gr);
        }

        // TODO 应该放到schedule中
        // find free machine and runnable goroutine
        Array* arr2 = 0;
        hashtable_get_keys(gnr__->mcs, &arr2);
        for (int i = 0; arr1 != 0 && i < array_size(arr1);) {
            int grid = (int)(uintptr_t)array_peek_at(arr1, i);
            goroutine* gr = 0;
            hashtable_get(mc->grs, (void*)(uintptr_t)grid, (void**)&gr);
            if (gr == 0) {
                linfo("why nil %d, %d\n", grid, hashtable_size(mc->grs));
            }
            assert(gr != 0);

            machine* mct = 0;
            for (int j = 0; arr2!=0 && j < array_size(arr2); j++) {
                void* key = array_peek_at(arr2, j);
                if ((uintptr_t)key <= 2) continue;

                linfo("checking machine %d/%d %d\n", j, array_size(arr2), key);
                hashtable_get(gnr__->mcs, key, &mct); assert(mct != 0);
                if (mct->parking) {
                    linfo("got a packing machine %d <- %d\n", mct->id, gr->id);
                    break;
                }
                mct = 0;
            }
            if (mct == 0) {
                // try select random one
            }
            if (mct == 0) {
                linfo("no enough mc? %d\n", 0);
                break;
            }
            if (mct != 0) {
                int rv = array_remove_at(arr1, i, 0);
                noro_machine_grdel(mc, gr->id);
                noro_machine_gradd(mct, gr);
                noro_machine_signal(mct);
            }
        }
    }
}
void* noro_processor(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d %d\n", mc->id, gettid());
    noro_processor_setname(mc->id);

    for (;;) {
        // check global queue
        machine* mc0 = noro_machine_get(1);
        mc0 = mc; // check outselves
        if (mc0 == 0) {
            mc->parking = true;
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
            mc->parking = false;
            continue;
        }
        Array* arr = 0;
        hashtable_get_keys(mc0->grs, &arr);
        goroutine* rungr = 0;
        for (int i = 0; arr != 0 && i < array_size(arr); i ++) {
            goroutine* gr = 0;
            void* key = 0;
            array_get_at(arr, i, &key); assert(key != 0);
            hashtable_get(mc0->grs, key, &gr); assert(gr != 0);
            if (gr->state == runnable) {
                linfo("found a runnable job %d\n", (uintptr_t)key);
                rungr = gr;
                break;
            }
        }
        if (rungr != 0) {
            noro_goroutine_run(rungr);
        } else {
            mc->parking = true;
            linfo("no task, parking... %d\n", mc->id);
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
            mc->parking = false;
        }
        // sleep(3);
    }
}

void coro_processor_yield(int fd) {
    linfo("yield %d\n", fd);
}

HashTableConf* noro_dft_htconf() { return &gnr__->htconf; }
int noro_nxtid(noro*nr) { return ++nr->gridno; }

int hashtable_cmp_int(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((uintptr_t)(key1) > (uintptr_t)(key2)) return 1;
    else return -1;
}

noro* noro_get() { return gnr__;}
noro* noro_new() {
    if (gnr__) {
        linfo("wtf...%d\n",1);
        return gnr__;
    }

    noro* nr = (noro*)GC_malloc(sizeof(noro));
    hashtable_conf_init(&nr->htconf);
    nr->htconf.key_length = sizeof(void*);
    nr->htconf.hash = hashtable_hash_ptr;
    nr->htconf.key_compare = hashtable_cmp_int;

    hashtable_new_conf(&nr->htconf, &nr->mths);
    hashtable_new_conf(&nr->htconf, &nr->mcs);

    gnr__ = nr;
    return nr;
}

// 开启的总线程数除了以下，还有libgc的线程（3个？）
void noro_init(noro* nr) {
    for (int i = 5; i > 0; i --) {
        pthread_t* t = (pthread_t*)GC_malloc(sizeof(pthread_t));
        hashtable_add(nr->mths, (void*)(uintptr_t)i, t);
        machine* mc = noro_machine_new(i);
        hashtable_add(nr->mcs, (void*)(uintptr_t)i, mc);
        if (i == 1) {
            pthread_create(t, 0, noro_processor0, (void*)mc);
        } else if (i == 2) {
            pthread_create(t, 0, noro_processor_netpoller, (void*)mc);
        } else {
            pthread_create(t, 0, noro_processor, (void*)mc);
        }
    }
}
void noro_free(noro* lnr) {
    lnr = 0;
    gnr__ = 0;
}
void noro_wait_init_done(noro* nr) {
    if (nr->noroinited) {
        return;
    }
    pthread_cond_wait(&nr->noroinitcd, &nr->noroinitmu);
}

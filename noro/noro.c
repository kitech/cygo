
#include <unistd.h>
#include <stdio.h>
#include <assert.h>

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

typedef enum grstate {runnable=0, executing, waiting, finished, } grstate;
const int dftstksz = 128*1024;

typedef struct goroutine {
    int id;
    coro_func fnproc;
    void* arg;
    coro_stack stack;
    coro_context coctx;
    coro_context coctx0;
    grstate state;
} goroutine;

typedef struct machine {
    int id;
    HashTable* ngrs; // id => goroutine* 新任务，未分配栈
    HashTable* grs;  // # grid => goroutine*
    goroutine* gr;   // 当前在执行的
    pthread_mutex_t pkmu; // pack lock
    pthread_cond_t pkcd;
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
// init stack and context
void noro_goroutine_new2(goroutine*gr) {
    corowp_create(&gr->coctx0, 0, 0, 0, 0);
    corowp_stack_alloc(&gr->stack, dftstksz);
    corowp_create(&gr->coctx, gr->fnproc, gr->arg, gr->stack.sptr, dftstksz);
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


void noro_post(coro_func fn, void*arg) {
    int id = noro_nxtid(gnr__);
    goroutine* gr = noro_goroutine_new(id, fn, arg);
    machine* mc = noro_machine_get(0);
    linfo("mc=%p\n", mc);
    hashtable_add(mc->ngrs, (void*)(uintptr_t)id, gr);
    pthread_cond_signal(&mc->pkcd);
}

void* noro_processor_netpoller(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d\n", mc->id);
}

void* noro_processor0(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d\n", mc->id);
    gnr__->noroinited = true;
    pthread_cond_signal(&gnr__->noroinitcd);

    for (;;) {
        int newg = hashtable_size(mc->ngrs);
        if (newg == 0) {
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
            newg = hashtable_size(mc->ngrs);
        }

        linfo("newgr %d\n", newg);
        Array* arr = 0;
        hashtable_get_keys(mc->ngrs, &arr);

        for (int i = 0; i < array_size(arr); i++) {
            void*key = 0;
            array_get_at(arr, i, &key);
            goroutine* gr = 0;
            hashtable_get(mc->ngrs, key, &gr);
            assert(gr != 0);
            noro_goroutine_new2(gr);
            linfo("process %d, %d\n", gr->id, dftstksz);
        }

        // cleanout
        for (int i = 0; i < array_size(arr); i++) {
            void*key = 0;
            array_get_at(arr, i, &key);
            hashtable_remove(mc->ngrs, key, 0);
        }

        // find free machine and runnable goroutine
    }
}
void* noro_processor(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d\n", mc->id);
    for (;;) {
        // check global queue
        machine* mc0 = noro_machine_get(0);
        if (mc0 == 0) {
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
            continue;
        }
        Array* arr = 0;
        hashtable_get_keys(mc0->grs, &arr);
        goroutine* rungr = 0;
        for (int i = 0; i < array_size(arr); i ++) {
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
        } else {
            linfo("no task, parking... %d\n", mc->id);
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
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

void noro_init(noro* nr) {
    for (int i = 3; i >= 0; i --) {
        pthread_t* t = (pthread_t*)GC_malloc(sizeof(pthread_t));
        hashtable_add(nr->mths, (void*)(uintptr_t)i, t);
        machine* mc = noro_machine_new(i);
        hashtable_add(nr->mcs, (void*)(uintptr_t)i, mc);
        if (i == 0) {
            pthread_create(t, 0, noro_processor0, (void*)mc);
        } else if (i == 1) {
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
    pthread_cond_wait(&nr->noroinitcd, &nr->noroinitmu);
}

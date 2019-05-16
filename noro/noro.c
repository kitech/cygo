
#include <noro.h>

#include <unistd.h>
#include <stdio.h>

#include <gc/gc.h>
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d:%s ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;

typedef struct coro_stack coro_stack;

typedef enum {waiting = 0, runnable, executing, finished, } grstate;

typedef struct {
    int id;
    coro_func fnproc;
    void* arg;
    coro_stack stack;
    grstate state;
} goroutine;

typedef struct {
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
} noro;


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
    (machine*)hashtable_get(gnr__->mcs, (void*)id, &mc);
    return mc;
}


void noro_post(coro_func fn, void*arg) {
    int id = noro_nxtid(gnr__);
    goroutine* gr = noro_goroutine_new(id, fn, arg);
    machine* mc = noro_machine_get(0);
    linfo("mc=%p\n", mc);
    hashtable_add(mc->ngrs, (void*)id, gr);
    pthread_cond_signal(&mc->pkcd);
}

void* noro_processor0(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d\n", mc->id);
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
            linfo("process %d\n", gr->id);
        }

        // cleanout
        for (int i = 0; i < array_size(arr); i++) {
            void*key = 0;
            array_get_at(arr, i, &key);
            hashtable_remove(mc->ngrs, key, 0);
        }
    }
}
void* noro_processor(void*arg) {
    machine* mc = (machine*)arg;
    linfo("%d\n", mc->id);
    for (;;) {
        sleep(3);
    }
}

HashTableConf* noro_dft_htconf() { return &gnr__->htconf; }
int noro_nxtid(noro*nr) { return ++nr->gridno; }

int hashtable_cmp_int(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((int)(key1)>(int)(key2)) return 1;
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
    for (int i = 2; i >= 0; i --) {
        pthread_t* t = (pthread_t*)GC_malloc(sizeof(pthread_t));
        hashtable_add(nr->mths, (void*)i, t);
        machine* mc = noro_machine_new(i);
        hashtable_add(nr->mcs, (void*)i, mc);
        if (i == 0) {
            pthread_create(t, 0, noro_processor0, (void*)mc);
        }else{
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


#include <unistd.h>
#include <stdio.h>
#include <assert.h>
#include <pthread.h>

#include <gc/gc.h>
// #include <private/pthread_support.h>
#include <coro.h>
#include <collectc/hashtable.h>
#include <collectc/array.h>

#include <noro.h>
#include <noropriv.h>

const int dftstksz = 128*1024;
const int dftstkusz = dftstksz/8; // unit size by sizeof(void*)

typedef struct machine {
    int id;
    HashTable* ngrs; // id => goroutine* 新任务，未分配栈
    mtx_t ngrsmu;
    HashTable* grs;  // # grid => goroutine*
    pthread_mutex_t grsmu;
    goroutine* curgr;   // 当前在执行的, 这好像得用栈结构吗？(应该不需要，goroutines之间是并列关系)
    pthread_mutex_t pkmu; // pack lock
    pthread_cond_t pkcd;
    bool parking;
    struct GC_stack_base stksb;
    void* gchandle;
    void* savefrm;
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
    netpoller* np;
} noro;


///
extern void corowp_create(coro_context *ctx, coro_func coro, void *arg, void *sptr,  size_t ssze);
extern void corowp_transfer(coro_context *prev, coro_context *next);
extern void corowp_destroy (coro_context *ctx);
extern int corowp_stack_alloc (coro_stack *stack, unsigned int size);
extern void corowp_stack_free(coro_stack* stack);
extern int gettid();

static noro* gnr__ = 0;
static void(*noro_thread_createcb)(void*arg) = 0;
HashTableConf* noro_dft_htconf();
int noro_nxtid(noro*nr);

// 前置声明一些函数
machine* noro_machine_get(int id);
void noro_machine_grfree(machine* mc, int id);
void noro_machine_signal(machine* mc);

/////
goroutine* noro_goroutine_new(int id, coro_func fn, void* arg) {
    goroutine* gr = (goroutine*)calloc(1, sizeof(goroutine));
    gr->id = id;
    gr->fnproc = fn;
    gr->arg = arg;
    gr->hcelem = invlidptr;
    return gr;
}
// alloc stack and context
void noro_goroutine_new2(goroutine*gr) {
    corowp_create(&gr->coctx0, 0, 0, 0, 0);
    // corowp_stack_alloc(&gr->stack, dftstkusz);
    gr->stack.sptr = GC_malloc_uncollectable(dftstksz);
    gr->stack.ssze = dftstksz;
    gr->mystksb.mem_base = (void*)((uintptr_t)gr->stack.sptr + dftstksz);
    atomic_setint(&gr->state, runnable);
    // GC_add_roots(gr->stack.sptr, gr->stack.sptr+(gr->stack.ssze));
    // 这一句会让fnproc直接执行，但是可能需要的是创建与执行分开。原来是针对-DCORO_PTHREAD
    // corowp_create(&gr->coctx, gr->fnproc, gr->arg, gr->stack.sptr, dftstksz);
}
void noro_goroutine_destroy(goroutine* gr) {
    assert(gr->state != executing);
    int grid = gr->id;
    int mcid = gr->mcid;
    size_t ssze = gr->stack.ssze; // save temp value

    atomic_setint(&gr->state, nostack);
    if (gr->stack.sptr != 0) {
        GC_FREE(gr->stack.sptr);
    }
    free(gr); // malloc/calloc分配的不能用GC_FREE()释放
    ssze += sizeof(goroutine);

    // linfo("gr %d on %d, freed %d, %d\n", grid, mcid, ssze, sizeof(goroutine));
}

// frame related for some frame based integeration
static void* noro_get_frame_default() { return nilptr; }
static void noro_set_frame_default(void*f) {}
static void*(*noro_get_frame)() = noro_get_frame_default;
static void(*noro_set_frame)(void*f) = noro_set_frame_default;
void noro_set_frame_funcs(void*(*getter)(), void(*setter)(void*)) {
    noro_get_frame = getter;
    noro_set_frame = setter;
}

// 恢复到线程原始的栈
void* noro_gc_setbottom0(void*arg) {
    goroutine* gr = (goroutine*)arg;
    GC_set_stackbottom(gr->gchandle, gr->stksb);
    // GC_stackbottom = sb2.bottom;
    return 0;
}
// coroutine动态栈
void* noro_gc_setbottom1(void*arg) {
    goroutine* gr = (goroutine*)arg;

    GC_set_stackbottom(gr->gchandle, &gr->mystksb);
    // GC_stackbottom = sb1.bottom;
    return 0;
}

void noro_goroutine_forward(void* arg) {
    goroutine* gr = (goroutine*)arg;
    GC_call_with_alloc_lock(noro_gc_setbottom1, gr);

    gr->fnproc(gr->arg);
    atomic_setint(&gr->state, finished);
    // linfo("coro end??? %d\n", 1);
    // TODO coro 结束，回收coro栈

    // 好像应该在外层处理
    GC_call_with_alloc_lock(noro_gc_setbottom0, gr);

    // 这个跳回ctx应该是不对的，有可能要跳到其他的gr而不是默认gr？
    corowp_transfer(&gr->coctx, &gr->coctx0); // 这句要写在函数fnproc退出之前？
}

// 由于需要考虑线程的问题，不能直接在netpoller线程调用
void noro_goroutine_resume(goroutine* gr) {
    assert(gr->isresume == true);
    assert(gr->state != executing);
    atomic_setint(&gr->state, executing);

    if (gr->myfrm != nilptr) noro_set_frame(gr->myfrm); // 恢复goroutine的frame
    GC_call_with_alloc_lock(noro_gc_setbottom1, gr);
    // 对-DCORO_UCONTEXT/-DCORO_ASM等来说，这句是真正开始执行
    corowp_transfer(&gr->coctx0, &gr->coctx);
}

// TODO 有时候它不一定是从ctx0跳转，或者是跳转到ctx0。这几个函数都是 noro_goroutine_run/resume,suspend
// 一定是从ctx0跳过来的，因为所有的goroutines是由调度器发起 run/resume/suspend，而不是其中某一个goroutine发起
void noro_goroutine_run_first(goroutine* gr) {
    // first run
    assert(atomic_getbool(&gr->isresume) == false);
    atomic_setbool(&gr->isresume, true);
    // 对-DCORO_PTHREAD来说，这句是真正开始执行
    corowp_create(&gr->coctx, noro_goroutine_forward, gr, gr->stack.sptr, gr->stack.ssze);

    atomic_setint(&gr->state, executing);
    machine* mc = noro_machine_get(gr->mcid);
    goroutine* curgr = mc->curgr;
    mc->curgr = gr;
    coro_context* curcoctx = curgr == 0? &gr->coctx0 : &curgr->coctx; // 暂时无用

    // 对-DCORO_UCONTEXT/-DCORO_ASM等来说，这句是真正开始执行
    corowp_transfer(&gr->coctx0, &gr->coctx);
    // corowp_transfer(&gr->coctx, &gr->coctx0); // 这句要写在函数fnproc退出之前？
}

void noro_goroutine_run(goroutine* gr) {
    // linfo("rungr %d %d\n", gr->id, gr->isresume);
    if (atomic_getbool(&gr->isresume) == true) {
        noro_goroutine_resume(gr);
    } else {
        noro_goroutine_run_first(gr);
    }
}

// 只改状态，不切换
void noro_goroutine_resume_same_thread(goroutine* gr) {
    assert(gr->isresume == true);
    assert(gr->state != executing);
    atomic_setint(&gr->state, runnable);
}
void noro_goroutine_resume_xthread(goroutine* gr) {
    // assert(gr->state != runnable);
    // assert(gr->state != executing);
    if (gr->id <= 0) {
        linfo("some error occurs??? %d\n", gr->id);
        return;
        // maybe goroutine already finished and deleted
        // TODO assert(gr != nilptr && gr->id > 0); // needed ???
    }
    if (atomic_getint(&gr->state) == executing) {
        linfo("resume but executing grid=%d, mcid=%d\n", gr->id, gr->mcid);
        return;
    }
    if (atomic_getint(&gr->state) == runnable) {
        linfo("resume but runnable %d\n", gr->id);
        noro_machine_signal(noro_machine_get(gr->mcid));
        return;
    }
    atomic_setint(&gr->state, runnable);
    noro_machine_signal(noro_machine_get(gr->mcid));
}
void noro_goroutine_suspend(goroutine* gr) {
    gr->myfrm = noro_get_frame();
    noro_set_frame(gr->savefrm);
    atomic_setint(&gr->state, waiting);
    GC_call_with_alloc_lock(noro_gc_setbottom0, gr);
    corowp_transfer(&gr->coctx, &gr->coctx0);
}


machine* noro_machine_new(int id) {
    machine* mc = (machine*)calloc(1, sizeof(machine));
    mc->id = id;
    linfo("htconf=%o\n", noro_dft_htconf());
    hashtable_new_conf(noro_dft_htconf(), &mc->ngrs);
    hashtable_new_conf(noro_dft_htconf(), &mc->grs);
    return mc;
}
machine* noro_machine_get(int id) {
    assert(id > 0);
    machine* mc = 0;
    hashtable_get(gnr__->mcs, (void*)(uintptr_t)id, (void**)&mc);
    // linfo("get mc %d=%p\n", id, mc);
    if (mc != 0) {
        // FIXME
        if (mc->id != id) {
            linfo("get mc %d=%p, found=%d, size=%d\n", id, mc, mc->id, hashtable_size(gnr__->mcs));

            machine* mc2 = 0;
            hashtable_get(gnr__->mcs, (void*)(uintptr_t)id, (void**)&mc2);
            linfo("get mc %d=%p found=%d\n", id, mc2, mc2->id);
        }
        assert(mc->id == id);
    }
    return mc;
}

void noro_machine_gradd(machine* mc, goroutine* gr) {
    pthread_mutex_lock(&mc->grsmu);
    hashtable_add(mc->grs, (void*)(uintptr_t)gr->id, gr);
    pthread_mutex_unlock(&mc->grsmu);
}
goroutine* noro_machine_grget(machine* mc, int id) {
    goroutine* gr = 0;
    pthread_mutex_lock(&mc->grsmu);
    hashtable_get(mc->grs, (void*)(uintptr_t)id, (void**)&gr);
    pthread_mutex_unlock(&mc->grsmu);
    return gr;
}
goroutine* noro_machine_grdel(machine* mc, int id) {
    goroutine* gr = 0;
    pthread_mutex_lock(&mc->grsmu);
    hashtable_remove(mc->grs, (void*)(uintptr_t)id, (void**)&gr);
    pthread_mutex_unlock(&mc->grsmu);
    // assert(gr != 0);
    return gr;
}
void noro_machine_grfree(machine* mc, int id) {
    goroutine* gr = noro_machine_grdel(mc, id);
    assert(gr != nilptr);
    assert(gr->id == id);
    noro_goroutine_destroy(gr);
}
// for mc->id == 1
goroutine* noro_machine_grtake(machine* mc) {
    assert(mc->id == 1);
    goroutine* gr = nilptr;
    Array* arr = nilptr;
    pthread_mutex_lock(&mc->grsmu);
    hashtable_get_keys(mc->grs, (void**)&arr);
    if (arr != nilptr) {
        void* key = nilptr;
        array_get_at(arr, 0, (void**)&key);
        hashtable_remove(mc->grs, key, (void**)&gr);
        array_destroy(arr);
    }
    pthread_mutex_unlock(&mc->grsmu);
    return gr;
}
void noro_machine_signal(machine* mc) {
    pthread_cond_signal(&mc->pkcd);
}

static __thread int gcurmcid__ = 0; // thread local
static __thread int gcurgrid__ = 0; // thread local
int noro_get_goid() { return gcurgrid__; }
goroutine* noro_goroutine_getcur() {
    int grid = gcurgrid__;
    int mcid = gcurmcid__;
    if (mcid == 0) {
        // linfo("Not goroutine, main/poller thread %d?\n", mcid);
        return 0;
    }
    machine* mcx = noro_machine_get(mcid);
    goroutine* gr = 0;
    gr = noro_machine_grget(mcx, grid);
    if (gr == nilptr) {
        linfo("wtf why gr nil, curmc %d, curgr %d\n", mcid, grid);
    }
    assert(gr != nilptr);
    return gr;
}
// int noro_num_gorutines() { return atomic_getint(gnr__)}

void noro_post(coro_func fn, void*arg) {
    int id = noro_nxtid(gnr__);
    goroutine* gr = noro_goroutine_new(id, fn, arg);
    machine* mc = noro_machine_get(1);
    // linfo("mc=%p, %d %p, %d\n", mc, mc->id, mc->ngrs, hashtable_size(mc->ngrs));
    if (mc != 0 && mc->id != 1) {
        // FIXME
        linfo("nothing mc=%p, %d %p, %d\n", mc, mc->id, mc->ngrs, hashtable_size(mc->ngrs));
        return;
    }
    mtx_lock(&mc->ngrsmu);
    hashtable_add(mc->ngrs, (void*)(uintptr_t)id, gr);
    mtx_unlock(&mc->ngrsmu);
    pthread_cond_signal(&mc->pkcd);
}

void noro_processor_setname(int id) {
    char buf[32] = {0};
    snprintf(buf, sizeof(buf), "noro_processor_%d", id);
    pthread_setname_np(pthread_self(), buf);
}
void* noro_processor_netpoller(void*arg) {
    machine* mc = (machine*)arg;
    struct GC_stack_base stksb = {};
    GC_get_stack_base(&stksb);
    GC_register_my_thread(&stksb);
    linfo("%d, %d\n", mc->id, gettid());
    noro_processor_setname(mc->id);

    netpoller_loop();

    assert(1==2);
    // cannot reachable
    for (;;) {
        sleep(600);
    }
    return nilptr;
}

void* noro_processor0(void*arg) {
    machine* mc = (machine*)arg;
    // linfo("%d %d\n", mc->id, gettid());
    noro_processor_setname(mc->id);
    gnr__->noroinited = true;
    pthread_cond_signal(&gnr__->noroinitcd);

    for (;;) {
        mtx_lock(&mc->ngrsmu);
        int newg = hashtable_size(mc->ngrs);
        mtx_unlock(&mc->ngrsmu);
        if (newg == 0) {
            mc->parking = true;
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
            mc->parking = false;
            mtx_lock(&mc->ngrsmu);
            newg = hashtable_size(mc->ngrs);
            mtx_unlock(&mc->ngrsmu);
        }

        // linfo("newgr %d\n", newg);
        Array* arr1 = 0;
        mtx_lock(&mc->ngrsmu);
        hashtable_get_keys(mc->ngrs, &arr1);
        mtx_unlock(&mc->ngrsmu);
        if (arr1 == nilptr || array_size(arr1) == 0) {
            continue;
        }

        for (int i = 0; arr1 != 0 && i < array_size(arr1); i++) {
            void* key = array_peek_at(arr1, i);
            goroutine* gr = 0;
            mtx_lock(&mc->ngrsmu);
            hashtable_remove(mc->ngrs, key, (void**)&gr); assert(gr != 0);
            mtx_unlock(&mc->ngrsmu);
            noro_goroutine_new2(gr);
            // linfo("process %d, %d\n", gr->id, dftstksz);
            noro_machine_gradd(mc, gr);
        }

        // TODO 应该放到schedule中
        // find free machine and runnable goroutine
        Array* arr2 = 0;
        hashtable_get_keys(gnr__->mcs, &arr2);
        for (;;) {
            goroutine* gr = noro_machine_grtake(mc);
            if (gr == nilptr) {
                linfo("why nil %d\n", hashtable_size(mc->grs));
                break;
            }

            machine* mct = 0;
            if (arr2 != nilptr) array_sort(arr2, array_randcmp);
            for (int j = 0; arr2!=0 && j < array_size(arr2); j++) {
                void* key = array_peek_at(arr2, j);
                if ((uintptr_t)key <= 2) continue;

                // linfo("checking machine %d/%d %d\n", j, array_size(arr2), key);
                mct = noro_machine_get((int)(uintptr_t)key);
                if (mct->parking) {
                    // linfo("got a packing machine %d <- %d\n", mct->id, gr->id);
                    break;
                }
                mct = 0;
            }
            if (mct == 0) {
                linfo("no enough mc? %d\n", gr->id);
                // try select random one?
                // 暂时先放回全局队列中吧
                noro_machine_gradd(mc, gr);
                break;
            }
            if (mct != 0) {
                // linfo("move %d to %d\n", gr->id, mct->id);
                noro_machine_gradd(mct, gr);
                noro_machine_signal(mct);
                break;
            }
        }
        if (arr1 != nilptr) array_destroy(arr1);
        if (arr2 != nilptr) array_destroy(arr2);
    }
}

// schedue functions
goroutine* noro_sched_get_glob_one(machine*mc) {
    // linfo("try get glob %d\n", mc->id);
    machine* mc1 = noro_machine_get(1);
    if (mc1 == 0) return 0;

    goroutine* gr = noro_machine_grtake(mc1);
    if (gr != 0) {
        linfo("got %d glob task on %d\n", gr->id, mc->id);
    }
    return gr;
}
// prepare new task
static
goroutine* noro_sched_get_ready_one(machine*mc) {
    Array* arr = 0;
    pthread_mutex_lock(&mc->grsmu);
    hashtable_get_keys(mc->grs, &arr);
    goroutine* rungr = 0;
    for (int i = 0; arr != 0 && i < array_size(arr); i ++) {
        goroutine* gr = 0;
        void* key = 0;
        array_get_at(arr, i, &key);  assert(key != 0);
        hashtable_get(mc->grs, key, (void**)&gr); assert(gr != 0);
        if (atomic_getint(&gr->state) == runnable) {
            // linfo("found a runnable job %d on %d\n", (uintptr_t)key, mc->id);
            rungr = gr;
            break;
        }
        // linfo("grid=%d mcid=%d, grstate=%d\n", gr->id, gr->mcid, atomic_getint(&gr->state));
    }
    if (arr != 0) {
        array_destroy(arr);
    }
    pthread_mutex_unlock(&mc->grsmu);
    return rungr;
}
static
void noro_sched_run_one(machine* mc, goroutine* rungr) {
    gcurgrid__ = rungr->id;
    rungr->stksb = &mc->stksb;
    rungr->gchandle = mc->gchandle;
    rungr->mcid = mc->id;
    rungr->savefrm = mc->savefrm;
    noro_goroutine_run(rungr);
    gcurgrid__ = 0;

    int curst = atomic_getint(&rungr->state);
    if (curst == waiting) {
        // 在这才解锁，用于确保rungr状态完全切换完成
        if (rungr->hclock != nilptr) {
            mtx_t* l = rungr->hclock;
            rungr->hclock = nilptr;
            mtx_unlock(l);
            // linfo("unlocked chan lock %p on %d\n", l, rungr->id);
        }
    } else if (curst == finished) {
        // linfo("finished gr %d\n", rungr->id);
        noro_machine_grfree(mc, rungr->id);
    }else{
        linfo("break from gr %d, state=%d pkreason=%d\n", rungr->id, curst, rungr->pkreason);
    }
}

void* noro_processor(void*arg) {
    machine* mc = (machine*)arg;
    GC_get_stack_base(&mc->stksb);
    GC_register_my_thread(&mc->stksb);
    mc->gchandle = GC_get_my_stackbottom(&mc->stksb);
    if (noro_thread_createcb != 0) {
        noro_thread_createcb((void*)(uintptr_t)mc->id);
    }

    // linfo("%d %d\n", mc->id, gettid());
    noro_processor_setname(mc->id);
    gcurmcid__ = mc->id;

    mc->savefrm = noro_get_frame();
    for (;;) {
        // check global queue
        goroutine* rungr = noro_sched_get_ready_one(mc);
        if (rungr != 0) {
            noro_sched_run_one(mc, rungr);
            continue;
        }
        if (rand() % 3 == 2) {
            rungr = noro_sched_get_glob_one(mc);
        }
        if (rungr != 0) {
            noro_machine_gradd(mc, rungr);
        } else {
            mc->parking = true;
            // linfo("no task, parking... %d\n", mc->id);
            pthread_cond_wait(&mc->pkcd, &mc->pkmu);
            mc->parking = false;
        }
        // sleep(3);
    }
}

bool noro_in_processor() { return gcurmcid__ != 0; }
int noro_processor_yield(long fd, int ytype) {
    // check是否是processor线程
    if (gcurmcid__ == 0) {
        linfo("maybe not processor thread %ld %d\n", fd, ytype)
            // 应该不是 processor线程
            return -1;
    }
    // linfo("yield fd=%ld, ytype=%s(%d), mcid=%d, grid=%d\n", fd, yield_type_name(ytype), ytype, gcurmcid__, gcurgrid__);
    goroutine* gr = noro_goroutine_getcur();
    gr->pkreason = ytype;
    if (ytype == YIELD_TYPE_CHAN_RECV || ytype == YIELD_TYPE_CHAN_SEND ||
        ytype == YIELD_TYPE_CHAN_SELECT || ytype == YIELD_TYPE_CHAN_SELECT_NOCASE) {
    } else {
        netpoller_yieldfd(fd, ytype, gr);
    }
    noro_goroutine_suspend(gr);
    return 0;
}
int noro_processor_yield_multi(int ytype, int nfds, long fds[], int ytypes[]) {
    // check是否是processor线程
    if (gcurmcid__ == 0) {
        linfo("maybe not processor thread %d %d\n", nfds, ytype)
            // 应该不是 processor线程
            return -1;
    }
    // linfo("yield %d ytype=%s(%d), mcid=%d, grid=%d\n", nfds, yield_type_name(ytype), ytype, gcurmcid__, gcurgrid__);
    goroutine* gr = noro_goroutine_getcur();
    gr->pkreason = ytype;
    for (int i = 0; i < nfds; i ++) {
        long fd = fds[i];
        int ytype = ytypes[i];
        if (ytype == YIELD_TYPE_CHAN_RECV || ytype == YIELD_TYPE_CHAN_SEND ||
            ytype == YIELD_TYPE_CHAN_SELECT || ytype == YIELD_TYPE_CHAN_SELECT_NOCASE) {
        } else {
            netpoller_yieldfd(fd, ytype, gr);
        }
    }
    noro_goroutine_suspend(gr);
    return 0;
}
void noro_processor_resume_some(void* gr_, int ytype) {
    goroutine* gr = (goroutine*)gr_;
    goroutine* mygr = noro_goroutine_getcur();
    ytype = (ytype == 0 ? gr->pkreason : ytype);
    // linfo("netpoller notify, ytype=%d %p, id=%d\n", ytype, gr, gr->id);
    if (mygr != nilptr && gr->mcid == mygr->mcid) {
        noro_goroutine_resume_same_thread(gr);
        // 相同machine线程的情况，要主动出让执行权。
        // 另外考虑是否只针对chan send/recv。
        noro_processor_yield(1001, YIELD_TYPE_NANOSLEEP);
    }else {
        noro_goroutine_resume_xthread(gr);
    }
}
void noro_sched() {
    noro_processor_yield(1000, YIELD_TYPE_NANOSLEEP);
}

HashTableConf* noro_dft_htconf() { return &gnr__->htconf; }
int noro_nxtid(noro*nr) {
    int id = atomic_addint(&nr->gridno, 1);
    return id;
}

int hashtable_cmp_int(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((uintptr_t)(key1) > (uintptr_t)(key2)) return 1;
    else return -1;
}

static void noro_gc_push_other_roots() {
    linfo("push other roots %d\n", gettid());
}
static void noro_gc_on_collection_event(GC_EventType evty) {
    linfo("%d=%s\n", evty, noro_gc_event_name(evty));
}
static void noro_gc_on_thread_event(GC_EventType evty, void* thid) {
    linfo("%d=%s %p\n", evty, noro_gc_event_name(evty), thid);
}
static void noro_init_intern() {
    srand(time(0));
    // GC_enable_incremental();
    // GC_set_rate(5);
    // GC_set_all_interior_pointers(1);
    // TODO
    // GC_set_push_other_roots(noro_gc_push_other_roots);
    // GC_set_on_collection_event(noro_gc_on_collection_event);
    // GC_set_on_thread_event(noro_gc_on_thread_event);
    GC_INIT();
    GC_allow_register_threads();
    // linfo("main thread registered: %d\n", GC_thread_is_registered()); // yes
    // linfo("gcfreq=%d\n", GC_get_full_freq()); // 19
    // GC_set_full_freq(5);

    // log init here
    netpoller_use_threads();
}

noro* noro_get() { return gnr__;}
noro* noro_new() {
    if (gnr__) {
        linfo("wtf...%d\n",1);
        return gnr__;
    }
    noro_init_intern();

    noro* nr = (noro*)calloc(1, sizeof(noro));
    hashtable_conf_init(&nr->htconf);
    nr->htconf.key_length = sizeof(void*);
    nr->htconf.hash = hashtable_hash_ptr;
    nr->htconf.key_compare = hashtable_cmp_int;

    hashtable_new_conf(&nr->htconf, &nr->mths);
    hashtable_new_conf(&nr->htconf, &nr->mcs);

    nr->gridno = 1;
    nr->np = netpoller_new();

    gnr__ = nr;
    return nr;
}


// 开启的总线程数除了以下，还有libgc的线程（3个？）
void noro_init(noro* nr) {
    GC_disable();
    for (int i = 5; i > 0; i --) {
        pthread_t* t = (pthread_t*)calloc(1, sizeof(pthread_t));
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
    GC_enable();
}
void noro_destroy(noro* lnr) {
    lnr = 0;
    gnr__ = 0;
}
void noro_wait_init_done(noro* nr) {
    linfo("noroinited? %d\n", nr->noroinited);
    if (nr->noroinited) {
        return;
    }
    pthread_cond_wait(&nr->noroinitcd, &nr->noroinitmu);
}

noro* noro_init_and_wait_done() {
    noro* nr = noro_new();
    noro_init(nr);
    linfo("wait signal...%d\n", 0);
    noro_wait_init_done(nr);
    linfo("wait signal done %d\n", 0);
    return nr;
}

void* noro_set_thread_createcb(void(*fn)(void*arg), void* arg) {
    void(*oldfn)(void*arg) = noro_thread_createcb;
    noro_thread_createcb = fn;
    return oldfn;
}

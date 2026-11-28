
// #include <collectc/hashtable.h>
// #include <collectc/array.h>
// #include <collectc/queue.h>
#include <collect_cds.h>
#include <pthread.h>

#include "coronapriv.h"
#include "futex.h"

// thread safe data struct

typedef struct crnmap crnmap;
struct crnmap {
    HashTable* ht;
    pmutex_t mu;
};
typedef struct crnarray crnarray;
struct crnarray {
    Array* arr;
    pmutex_t mu;
};
typedef struct crnqueue crnqueue;
struct crnqueue {
    Queue* qo;
    pmutex_t mu;
};
typedef struct crnunique crnunique;
struct crnunique {
    HashTable* ht;
    Queue* qo;
    pmutex_t mu;
};

static
int __attribute__((no_instrument_function))
crnhashtable_cmp_uintptr(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((uintptr_t)(key1) > (uintptr_t)(key2)) return 1;
    else return -1;
}

static
HashTable* crnhashtable_new_conf(HashTableConf* htconf) {
    htconf->key_length = sizeof(void*);
    htconf->mem_alloc = crn_gc_malloc;
    htconf->mem_calloc = crn_gc_calloc;
    htconf->mem_free = crn_gc_free;

    HashTable* out = 0;
    int rv = hashtable_new_conf(htconf, &out);
    assert(rv == CC_OK);
    return out;
}

HashTable* crnhashtable_new_uintptr() {
    HashTableConf htconf = {0};
    hashtable_conf_init(&htconf);
    htconf.hash = hashtable_hash_ptr;
    htconf.key_compare = crnhashtable_cmp_uintptr;

    return crnhashtable_new_conf(&htconf);
}

crnmap* crnmap_new_uintptr() {
    crnmap* mp = (crnmap*)crn_gc_malloc(sizeof(crnmap));
    pthread_mutexattr_t attr;
    pthread_mutexattr_init(&attr);
    if (0) {
	pthread_mutexattr_settype(&attr, PTHREAD_MUTEX_RECURSIVE);
    }
    pmutex_init(&mp->mu, &attr);
    mp->ht = crnhashtable_new_uintptr();
    return mp;
}

void crnmap_free (crnmap *table) {
    pmutex_lock(&table->mu);
    hashtable_destroy(table->ht);
    table->ht = 0;
    pmutex_unlock(&table->mu);
}
enum cc_stat crnmap_add (crnmap *table, uintptr_t key, void *val) {
    pmutex_lock(&table->mu);
    enum cc_stat rv = hashtable_add(table->ht, (void*)key, val);
    pmutex_unlock(&table->mu);
    return rv;
}
enum cc_stat crnmap_get(crnmap *table, uintptr_t key, void **out) {
    pmutex_lock(&table->mu);
    enum cc_stat rv = hashtable_get(table->ht, (void*)key, out);
    pmutex_unlock(&table->mu);
    return rv;
}
enum cc_stat crnmap_get_nolk(crnmap *table, uintptr_t key, void **out) {
    enum cc_stat rv = hashtable_get(table->ht, (void*)key, out);
    return rv;
}
enum cc_stat crnmap_remove(crnmap *table, uintptr_t key, void **out) {
    pmutex_lock(&table->mu);
    enum cc_stat rv = hashtable_remove(table->ht, (void*)key, out);
    pmutex_unlock(&table->mu);
    return rv;
}
void crnmap_remove_all(crnmap *table) {
    pmutex_lock(&table->mu);
    hashtable_remove_all(table->ht);
    pmutex_unlock(&table->mu);
}
bool crnmap_contains_key(crnmap *table, uintptr_t key) {
    pmutex_lock(&table->mu);
    bool rv = hashtable_contains_key(table->ht, (void*)key);
    pmutex_unlock(&table->mu);
    return rv;
}

// atomic contains+add: closes the race between two concurrent adders that
// plain contains_key-then-add has; used for fiber id allocation
bool crnmap_add_ifabsent(crnmap *table, uintptr_t key, void *val) {
    pmutex_lock(&table->mu);
    bool exists = hashtable_contains_key(table->ht, (void*)key);
    if (!exists) {
        hashtable_add(table->ht, (void*)key, val);
    }
    pmutex_unlock(&table->mu);
    return !exists;
}

size_t crnmap_size(crnmap *table){
    pmutex_lock(&table->mu);
    size_t rv = hashtable_size(table->ht);
    pmutex_unlock(&table->mu);
    return rv;
}
size_t crnmap_capacity(crnmap *table){
    pmutex_lock(&table->mu);
    size_t rv = hashtable_capacity(table->ht);
    pmutex_unlock(&table->mu);
    return rv;
}

enum cc_stat crnmap_get_keys(crnmap *table, Array **out){
    pmutex_lock(&table->mu);
    enum cc_stat rv = hashtable_get_keys(table->ht, out);
    pmutex_unlock(&table->mu);
    return rv;
}
enum cc_stat crnmap_get_values(crnmap *table, Array **out){
    pmutex_lock(&table->mu);
    enum cc_stat rv = hashtable_get_values(table->ht, out);
    pmutex_unlock(&table->mu);
    return rv;
}

// Allocation-free iteration over a crnmap.  Non-blocking: if the table lock
// is already held (e.g. by a suspended thread), return false and skip the pass
// instead of risking a deadlock in the yield path.
bool crnmap_foreach(crnmap *table, crnmap_iter_fn fn, void *arg) {
    if (pmutex_trylock(&table->mu) != 0) {
        return false;
    }
    HashTableIter iter;
    TableEntry *te = NULL;
    hashtable_iter_init(&iter, table->ht);
    while (hashtable_iter_next(&iter, &te) != CC_ITER_END) {
        if (te == NULL) break;
        if (fn((uintptr_t)te->key, te->value, arg) != 0) break;
    }
    pmutex_unlock(&table->mu);
    return true;
}

// 随机摘除一个元素
void* crnmap_takeone(crnmap* table) {
    void* val = nilptr;
    Array* arr = nilptr;

    pmutex_lock(&table->mu);
    int rv = hashtable_get_keys(table->ht, &arr);
    assert(rv == CC_OK || rv == 2);
    if (arr != nilptr) {
        void* key = nilptr;
        rv = array_get_at(arr, 0, (void**)&key);
        assert(rv == CC_OK);
        rv = hashtable_remove(table->ht, key, (void**)&val);
        assert(rv == CC_OK);
    }
    pmutex_unlock(&table->mu);
    if (arr != nilptr) array_destroy(arr);

    return val;
}
// dont call oother crnmap_ in chkfn, or deadlock
void* crnmap_findone(crnmap* table, bool(*chkfn)(void* v)) {
    void* val = nilptr;
    Array* arr = nilptr;

    pmutex_lock(&table->mu);
    int rv = hashtable_get_keys(table->ht, &arr);
    assert(rv == CC_OK || rv == CC_ERR_INVALID_CAPACITY);
    for (int i = 0; arr != 0 && i < array_size(arr); i ++) {
        void* tval = 0;
        void* key = 0;
        rv = array_get_at(arr, i, &key);  assert(key != 0);
        assert(rv == CC_OK);
        rv = hashtable_get(table->ht, (void*)(uintptr_t)key, (void**)&tval);
        assert(tval != 0);
        assert(rv == CC_OK);
        if (chkfn(tval)) {
            val = tval;
            break;
        }
    }
    pmutex_unlock(&table->mu);
    if (arr != nilptr) array_destroy(arr);

    return val;
}
HashTable* crnmap_getraw(crnmap* table)  { return table->ht; }

/////
static Queue* crnqueue_new_conf(QueueConf *qconf) {
    qconf->mem_alloc = crn_gc_malloc;
    qconf->mem_free = crn_gc_free;
    qconf->mem_calloc = crn_gc_calloc;

    Queue* q = nilptr;
    int rv = queue_new_conf(qconf, &q);
    assert(rv == CC_OK);
    return q;
}
crnqueue* crnqueue_new() {
    crnqueue* q = crn_gc_malloc(sizeof(crnqueue));
    pmutex_init(&q->mu, nilptr);

    QueueConf qconf = {0};
    queue_conf_init(&qconf);
    q->qo = crnqueue_new_conf(&qconf);
    return q;
}

void crnqueue_free(crnqueue* q) {
    pmutex_lock(&q->mu);
    queue_destroy(q->qo);
    q->qo = 0;
    pmutex_unlock(&q->mu);
}

enum cc_stat crnqueue_peek(crnqueue* q, void **out) {
    pmutex_lock(&q->mu);
    enum cc_stat rv = queue_peek(q->qo, out);
    pmutex_unlock(&q->mu);
    return rv;
}

enum cc_stat crnqueue_poll(crnqueue *q, void **out){
    pmutex_lock(&q->mu);
    enum cc_stat rv = queue_poll(q->qo, out);
    pmutex_unlock(&q->mu);
    return rv;
}

enum cc_stat crnqueue_enqueue(crnqueue *q, void *element){
    pmutex_lock(&q->mu);
    enum cc_stat rv = queue_enqueue(q->qo, element);
    pmutex_unlock(&q->mu);
    return rv;
}

size_t crnqueue_size(crnqueue* q){
    pmutex_lock(&q->mu);
    enum cc_stat rv = queue_size(q->qo);
    pmutex_unlock(&q->mu);
    return rv;
}

/////
crnunique* crnunique_new() {
    crnunique* q = (crnunique*)crn_gc_malloc(sizeof(crnunique));
    pmutex_init(&q->mu, nilptr);
    q->ht = crnhashtable_new_uintptr();
    QueueConf qconf = {0};
    queue_conf_init(&qconf);
    q->qo = crnqueue_new_conf(&qconf);
    return q;
}

void crnunique_free(crnunique* q) {
    pmutex_lock(&q->mu);
    queue_destroy(q->qo);
    q->qo = 0;
    hashtable_destroy(q->ht);
    q->ht = 0;
    pmutex_unlock(&q->mu);
}

enum cc_stat crnunique_peek(crnunique* q, void **out) {
    pmutex_lock(&q->mu);
    enum cc_stat rv = queue_peek(q->qo, out);
    pmutex_unlock(&q->mu);
    return rv;
}

enum cc_stat crnunique_poll(crnunique *q, void **out){
    pmutex_lock(&q->mu);
    enum cc_stat rv = queue_poll(q->qo, out);
    if (rv == CC_OK) {
        enum cc_stat rv1 = hashtable_remove(q->ht, *out, 0);
        assert(rv1 == CC_OK);
    }
    pmutex_unlock(&q->mu);
    return rv;
}

enum cc_stat crnunique_enqueue(crnunique *q, void *element){
    pmutex_lock(&q->mu);
    bool has = hashtable_contains_key(q->ht, element);
    enum cc_stat rv = CC_OK;
    if (!has) {
        rv = queue_enqueue(q->qo, element);
        enum cc_stat rv2 = hashtable_add(q->ht, element, (void*)0x1);
        assert(rv2 == CC_OK);
    }
    pmutex_unlock(&q->mu);
    return rv;
}

size_t crnunique_size(crnunique* q){
    pmutex_lock(&q->mu);
    enum cc_stat rv = queue_size(q->qo);
    int rv2 = hashtable_size(q->ht);
    pmutex_unlock(&q->mu);
    assert(rv == rv2);
    return rv;
}

/////
// netpoller pending-registration registry（event/ev/uv 三后端共享，无锁）。
//
// 结构：每个 fiber 一条单链表（头在 gr->nphead），节点是各后端 evdata
// 的首成员 npreg_hdr。
//
// 线程纪律（这是无锁成立的前提）：
//  - 链表只由该 fiber 自己的 machine 线程读写：add（注册时入列）与
//    drain（销毁/poll轮清理时遍历）。loop 线程绝不遍历、绝不改链表，
//    跨线程只碰 npstate 一个原子字。
//  - evdata 只由 machine 线程（drain）释放；回调（loop 线程）认领后
//    只标记状态、拷贝字段、event_free 自己的 event，从不 free evdata。
//    销毁时仍在途的已认领条目直接弃置——它是 GC 内存，链表随 fiber
//    消失后无人引用，GC 自然回收。
//  - event（libevent 自有内存）由认领方（回调）或 drain 中的一方释放，
//    归属由 npstate 的原子交换裁决，恰好一次。
//
// npstate: 0=已注册 NPREG_CLAIMED=回调已认领 NPREG_DONE=回调已收尾
//          NPREG_DYING 位=fiber 正在销毁（drain 对已认领条目置位，
//          在途回调看到后跳过唤醒）
void crn_npreg_add(fiber* gr, void* d) {
    npreg_hdr* h = (npreg_hdr*)d;
    h->npstate = NPREG_REGD;
    h->npnext = gr->nphead;
    gr->nphead = d;
}
// machine 线程：撤销刚压入的链头登记（武装失败路径）。
// 单写者纪律保证 d 就是当前链头。
void crn_npreg_undo(fiber* gr, void* d) {
    npreg_hdr* h = (npreg_hdr*)d;
    if (gr->nphead == d) {
        gr->nphead = h->npnext;
    }
}
// loop 线程回调入口调用。返回：
//   0 = 认领失败（并发 drain 已接管该条目），调用方不得再触碰 d
//   1 = 认领成功，正常走唤醒
//   2 = 认领成功但 fiber 正在销毁：只 event_free 自己的 event 并收尾，
//       跳过唤醒
int crn_npreg_claim(void* d) {
    npreg_hdr* h = (npreg_hdr*)d;
    int old = __atomic_exchange_n(&h->npstate, NPREG_CLAIMED, __ATOMIC_ACQ_REL);
    if (old & (NPREG_CLAIMED | NPREG_DONE | NPREG_DRAINING)) { return 0; }
    if (__atomic_load_n(&h->npstate, __ATOMIC_ACQUIRE) & NPREG_DYING) { return 2; }
    return 1;
}
// loop 线程回调收尾调用：标记 DONE，条目留给下一次 drain 释放
// （本 fiber 已销毁时无人再 drain，弃置给 GC）
void crn_npreg_done(void* d) {
    npreg_hdr* h = (npreg_hdr*)d;
    __atomic_store_n(&h->npstate, NPREG_DONE, __ATOMIC_RELEASE);
}
// machine 线程调用（destroy 兜底 / poll 每轮清理）。返回释放条数。
// 遍历链表安全：节点只由本线程释放，在途回调只碰状态字不改链。
int crn_npreg_drain(fiber* gr, void (*free_fn)(void* d)) {
    int n = 0;
    void* p = gr->nphead;
    gr->nphead = nilptr;
    while (p != nilptr) {
        npreg_hdr* h = (npreg_hdr*)p;
        void* nxt = h->npnext;   // 先取 next：free_fn 会释放节点
        int st = __atomic_load_n(&h->npstate, __ATOMIC_ACQUIRE);
        if (st == NPREG_REGD) {
            // 0 -> DRAINING 抢归属：抢到才释放；被并发回调抢走则置 DYING
            // 让其跳过唤醒（回调从不释放 evdata，条目弃置给 GC）
            int exp = NPREG_REGD;
            if (__atomic_compare_exchange_n(&h->npstate, &exp, NPREG_DRAINING,
                                            0, __ATOMIC_ACQ_REL, __ATOMIC_ACQUIRE)) {
                if (free_fn != nilptr) { free_fn(p); }
                n++;
            } else {
                __atomic_fetch_or(&h->npstate, NPREG_DYING, __ATOMIC_ACQ_REL);
            }
        } else if (st == NPREG_DONE) {
            // 回调已收尾（event 已由其释放）：只回收 evdata
            if (free_fn != nilptr) { free_fn(p); }
            n++;
        } else if (st & NPREG_CLAIMED) {
            // 在途回调：置 DYING 让它跳过唤醒；条目不再动（弃置给 GC
            // 或下一次 drain）
            __atomic_fetch_or(&h->npstate, NPREG_DYING, __ATOMIC_ACQ_REL);
        }
        p = nxt;
    }
    return n;
}
// drain 的"只收集不释放"变体：给注册/释放都必须在 loop 线程做的后端
// （ev/uv）用。machine 线程把 REGD 条目摘出（CAS 0->DRAINING 抢归属）
// 放进 arr，调用方再经后端线程桥转给 loop 释放；在途回调置 DYING 跳过
// 唤醒后自行收尾。返回收集条数（arr 满则停止，剩余留给下次）。
int crn_npreg_take(fiber* gr, void** arr, int cap) {
    int n = 0;
    void** link = &gr->nphead;
    while (*link != nilptr) {
        npreg_hdr* h = (npreg_hdr*)*link;
        if (n >= cap) { break; }
        int st = __atomic_load_n(&h->npstate, __ATOMIC_ACQUIRE);
        if (st == NPREG_REGD) {
            int exp = NPREG_REGD;
            if (__atomic_compare_exchange_n(&h->npstate, &exp, NPREG_DRAINING,
                                            0, __ATOMIC_ACQ_REL, __ATOMIC_ACQUIRE)) {
                void* taken = *link;
                *link = h->npnext;   // 摘除
                arr[n++] = taken;
                continue;
            }
            __atomic_fetch_or(&h->npstate, NPREG_DYING, __ATOMIC_ACQ_REL);
        } else if (st & NPREG_CLAIMED) {
            __atomic_fetch_or(&h->npstate, NPREG_DYING, __ATOMIC_ACQ_REL);
        } else if (st == NPREG_DONE) {
            if (n >= cap) { break; }
            // 回调已收尾：obj 已由唤醒路径释放，令牌本体摘除弃置（GC）
            void* taken = *link;
            *link = h->npnext;
            arr[n++] = taken;
            continue;
        }
        link = (void**)&h->npnext;
    }
    return n;
}

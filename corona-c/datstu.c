
#include <collectc/hashtable.h>
#include <collectc/array.h>
#include <collectc/queue.h>

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
    pmutex_init(&mp->mu, nilptr);
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

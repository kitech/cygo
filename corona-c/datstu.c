
#include <collectc/hashtable.h>
#include <collectc/array.h>
#include <collectc/queue.h>

// aim thread safe data struct

typedef struct CrnHashTable CrnHashTable;
struct CrnHashTable {
    HashTable* ht;
    pmutex_t mu;
};
typedef struct CrnArray CrnArray;
struct CrnArray {
    Array* arr;
    pmutex_t mu;
};
typedef struct CrnQueue CrnQueue;
struct CrnQueue {
    Queue* qo;
    pmutex_t mu;
};


CrnHashTable* CrnHashTable_new() {
    CrnHashTable* ht = (CrnHashTable*)crn_gc_malloc(sizeof(CrnHashTable));
    HashTableConf htconf;
    hashtable_conf_init(&htconf);
    int rv = hashtable_new_conf(&htconf, &ht->ht);
    assert(rv == CC_OK);
    return ht;
}



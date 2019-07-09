#include <assert.h>

#include <collectc/hashtable.h>

#include "cxrtbase.h"

extern void* cxmalloc(size_t size);
extern void* cxcalloc(size_t blocks, size_t size);
extern void cxfree(void* ptr);

static
int __attribute__((no_instrument_function))
cxhashtable_cmp_uintptr(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((uintptr_t)(key1) > (uintptr_t)(key2)) return 1;
    else return -1;
}

static
int __attribute__((no_instrument_function))
cxhashtable_cmp_cxstr(const void *key1, const void *key2) {
    return cxstring_cmp((cxstring*)key1, (cxstring*)key2);
}

static
size_t cxhashtable_hash_cxstr(const void *key, int len, uint32_t seed) {
    cxstring* s = (cxstring*)key;
    return hashtable_hash_string(s->ptr, s->len, seed);
}

static
HashTable* cxhashtable_new_conf(HashTableConf* htconf) {
    htconf->key_length = sizeof(void*);
    htconf->mem_alloc = cxmalloc;
    htconf->mem_calloc = cxcalloc;
    htconf->mem_free = cxfree;

    HashTable* out = 0;
    int rv = hashtable_new_conf(htconf, &out);
    assert(rv == CC_OK);
    return out;
}

HashTable* cxhashtable_new_uintptr() {
    HashTableConf htconf = {0};
    hashtable_conf_init(&htconf);
    htconf.hash = hashtable_hash_ptr;
    htconf.key_compare = cxhashtable_cmp_uintptr;

    return cxhashtable_new_conf(&htconf);
}
HashTable* cxhashtable_new() {
    return cxhashtable_new_uintptr();
}
HashTable* cxhashtable_new_cxstr() {
    HashTableConf htconf = {0};
    hashtable_conf_init(&htconf);

    htconf.hash = cxhashtable_hash_cxstr;
    htconf.key_compare = cxhashtable_cmp_cxstr;

    return cxhashtable_new_conf(&htconf);
}

HashTable* cxhashtable_new_cstr() {
    HashTableConf htconf = {0};
    hashtable_conf_init(&htconf);

    htconf.hash = hashtable_hash_string;
    htconf.key_compare = cc_common_cmp_str;

    return cxhashtable_new_conf(&htconf);
}

size_t cxhashtable_hash_str(const char *key) {
    return hashtable_hash_string(key, strlen(key), 0);
}

size_t cxhashtable_hash_str2(const char *key, int len) {
    return hashtable_hash_string(key, len, 0);
}

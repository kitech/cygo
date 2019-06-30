#include <assert.h>

#include <collectc/hashtable.h>


static
int
__attribute__((no_instrument_function))
cxhashtable_cmp_int(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((uintptr_t)(key1) > (uintptr_t)(key2)) return 1;
    else return -1;
}

extern void* cxmalloc(size_t size);
extern void* cxcalloc(size_t blocks, size_t size);
extern void cxfree(void* ptr);

HashTable* cxhashtable_new() {
    HashTableConf htconf = {0};
    hashtable_conf_init(&htconf);
    htconf.key_length = sizeof(void*);
    htconf.hash = hashtable_hash_ptr;
    htconf.key_compare = cxhashtable_cmp_int;
    htconf.mem_alloc = cxmalloc;
    htconf.mem_calloc = cxcalloc;
    htconf.mem_free = cxfree;

    HashTable* out = 0;
    int rv = hashtable_new_conf(&htconf, &out);
    assert(rv == CC_OK);
    return out;
}

size_t cxhashtable_hash_str(const char *key) {
    return hashtable_hash_string(key, strlen(key), 0);
}

size_t cxhashtable_hash_str2(const char *key, int len) {
    return hashtable_hash_string(key, len, 0);
}

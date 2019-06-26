
#include <collectc/hashtable.h>


static
int
__attribute__((no_instrument_function))
cxhashtable_cmp_int(const void *key1, const void *key2) {
    if (key1 == key2) return 0;
    else if((uintptr_t)(key1) > (uintptr_t)(key2)) return 1;
    else return -1;
}

int cxhashtable_new(HashTable** out) {
    HashTableConf htconf = {0};
    hashtable_conf_init(&htconf);
    htconf.key_length = sizeof(void*);
    htconf.hash = hashtable_hash_ptr;
    htconf.key_compare = cxhashtable_cmp_int;

    return hashtable_new_conf(&htconf, out);
}

size_t cxhashtable_hash_str(const char *key) {
    return hashtable_hash_string(key, strlen(key), 0);
}

size_t cxhashtable_hash_str2(const char *key, int len) {
    return hashtable_hash_string(key, len, 0);
}

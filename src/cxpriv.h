#ifndef _CXPRIV_H_
#define _CXPRIV_H_

#include <stdlib.h>
#include <stdio.h>
#include <assert.h>
#include <string.h>
#include <stdint.h>
// #include <stdbool.h>

#include "cxmemory.h"
#include "cxstring.h"
#include "cxhashtable.h"

// cxstring begin
typedef struct cxstring {
    char* ptr;
    int len;
} cxstring;

cxstring* cxstring_new_cstr(char* s);
cxstring* cxstring_new_cstr2(char* s, int len);
// cxstring end

// cxhashtable begin
size_t cxhashtable_hash_str(const char *key);
size_t cxhashtable_hash_str2(const char *key, int len);
// cxhashtable end

#endif

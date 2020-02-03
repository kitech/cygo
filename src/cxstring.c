#include <stdbool.h>
#include <ctype.h>

#include "cxrtbase.h"

/* typedef struct cxstring { */
/*     char* ptr; */
/*     int len; */
/* } cxstring; */

cxstring* cxstring_new() {
    return (cxstring*)cxmalloc(sizeof(cxstring));
}

void cxstring_free(cxstring* sobj) {
    cxfree(sobj->ptr);
    sobj->ptr = 0;
    sobj->len = 0;
    cxfree(sobj);
}

int cxstring_len(cxstring* sobj) { return sobj == nilptr ? 0 : sobj->len; }
int cxstring_at(cxstring* sobj, int idx) { return sobj->ptr[idx]; }

// for null terminated string
cxstring* cxstring_new_cstr(char* s) {
    cxstring* t = cxstring_new();
    t->ptr = cxstrdup(s);
    t->len = strlen(s);
    return t;
}
// for non null terminated string
cxstring* cxstring_new_cstr2(char* s, int len) {
    cxstring* t = cxstring_new();
    t->ptr = cxstrndup(s, len);
    t->len = len;
    return t;
}
cxstring* cxstring_new_char(char ch) {
    cxstring* t = cxstring_new();
    t->ptr = cxmalloc(8);
    t->len = 1;
    t->ptr[0] = ch;
    return t;
}
cxstring* cxstring_new_rune(rune ch) {
    cxstring* t = cxstring_new();
    t->ptr = cxmalloc(8);
    t->len = 3;
    char* p = (char*)&ch;
    t->ptr[0] = p[0];
    t->ptr[1] = p[1];
    t->ptr[2] = p[2];
    return t;
}

// for null terminated string
char* cxstring_to_cstr(cxstring* sobj) {
    return (char*)cxstrndup(sobj->ptr, sobj->len);
}
// for non null terminated string
char* cxstring_to_cstr2(cxstring* sobj, int len) {
    return (char*)cxstrndup(sobj->ptr, len);
}

cxstring* cxstring_add(cxstring* s0, cxstring* s1) {
    cxstring* ns = cxstring_new();
    int rlen = s0->len + s1->len;
    char* rs0 = cxmalloc(rlen+1);
    memcpy(rs0, s0->ptr, s0->len);
    memcpy(rs0+s0->len, s1->ptr, s1->len);
    ns->ptr = rs0;
    ns->len = rlen;
    return ns;
}

cxstring* cxstring_sub(cxstring* s0, int start, int end) {
    assert(s0 != nilptr);
    cxstring* ns = cxstring_new();
    int rlen = end - start;
    char* rs = cxmalloc(rlen+1);
    memcpy(rs, s0->ptr+start, rlen);
    ns->ptr = rs;
    ns->len = rlen;
    return ns;
}

cxstring* cxstring_double(double v) {
    cxstring* ns = cxstring_new();
    char buf[64] = {0};
    snprintf(buf, sizeof(buf)-1, "%f", v);
    ns->ptr = cxstrdup(buf);
    ns->len = strlen(ns->ptr);
    return ns;
}
cxstring* cxstring_float(float v) { return cxstring_double((double)(v)); }

cxstring* cxstring_int64(int64_t v) {
    cxstring* ns = cxstring_new();
    char buf[64] = {0};
    snprintf(buf, sizeof(buf)-1, "%ld", v);
    ns->ptr = cxstrdup(buf);
    ns->len = strlen(ns->ptr);
    return ns;
}
cxstring* cxstring_int(int v) { return cxstring_int64(v); }

cxstring* cxstring_uint64(uint64_t v) {
    cxstring* ns = cxstring_new();
    char buf[64] = {0};
    snprintf(buf, sizeof(buf)-1, "%uld", v);
    ns->ptr = cxstrdup(buf);
    ns->len = strlen(ns->ptr);
    return ns;
}
cxstring* cxstring_uint(unsigned int v) { return cxstring_uint64(v); }

static cxstring cxtruestr = {.ptr="true", .len=4};
static cxstring cxfalsestr = {.ptr="false", .len=5};
cxstring* cxstring_bool(_Bool v) {
    return v == 1 ? &cxtruestr : &cxfalsestr;
}


int cxstring_index2(cxstring* s0, cxstring* sub) {
    char* pos = memmem(s0->ptr, s0->len, sub->ptr, sub->len);
    if (pos == 0) return -1;
    return pos - s0->ptr;
}

int cxstring_rindex(cxstring* s0, cxstring* sub) {
    int bpos = s0->len - sub->len;
    for (; bpos >= 0; bpos --) {
        char* pos = memmem(s0->ptr+bpos, sub->len, sub->ptr, sub->len);
        if (pos != 0) {
            return bpos;
        }
    }
    return -1;
}

bool char_is_space(char c) { return isspace(c); }

cxstring* cxstring_trim_space(cxstring* s) {
    cxstring* ns = cxstring_new();
    char* rs = cxmalloc(s->len+1);
    char* p = rs;
    for (int i = 0; i < s->len; i++) {
        if (char_is_space(s->ptr[i])) continue;
        *p++ = s->ptr[i];
    }
    ns->ptr = rs;
    ns->len = p - rs;
    return ns;
}

cxstring* cxstring_trim(cxstring* s, cxstring* cutset) {
    int bpos = 0;
    int epos = s->len-1;
    for (int i = 0; i < s->len; i++) {
        bool found = false;
        for (int j = 0; j < cutset->len; j++) {
            if (s->ptr[i] == cutset->ptr[j]) {
                found = true;
            }
        }
        if (!found) {
            bpos = i;
            break;
        }
    }
    for (int i = s->len-1; i>=0; i--) {
        bool found = false;
        for (int j = 0; j < cutset->len; j++) {
            if (s->ptr[i] == cutset->ptr[j]) {
                found = true;
            }
        }
        if (!found) {
            epos = i;
            break;
        }
    }
    cxstring* ns = cxstring_new();
    ns->ptr = cxstrndup(s->ptr+bpos, epos - bpos);
    ns->len = epos - bpos;
    return ns;
}

cxstring* cxstring_rtrim(cxstring* s, cxstring* cutset) {
    int bpos = 0;
    int epos = s->len-1;
    for (int i = s->len-1; i>=0; i--) {
        bool found = false;
        for (int j = 0; j < cutset->len; j++) {
            if (s->ptr[i] == cutset->ptr[j]) {
                found = true;
            }
        }
        if (!found) {
            epos = i;
            break;
        }
    }
    cxstring* ns = cxstring_new();
    ns->ptr = cxstrndup(s->ptr+bpos, epos - bpos);
    ns->len = epos - bpos;
    return ns;
}

cxstring* cxstring_ltrim(cxstring* s, cxstring* cutset) {
    int bpos = 0;
    int epos = s->len-1;
    for (int i = 0; i < s->len; i++) {
        bool found = false;
        for (int j = 0; j < cutset->len; j++) {
            if (s->ptr[i] == cutset->ptr[j]) {
                found = true;
            }
        }
        if (!found) {
            bpos = i;
            break;
        }
    }
    cxstring* ns = cxstring_new();
    ns->ptr = cxstrndup(s->ptr+bpos, epos - bpos);
    ns->len = epos - bpos;
    return ns;
}

cxstring* cxstring_to_lower(cxstring* s) {
    cxstring* ns = cxstring_new();
    char* rs = cxmalloc(s->len+1);
    char* p = rs;
    for (int i = 0; i < s->len; i++) {
        *p++ = tolower(s->ptr[i]);
    }
    ns->ptr = rs;
    ns->len = s->len;
    return ns;
}
cxstring* cxstring_to_upper(cxstring* s) {
    cxstring* ns = cxstring_new();
    char* rs = cxmalloc(s->len+1);
    char* p = rs;
    for (int i = 0; i < s->len; i++) {
        *p++ = toupper(s->ptr[i]);
    }
    ns->ptr = rs;
    ns->len = s->len;
    return ns;
}

int cxstring_cmp(cxstring* s0, cxstring* s1) {
    if (s0->len <= s1->len) {
        return memcmp(s0->ptr, s1->ptr, s0->len);
    }
    return memcmp(s0->ptr, s1->ptr, s1->len);
}

bool cxstring_eq(cxstring* s0, cxstring* s1) {
    if (s0 == nilptr && s1 == nilptr) return true;
    if (s0 == nilptr) {
        if (s1 != nilptr && s1->len == 0) return true;
    }
    if (s1 == nilptr) {
        if (s0 != nilptr && s0->len == 0) return true;
    }
    if (s0->len != s1->len) return false;
    return memcmp(s0->ptr, s1->ptr, s0->len) == 0;
}
bool cxstring_ne(cxstring* s0, cxstring* s1) { return !cxstring_eq(s0, s1); }
bool cxstring_lt(cxstring* s0, cxstring* s1) {
    for (int i = 0; i < s0->len; i++) {
        if (i >= s1->len || s0->ptr[i] > s1->ptr[i]) {
            return 0;
        } else if (s0->ptr[i] < s1->ptr[i]) {
            return 1;
        }
    }

    if (s0->len < s1->len) {
        return 1;
    }
    return 0;
}
bool cxstring_le(cxstring* s0, cxstring* s1) {
    return cxstring_lt(s0, s1) || cxstring_eq(s0, s1);
}
bool cxstring_gt(cxstring* s0, cxstring* s1) {
    return !cxstring_le(s0, s1);
}
bool cxstring_ge(cxstring* s0, cxstring* s1) {
    return !cxstring_lt(s0, s1);
}

cxstring* cxstring_dup(cxstring* s) {
    cxstring* ns = cxstring_new();
    ns->ptr = cxstrndup(s->ptr, s->len);
    ns->len = s->len;
    return ns;
}
cxstring* cxstring_title(cxstring* s) {
    cxstring* ns = cxstring_dup(s);
    ns->ptr[0] = toupper(ns->ptr[0]);
    return ns;
}

int cxstring_to_int(cxstring* s) {
    char* t = cxstrndup(s->ptr, s->len);
    int v = atoi(t);
    cxfree(t);
    return v;
}

float cxstring_to_float(cxstring* s) {
    char* t = cxstrndup(s->ptr, s->len);
    float v = atof(t);
    cxfree(t);
    return v;
}

char* CString(cxstring* s) {
    return cxstrndup(s->ptr, s->len);
}
cxstring* GoString(char* s) {
    return cxstring_new_cstr(s);
}
cxstring* GoStringN(char* s, int n) {
    return cxstring_new_cstr2(s, n);
}


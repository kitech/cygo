#include "cxrtbase.h"

cxeface cxeface_new_of(void* data) {
    cxeface efv = {0};
    efv.data = data;
    return efv;
}
cxeface cxeface_new_of2(void* data, int sz) {
    cxeface efv = {0};
    efv.data = cxmemdup(data, sz);
    return efv;
}

cxeface cxeface_new_int(int64 v) {
    return cxeface_new_of(cxmemdup(&v, sizeof(v)));
}

cxeface cxeface_new_float(double v) {
    return cxeface_new_of(cxmemdup(&v, sizeof(v)));
}

cxeface cxeface_new_string(cxstring* s) {
    return cxeface_new_of(s);
}



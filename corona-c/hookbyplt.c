
#include "plthook.h"

void crn_dump_plt_entries(const char* filename) {
    plthook_t *plthook;
    unsigned int pos = 0; /* This must be initialized with zero. */
    const char *name;
    void **addr;

    if (plthook_open(&plthook, filename) != 0) {
        printf("plthook_open error: %s\n", plthook_error());
        return -1;
    }
    while (plthook_enum(plthook, &pos, &name, &addr) == 0) {
        printf("%p(%p) %s\n", addr, *addr, name);
    }
    plthook_close(plthook);
    return 0;
}

// only dump about 20 symbols, why?
void crn_dump_libc_plt() {
    crn_dump_plt_entries("libc.so.6");
}


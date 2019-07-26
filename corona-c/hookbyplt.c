#include <stdlib.h>
#include <stdio.h>

#include "plthook.h"

int crn_dump_plt_entries(const char* filename) {
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

static ssize_t my_recv(int sockfd, void *buf, size_t len, int flags)
{
    ssize_t rv;

    // ... do your task: logging, etc. ...
    rv = recv(sockfd, buf, len, flags); /* call real recv(). */
    // ... do your task: logging, check received data, etc. ...
    return rv;
}

// why cannot found "recv" function in libc?
int install_hook_function()
{
    plthook_t *plthook;

    if (plthook_open(&plthook, "libc.so.6") != 0) {
        printf("plthook_open error: %s\n", plthook_error());
        return -1;
    }
    if (plthook_replace(plthook, "recv", (void*)my_recv, NULL) != 0) {
        printf("plthook_replace error: %s\n", plthook_error());
        plthook_close(plthook);
        return -1;
    }
    plthook_close(plthook);
    return 0;
}

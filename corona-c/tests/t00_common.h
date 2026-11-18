#ifndef T00_COMMON_H
#define T00_COMMON_H

#include <pthread.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdio.h>
#include <assert.h>
#include <string.h>
#include <time.h>
#include <stdbool.h>
#include <stdint.h>

#include <gc.h>
#include "corona.h"
#include "coronagc.h"
#include "coronapriv.h"
#include "crnpub.h"
#include "hchan.h"
#include "futex.h"
#include "rxilog.h"
#include "corona_util.h"

// ---- hchan/select API: not exposed in public headers, declare here ----
extern crn_hchan* crn_hchan_new(int cap);
extern int crn_hchan_close(crn_hchan* hc);
extern int crn_hchan_send(crn_hchan* hc, void* data);
extern int crn_hchan_recv(crn_hchan* hc, void** pdata);
extern bool crn_goselect(int* rcasi, crn_scase** cas0, int ncases);

// ---- check helpers ----
int t_fails = 0;
#define CHECK(cond, ...) do { \
    if (!(cond)) { \
        t_fails++; \
        printf("FAIL %s:%d: ", __FILE__, __LINE__); \
        printf(__VA_ARGS__); printf("\n"); \
    } \
} while(0)
#define CHECK_EQ(a, b, ...) CHECK((a) == (b), __VA_ARGS__)
#define t_report_and_exit() do { \
    if (t_fails == 0) { printf("PASS %s\n", __FILE__); exit(0); } \
    printf("FAIL %s (%d checks)\n", __FILE__, t_fails); exit(1); \
} while(0)

// ---- shared init, mirrors main.c ----
static inline void t_init() {
    extern void initHook(); initHook();
    corona* nr = crn_new();
    crn_init(nr);
    crn_wait_init_done(nr);
}

// join helper: N fibers each send their id on the done chan, controller recvs N times
typedef struct t_joiner { crn_hchan* done; int n; } t_joiner;
static inline void t_join_send(crn_hchan* done) {
    int rv = crn_hchan_send(done, (void*)(uintptr_t)crn_goid());
    CHECK_EQ(rv, 1, "join send rv=%d", rv);
}
static inline void t_join_wait(crn_hchan* done, int n) {
    for (int i = 0; i < n; i++) {
        void* v = (void*)0xdeadbeef;
        int rv = crn_hchan_recv(done, &v);
        CHECK_EQ(rv, 1, "join recv rv=%d i=%d", rv, i);
        CHECK(v != (void*)0xdeadbeef && v != nilptr, "join recv bad val i=%d", i);
    }
}
static inline void t_usleep_ms(int ms) { usleep(ms * 1000); }

#endif

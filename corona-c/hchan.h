#ifndef _HCHAN_H_
#define _HCHAN_H_

#include <stdbool.h>
// #include <threads.h>

// typedef struct hchan hchan;

typedef struct hchan {
    chan_t* c;
    int cap;
    pmutex_t lock;
    szqueue_t* recvq; // fiber*
    szqueue_t* sendq; // fiber*
    bool closed;
} hchan;

typedef struct hcdata  {
    int grid;
    int mcid;
    fiber* gr;
    void* sdelem;
    void** rvelem;
} hcdata;

int hchan_is_closed(hchan* hc);
int hchan_cap(hchan* hc);
int hchan_len(hchan* hc);

hcdata* hcdata_new(fiber* gr);
void hcdata_free(hcdata* d);

typedef struct scase scase;
scase* scase_new(hchan* hc, uint16_t kind, void* elem);
void scase_free(scase* cas);

#endif


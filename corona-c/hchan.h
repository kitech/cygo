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
    int closed;
} hchan;

typedef struct hcdata  {
    int grid;
    int mcid;
    fiber* gr;
    void* sdelem;
    void** rvelem;
    int wokeby_grid;
    int wokeby_mcid;
    int wokecase; // caseSend/caseRecv
    fiber* wokeby; //
    void* wokehc; // hchan*
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


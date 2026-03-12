#ifndef _HCHAN_H_
#define _HCHAN_H_

#include <stdbool.h>
#include <stdint.h>
// #include <threads.h>

// typedef struct hchan hchan;

typedef struct crn_hchan {
    chan_t* c;
    int cap;
    pmutex_t lock;
    szqueue_t* recvq; // fiber*
    szqueue_t* sendq; // fiber*
    int closed;
} crn_hchan;

typedef struct crn_hcdata  {
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
} crn_hcdata;

int crn_hchan_is_closed(crn_hchan* hc);
int crn_hchan_cap(crn_hchan* hc);
int crn_hchan_len(crn_hchan* hc);

crn_hcdata* crn_hcdata_new(fiber* gr);
void crn_hcdata_free(crn_hcdata* d);

typedef struct crn_scase crn_scase;
crn_scase* crn_scase_new(crn_hchan* hc, uint16_t kind, void* elem);
void crn_scase_free(crn_scase* cas);

#endif

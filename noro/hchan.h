#ifndef _HCHAN_H_
#define _HCHAN_H_

#include <stdbool.h>
#include <threads.h>

// typedef struct hchan hchan;

typedef struct hchan {
    chan_t* c;
    int cap;
    mtx_t lock;
    szqueue_t* recvq; // goroutine*
    szqueue_t* sendq; // goroutine*
    bool closed;
} hchan;


#endif


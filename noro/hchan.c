
#include "chan.h"
#include "hchan.h"
#include "noropriv.h"

#define HKDEBUG 1
#define linfo(fmt, ...)                                                 \
    do { if (HKDEBUG) fprintf(stderr, "%s:%d:%s ", __FILE__, __LINE__, __FUNCTION__); } while (0); \
    do { if (HKDEBUG) fprintf(stderr, fmt, __VA_ARGS__); } while (0) ;  \
    do { if (HKDEBUG) fflush(stderr); } while (0) ;

// wrapper chan_t with goroutine integeration

typedef struct hchan {
    chan_t* c;
    int cap;
    queue_t* recvq; // goroutine*
    queue_t* sendq; // goroutine*
    mtx_t rcvqmu;
    mtx_t sndqmu;
} hchan;

hchan* hchan_new(int cap) {
    hchan* hc = (hchan*)calloc(1, sizeof(hchan));
    hc->c = chan_init(cap);
    hc->cap = cap;
    hc->recvq = queue_init(900000);
    hc->sendq = queue_init(900000);
}

void hchan_dispose(hchan* hc) {
    chan_dispose(hc->c);
    free(hc);
}

int hchan_close(hchan* hc) {
    return chan_close(hc->c);
}
int hchan_is_closed(hchan* hc) {
    return chan_is_closed(hc->c);
}

int hchan_send(hchan* hc, void* data) {
    if (hc->cap == 0) {
        // if any goroutine waiting, put data to it elem and then wakeup
        // else put self to sendq and then parking self

        mtx_lock(&hc->rcvqmu);
        goroutine* gr = (goroutine*)queue_remove(hc->recvq);
        mtx_unlock(&hc->rcvqmu);
        if (gr != nilptr) {
            bool swaped = atomic_casptr(&gr->hcelem, nilptr, data);
            if (swaped) {
                noro_processor_resume_some(gr);
                return 1;
            } else {
                linfo("wtf, cannot set rcvg hcelem %d\n", gr->id);
                assert(1==2);
            }
        } else {
            // put data to my hcelem, put self to sendq, then parking self
            goroutine* mygr = noro_goroutine_getcur();
            assert(mygr != nilptr);
            atomic_setptr(&mygr->hcelem, data);
            mtx_lock(&hc->sndqmu);
            queue_add(hc->sendq, mygr);
            mtx_unlock(&hc->sndqmu);
            noro_processor_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    } else {
        // if not full, enqueue data
        // if full, put self in sendq, then parking
        int bufsz = chan_size(hc->c);
        if (bufsz < hc->cap) {
            chan_send(hc->c, data);
            mtx_lock(&hc->rcvqmu);
            goroutine* gr = (goroutine*)queue_remove(hc->recvq);
            mtx_unlock(&hc->rcvqmu);
            if (gr != nilptr) {
                noro_processor_resume_some(gr);
            }
            return 1;
        }else{
            // if has recvq, put to peer hcelem, wakeup peer and return
            // put data to my hcelem, put self to sendq, then parking self
            mtx_lock(&hc->rcvqmu);
            goroutine* gr = (goroutine*)queue_remove(hc->recvq);
            mtx_unlock(&hc->rcvqmu);
            if (gr != nilptr) {
                gr->hcelem = data;
                noro_processor_resume_some(gr);
                return 1;
            }

            goroutine* mygr = noro_goroutine_getcur();
            assert(mygr != nilptr);
            atomic_setptr(&mygr->hcelem, data);
            mtx_lock(&hc->sndqmu);
            queue_add(hc->sendq, mygr);
            mtx_unlock(&hc->sndqmu);

            noro_processor_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    }
}

int hchan_recv(hchan* hc, void** pdata) {
    if (hc->cap == 0) {
        // if have elem not nil, get it
        // else if any sendq, wakeup them,
        // else parking
        mtx_lock(&hc->sndqmu);
        goroutine* gr = (goroutine*)queue_remove(hc->sendq);
        mtx_unlock(&hc->sndqmu);
        if (gr != nilptr) {
            *pdata = gr->hcelem;
            gr->hcelem = nilptr;
            noro_processor_resume_some(gr);
            return 1;
        } else {
            goroutine* mygr = noro_goroutine_getcur();
            assert(mygr != nilptr);
            mtx_lock(&hc->rcvqmu);
            queue_add(hc->recvq, mygr);
            mtx_unlock(&hc->rcvqmu);
            noro_processor_yield(-1, YIELD_TYPE_CHAN_RECV);
            *pdata = mygr->hcelem;
            mygr->hcelem = nilptr;
            return 1;
        }
    }else{
        // if size > 0, recv right now
        // if empty then put self in recvq, then parking
        // else parking
        int bufsz = chan_size(hc->c);
        if (bufsz > 0) {
            chan_recv(hc->c, pdata);
            return 1;
        }

        mtx_lock(&hc->sndqmu);
        goroutine* gr = queue_remove(hc->sendq);
        mtx_unlock(&hc->sndqmu);
        if (gr != nilptr) {
            *pdata = gr->hcelem;
            gr->hcelem = nilptr;
            noro_processor_resume_some(gr);
            return 1;
        }

        goroutine* mygr = noro_goroutine_getcur();
        assert(mygr != nilptr);
        mtx_lock(&hc->rcvqmu);
        queue_add(hc->recvq, mygr);
        mtx_unlock(&hc->rcvqmu);
        noro_processor_yield(-1, YIELD_TYPE_CHAN_RECV);
        *pdata = mygr->hcelem;
        return 1;
    }
}

// https://ninokop.github.io/2017/11/07/Go-Channel%E7%9A%84%E5%AE%9E%E7%8E%B0/

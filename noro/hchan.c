
#include "chan.h"
#include "hchan.h"
#include "noropriv.h"

// wrapper chan_t with goroutine integeration


hchan* hchan_new(int cap) {
    hchan* hc = (hchan*)calloc(1, sizeof(hchan));
    hc->c = chan_init(cap);
    hc->cap = cap;

    // only support max 32 concurrent goroutines on one hchan
    // should enough
    hc->recvq = queue_init(32);
    hc->sendq = queue_init(32);
}

int hchan_close(hchan* hc) {
    if (hc->closed) {
        return true;
    }

    goroutine* mygr = noro_goroutine_getcur();
    assert(mygr != nilptr);

    mtx_lock(&hc->lock);
    hc->closed = true;
    int qsz = 0;
    qsz = hc->recvq->size;
    if (qsz > 0) { linfo("discard recvq %d\n", qsz); }
    while (hc->recvq != nilptr) {
        goroutine* gr = (goroutine*)queue_remove(hc->recvq);
        if (gr == nilptr) {
            break;
        }
        gr->wokeby = mygr;
        gr->wokehc = hc;
        gr->wokecase = caseClose;
        noro_processor_resume_some(gr, 0);
    }
    if (hc->recvq != nilptr) queue_dispose(hc->recvq);

    qsz = hc->sendq->size;
    if (qsz > 0) { linfo("discard sendq %d\n", qsz); }
    while(hc->sendq != nilptr) {
        goroutine* gr = (goroutine*)queue_remove(hc->sendq);
        if (gr == nilptr) {
            break;
        }
        gr->wokeby = mygr;
        gr->wokehc = hc;
        gr->wokecase = caseClose;
        noro_processor_resume_some(gr, 0);
    }
    if (hc->sendq != nilptr) queue_dispose(hc->sendq);

    int bufsz = chan_size(hc->c);
    if (bufsz > 0) { linfo("Warning, discard bufsz %d\n", bufsz); }
    chan_dispose(hc->c);
    hc->c = nilptr;
    bzero(hc, sizeof(hchan));
    free(hc);
    mtx_unlock(&hc->lock);
    return true;
}
int hchan_is_closed(hchan* hc) {
    return hc->closed;
}
int hchan_cap(hchan* hc) { return hc->cap; }
int hchan_len(hchan* hc) { return chan_size(hc->c); }

// TODO when sending/recving, hchan closed case
int hchan_send(hchan* hc, void* data) {
    mtx_lock(&hc->lock);

    goroutine* mygr = noro_goroutine_getcur();
    assert(mygr != nilptr);

    if (hc->cap == 0) {
        // if any goroutine waiting, put data to it elem and then wakeup
        // else put self to sendq and then parking self

        goroutine* gr = (goroutine*)queue_remove(hc->recvq);
        if (gr != nilptr) {
            bool swaped = atomic_casptr(&gr->hcelem, invlidptr, data);
            if (swaped) {
                linfo("resume recver %d on %d/%d\n", gr->id, mygr->id, mygr->mcid);
                gr->wokeby = mygr;
                gr->wokehc = hc;
                gr->wokecase = caseRecv;
                mtx_unlock(&hc->lock);
                noro_processor_resume_some(gr, 0);
                return 1;
            } else {
                linfo("wtf, cannot set rcvg hcelem %d, swaped %d elem %p\n",
                      gr->id, swaped, gr->hcelem);
                // assert(swaped == true);
            }
        }

        // cannot send directly
        {
            // put data to my hcelem, put self to sendq, then parking self
            atomic_setptr(&mygr->hcelem, data);
            queue_add(hc->sendq, mygr);
            linfo("yield me sender %d/%d\n", mygr->id, mygr->mcid);
            mygr->hclock = &hc->lock;
            // mtx_unlock(&hc->lock);
            noro_processor_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    } else {
        // if not full, enqueue data
        // if full, put self in sendq, then parking
        int bufsz = chan_size(hc->c);
        if (bufsz < hc->cap) {
            chan_send(hc->c, data);
            goroutine* gr = (goroutine*)queue_remove(hc->recvq);
            if (gr != nilptr) {
                gr->wokeby = mygr;
                noro_processor_resume_some(gr, 0);
            }
            mtx_unlock(&hc->lock);
            return 1;
        }else{
            // if has recvq, put to peer hcelem, wakeup peer and return
            // put data to my hcelem, put self to sendq, then parking self
            goroutine* gr = (goroutine*)queue_remove(hc->recvq);
            if (gr != nilptr) {
                gr->hcelem = data;
                noro_processor_resume_some(gr, 0);
                mtx_unlock(&hc->lock);
                return 1;
            }

            atomic_setptr(&mygr->hcelem, data);
            queue_add(hc->sendq, mygr);

            mtx_unlock(&hc->lock);
            noro_processor_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    }
}

int hchan_recv(hchan* hc, void** pdata) {
    mtx_lock(&hc->lock);

    goroutine* mygr = noro_goroutine_getcur();
    assert(mygr != nilptr);

    if (hc->cap == 0) {
        // if have elem not nil, get it
        // else if any sendq, wakeup them,
        // else parking

        goroutine* gr = (goroutine*)queue_remove(hc->sendq);
        if (gr != nilptr) {
            void* oldptr = atomic_getptr(&gr->hcelem);
            bool swaped = atomic_casptr(&gr->hcelem, oldptr, invlidptr);
            if (swaped && oldptr != invlidptr) {
                *pdata = oldptr;
                linfo("resume sender %d on %d/%d\n", gr->id, mygr->id, mygr->mcid);
                gr->wokeby = mygr;
                gr->wokehc = hc;
                gr->wokecase = caseSend;
                mtx_unlock(&hc->lock);
                noro_processor_resume_some(gr, 0);
                return 1;
            } else {
                linfo("wtf, cannot set sndg hcelem %d, swaped %d elem %p\n",
                      gr->id, swaped, oldptr);
                // assert(swaped == true);
            }
        }

        // cannot recv directly
        {
            queue_add(hc->recvq, mygr);
            // linfo("chan recv %d\n", mygr->id);
            linfo("yield me recver %d/%d, qc %d\n", mygr->id, mygr->mcid, hc->recvq->size);
            mygr->hclock = &hc->lock;
            // mtx_unlock(&hc->lock);
            noro_processor_yield(-1, YIELD_TYPE_CHAN_RECV);
            mtx_lock(&hc->lock);
            void* oldptr = atomic_getptr(&mygr->hcelem);
            // assert(oldptr != invlidptr);
            bool swaped = atomic_casptr(&mygr->hcelem, oldptr, invlidptr);
            assert(swaped == true);
            *pdata = oldptr;
            assert(*pdata != invlidptr);
            mtx_unlock(&hc->lock);
            return 1;
        }
    }else{
        // if size > 0, recv right now
        // if empty then put self in recvq, then parking
        // else parking
        int bufsz = chan_size(hc->c);
        if (bufsz > 0) {
            chan_recv(hc->c, pdata);
            mtx_unlock(&hc->lock);
            return 1;
        }

        goroutine* gr = queue_remove(hc->sendq);
        if (gr != nilptr) {
            *pdata = gr->hcelem;
            gr->hcelem = nilptr;
            gr->wokeby = mygr;
            mtx_unlock(&hc->lock);
            noro_processor_resume_some(gr, 0);
            return 1;
        }

        queue_add(hc->recvq, mygr);
        mtx_unlock(&hc->lock);
        noro_processor_yield(-1, YIELD_TYPE_CHAN_RECV);
        *pdata = mygr->hcelem;
        return 1;
    }
}

// https://ninokop.github.io/2017/11/07/Go-Channel%E7%9A%84%E5%AE%9E%E7%8E%B0/

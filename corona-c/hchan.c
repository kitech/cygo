
#include "chan.h"
#include "hchan.h"
#include "coronapriv.h"

// wrapper chan_t with fiber integeration


hchan* hchan_new(int cap) {
    hchan* hc = (hchan*)crn_raw_malloc(sizeof(hchan));
    hc->c = chan_init(cap);
    hc->cap = cap;

    // only support max 32 concurrent fibers on one hchan
    // should enough
    hc->recvq = szqueue_init(32);
    hc->sendq = szqueue_init(32);
}

int hchan_close(hchan* hc) {
    if (hc->closed) {
        return true;
    }

    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    pmutex_lock(&hc->lock);
    hc->closed = true;
    int qsz = 0;
    qsz = hc->recvq->size;
    if (qsz > 0) { linfo("discard recvq %d\n", qsz); }
    while (hc->recvq != nilptr) {
        fiber* gr = (fiber*)szqueue_remove(hc->recvq);
        if (gr == nilptr) {
            break;
        }
        gr->wokeby = mygr;
        gr->wokehc = hc;
        gr->wokecase = caseClose;
        crn_procer_resume_one(gr, 0, gr->id, gr->mcid);
    }
    if (hc->recvq != nilptr) szqueue_dispose(hc->recvq);

    qsz = hc->sendq->size;
    if (qsz > 0) { linfo("discard sendq %d\n", qsz); }
    while(hc->sendq != nilptr) {
        fiber* gr = (fiber*)szqueue_remove(hc->sendq);
        if (gr == nilptr) {
            break;
        }
        gr->wokeby = mygr;
        gr->wokehc = hc;
        gr->wokecase = caseClose;
        crn_procer_resume_one(gr, 0, gr->id, gr->mcid);
    }
    if (hc->sendq != nilptr) szqueue_dispose(hc->sendq);

    int bufsz = chan_size(hc->c);
    if (bufsz > 0) { linfo("Warning, discard bufsz %d\n", bufsz); }
    chan_dispose(hc->c);
    hc->c = nilptr;
    bzero(hc, sizeof(hchan));
    free(hc);
    pmutex_unlock(&hc->lock);
    return true;
}
int hchan_is_closed(hchan* hc) {
    return hc->closed;
}
int hchan_cap(hchan* hc) { return hc->cap; }
int hchan_len(hchan* hc) { return chan_size(hc->c); }

// TODO when sending/recving, hchan closed case
int hchan_send(hchan* hc, void* data) {
    pmutex_lock(&hc->lock);

    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    if (hc->cap == 0) {
        // if any fiber waiting, put data to it elem and then wakeup
        // else put self to sendq and then parking self

        fiber* gr = (fiber*)szqueue_remove(hc->recvq);
        if (gr != nilptr) {
            bool swaped = atomic_casptr(&gr->hcelem, invlidptr, data);
            if (swaped) {
                linfo("resume recver %d on %d/%d\n", gr->id, mygr->id, mygr->mcid);
                gr->wokeby = mygr;
                gr->wokehc = hc;
                gr->wokecase = caseRecv;
                pmutex_unlock(&hc->lock);
                crn_procer_resume_one(gr, 0, gr->id, gr->mcid);
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
            szqueue_add(hc->sendq, mygr);
            linfo("yield me sender %d/%d\n", mygr->id, mygr->mcid);
            mygr->hclock = &hc->lock;
            // pmutex_unlock(&hc->lock);
            crn_procer_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    } else {
        // if not full, enqueue data
        // if full, put self in sendq, then parking
        int bufsz = chan_size(hc->c);
        if (bufsz < hc->cap) {
            chan_send(hc->c, data);
            fiber* gr = (fiber*)szqueue_remove(hc->recvq);
            if (gr != nilptr) {
                gr->wokeby = mygr;
                crn_procer_resume_one(gr, 0, gr->id, gr->mcid);
            }
            pmutex_unlock(&hc->lock);
            return 1;
        }else{
            // if has recvq, put to peer hcelem, wakeup peer and return
            // put data to my hcelem, put self to sendq, then parking self
            fiber* gr = (fiber*)szqueue_remove(hc->recvq);
            if (gr != nilptr) {
                gr->hcelem = data;
                crn_procer_resume_one(gr, 0, gr->id, gr->mcid);
                pmutex_unlock(&hc->lock);
                return 1;
            }

            atomic_setptr(&mygr->hcelem, data);
            szqueue_add(hc->sendq, mygr);

            pmutex_unlock(&hc->lock);
            crn_procer_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    }
}

int hchan_recv(hchan* hc, void** pdata) {
    pmutex_lock(&hc->lock);

    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    if (hc->cap == 0) {
        // if have elem not nil, get it
        // else if any sendq, wakeup them,
        // else parking

        fiber* gr = (fiber*)szqueue_remove(hc->sendq);
        if (gr != nilptr) {
            void* oldptr = atomic_getptr(&gr->hcelem);
            bool swaped = atomic_casptr(&gr->hcelem, oldptr, invlidptr);
            if (swaped && oldptr != invlidptr) {
                *pdata = oldptr;
                linfo("resume sender %d on %d/%d\n", gr->id, mygr->id, mygr->mcid);
                gr->wokeby = mygr;
                gr->wokehc = hc;
                gr->wokecase = caseSend;
                pmutex_unlock(&hc->lock);
                crn_procer_resume_one(gr, 0, gr->id, gr->mcid);
                return 1;
            } else {
                linfo("wtf, cannot set sndg hcelem %d, swaped %d elem %p\n",
                      gr->id, swaped, oldptr);
                // assert(swaped == true);
            }
        }

        // cannot recv directly
        {
            szqueue_add(hc->recvq, mygr);
            // linfo("chan recv %d\n", mygr->id);
            linfo("yield me recver %d/%d, qc %d\n", mygr->id, mygr->mcid, hc->recvq->size);
            mygr->hclock = &hc->lock;
            // pmutex_unlock(&hc->lock);
            crn_procer_yield(-1, YIELD_TYPE_CHAN_RECV);
            pmutex_lock(&hc->lock);
            void* oldptr = atomic_getptr(&mygr->hcelem);
            // assert(oldptr != invlidptr);
            bool swaped = atomic_casptr(&mygr->hcelem, oldptr, invlidptr);
            assert(swaped == true);
            *pdata = oldptr;
            assert(*pdata != invlidptr);
            pmutex_unlock(&hc->lock);
            return 1;
        }
    }else{
        // if size > 0, recv right now
        // if empty then put self in recvq, then parking
        // else parking
        int bufsz = chan_size(hc->c);
        if (bufsz > 0) {
            chan_recv(hc->c, pdata);
            pmutex_unlock(&hc->lock);
            return 1;
        }

        fiber* gr = szqueue_remove(hc->sendq);
        if (gr != nilptr) {
            *pdata = gr->hcelem;
            gr->hcelem = nilptr;
            gr->wokeby = mygr;
            pmutex_unlock(&hc->lock);
            crn_procer_resume_one(gr, 0, gr->id, gr->mcid);
            return 1;
        }

        szqueue_add(hc->recvq, mygr);
        pmutex_unlock(&hc->lock);
        crn_procer_yield(-1, YIELD_TYPE_CHAN_RECV);
        *pdata = mygr->hcelem;
        return 1;
    }
}

// https://ninokop.github.io/2017/11/07/Go-Channel%E7%9A%84%E5%AE%9E%E7%8E%B0/

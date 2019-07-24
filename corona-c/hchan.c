
#include "chan.h"
#include "hchan.h"
#include "coronapriv.h"

// wrapper chan_t with fiber integeration

hcdata* hcdata_new(fiber* gr) {
    hcdata* d = (hcdata*)crn_gc_malloc(sizeof(hcdata));
    d->gr = gr;
    d->grid = gr->id;
    d->mcid = gr->mcid;
    return d;
}
void hcdata_free(hcdata* d) {
    d->rvelem = nilptr;
    crn_gc_free(d);
}
void hcdata_woke_set(hcdata*d, fiber* wkgr, hchan* hc, int wkcase, void* elem) {
    d->wokeby = wkgr;
    d->wokeby_grid = wkgr->id;
    d->wokeby_mcid = wkgr->mcid;
    d->wokehc = hc;
    d->wokecase = wkcase;
    if (wkcase == caseSend) {
        d->sdelem = elem;
    }else if (wkcase == caseRecv) {
        *d->rvelem = elem;
    }else{
    }
}

static void hchan_finalizer(void* hc) {
    linfo("hchan dtor %p %d\n", hc, gettid());
    // assert(1==2);
}

hchan* hchan_new(int cap) {
    hchan* hc = (hchan*)crn_gc_malloc(sizeof(hchan));
    crn_set_finalizer(hc, hchan_finalizer);
    hc->c = chan_init(cap);  // why 248 cause hc gced too early?
    hc->cap = cap;

    // only support max 32 concurrent fibers on one hchan
    // should enough
    hc->recvq = szqueue_init(132);
    hc->sendq = szqueue_init(132);

    return hc;
}

int hchan_close(hchan* hc) {
    if (!atomic_casint(&hc->closed, 0, 1)) {
        return true;
    }

    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    pmutex_lock(&hc->lock);
    int qsz = 0;
    qsz = hc->recvq->size;
    if (qsz > 0) { linfo("discard recvq %d\n", qsz); }
    while (hc->recvq != nilptr) {
        hcdata* hcdt = (hcdata*)szqueue_remove(hc->recvq);
        fiber* gr = hcdt->gr;
        if (gr == nilptr) {
            break;
        }
        hcdata_woke_set(hcdt, mygr, hc, caseClose, nilptr);
        crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
        // hcdata_free(hcdt);
    }
    if (hc->recvq != nilptr) szqueue_dispose(hc->recvq);

    qsz = hc->sendq->size;
    if (qsz > 0) { linfo("discard sendq %d\n", qsz); }
    while(hc->sendq != nilptr) {
        hcdata* hcdt = (hcdata*)szqueue_remove(hc->sendq);
        fiber* gr = hcdt->gr;
        if (gr == nilptr) {
            break;
        }
        hcdata_woke_set(hcdt, mygr, hc, caseClose, nilptr);
        crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
        // hcdata_free(hcdt);
    }
    if (hc->sendq != nilptr) szqueue_dispose(hc->sendq);

    int bufsz = chan_size(hc->c);
    if (bufsz > 0) { linfo("Warning, discard bufsz %d\n", bufsz); }
    chan_dispose(hc->c);
    hc->c = nilptr;
    bzero(hc, sizeof(hchan));
    crn_gc_free(hc);
    pmutex_unlock(&hc->lock);
    return true;
}
int hchan_is_closed(hchan* hc) {
    return atomic_getint(&hc->closed);
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

        hcdata* hcdt = (hcdata*)szqueue_remove(hc->recvq);
        if (hcdt != nilptr) {
            fiber* gr = hcdt->gr;
            // assert(gr->id == hcdt->grid);
            hcdata_woke_set(hcdt, mygr, hc, caseRecv, data);
            linfo("resume recver %d/%d by %d/%d\n", hcdt->grid, hcdt->mcid, mygr->id, mygr->mcid);
            pmutex_unlock(&hc->lock);
            crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
            return 1;
        }

        // cannot send directly
        {
            // put data to my hcelem, put self to sendq, then parking self
            hcdata* hcdt = hcdata_new(mygr);
            hcdt->sdelem = data;
            int rv = szqueue_add(hc->sendq, hcdt);
            assert(rv != -1);
            linfo("yield me sender %d/%d %p\n", mygr->id, mygr->mcid, data);
            mygr->hclock = &hc->lock;
            crn_procer_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    } else {
        // if not full, enqueue data
        // if full, put self in sendq, then parking
        int bufsz = chan_size(hc->c);
        if (bufsz < hc->cap) {
            chan_send(hc->c, data);
            hcdata* hcdt = (hcdata*)szqueue_remove(hc->recvq);
            fiber* gr = hcdt->gr;
            if (gr != nilptr) {
                assert(gr->id == hcdt->grid);
                hcdata_woke_set(hcdt, mygr, hc, caseRecv, data);
                crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
            }
            pmutex_unlock(&hc->lock);
            return 1;
        }else{
            // if has recvq, put to peer hcelem, wakeup peer and return
            // put data to my hcelem, put self to sendq, then parking self
            hcdata* hcdt = (hcdata*)szqueue_remove(hc->recvq);
            fiber* gr = hcdt->gr;
            if (gr != nilptr) {
                // assert(gr->id == hcdt->grid);
                hcdata_woke_set(hcdt, mygr, hc, caseRecv, data);
                crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
                pmutex_unlock(&hc->lock);
                return 1;
            }

            hcdt = hcdata_new(mygr);
            hcdt->sdelem = data;
            int rv = szqueue_add(hc->sendq, hcdt);
            assert(rv != -1);

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

        hcdata* hcdt = (hcdata*)szqueue_remove(hc->sendq);
        if (hcdt != nilptr) {
            fiber* gr = hcdt->gr;
            // assert(gr->id == hcdt->grid);
            *pdata = hcdt->sdelem;
            linfo("resume sender %d/%d by %d/%d\n", hcdt->grid, hcdt->mcid, mygr->id, mygr->mcid);
            pmutex_unlock(&hc->lock);
            crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
            return 1;
        }

        // cannot recv directly
        {
            hcdata* hcdt = hcdata_new(mygr);
            hcdt->rvelem = pdata;
            int rv = szqueue_add(hc->recvq, hcdt);
            assert(rv != -1);
            // linfo("chan recv %d\n", mygr->id);
            linfo("yield me recver %d/%d, qc %d ch=%p\n", mygr->id, mygr->mcid, hc->recvq->size, hc);
            mygr->hclock = &hc->lock;
            crn_procer_yield(-1, YIELD_TYPE_CHAN_RECV);
            assert(*pdata != invlidptr);
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

        hcdata* hcdt = (hcdata*)szqueue_remove(hc->sendq);
        fiber* gr = hcdt->gr;
        if (gr != nilptr) {
            // assert(gr->id == hcdt->grid);
            *pdata = hcdt->sdelem;
            pmutex_unlock(&hc->lock);
            crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
            return 1;
        }

        hcdt = hcdata_new(mygr);
        hcdt->rvelem = pdata;
        int rv = szqueue_add(hc->recvq, hcdt);
        assert(rv != -1);
        pmutex_unlock(&hc->lock);
        crn_procer_yield(-1, YIELD_TYPE_CHAN_RECV);
        return 1;
    }
}

// https://ninokop.github.io/2017/11/07/Go-Channel%E7%9A%84%E5%AE%9E%E7%8E%B0/

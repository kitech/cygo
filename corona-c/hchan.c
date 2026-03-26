
#include "chan.h"
#include "hchan.h"
#include "coronapriv.h"

// wrapper chan_t with fiber integeration

crn_hcdata* crn_hcdata_new(fiber* gr) {
    crn_hcdata* d = (crn_hcdata*)crn_gc_malloc(sizeof(crn_hcdata));
    d->gr = gr;
    d->grid = gr->id;
    d->mcid = gr->mcid;
    return d;
}
void crn_hcdata_free(crn_hcdata* d) {
    d->rvelem = nilptr;
    crn_gc_free(d);
}
void crn_hcdata_woke_set(crn_hcdata*d, fiber* wkgr, crn_hchan* hc, int wkcase, void* elem) {
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

static void crn_hchan_finalizer(void* hc) {
    linfo("hchan dtor %p %d\n", hc, gettid());
    // assert(1==2);
}

crn_hchan* crn_hchan_new(int cap) {
    crn_hchan* hc = (crn_hchan*)crn_gc_malloc(sizeof(crn_hchan));
    pmutex_init(&hc->lock, nilptr);
    crn_set_finalizer(hc, crn_hchan_finalizer);
    hc->c = chan_init(cap);  // why 248 cause hc gced too early?
    hc->cap = cap;

    // only support max 32 concurrent fibers on one hchan
    // should enough
    hc->recvq = szqueue_init(256);
    hc->sendq = szqueue_init(256);

    return hc;
}

// TODO non fiber thread should works also
int crn_hchan_close(crn_hchan* hc) {
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
        crn_hcdata* hcdt = (crn_hcdata*)szqueue_remove(hc->recvq);
	if (hcdt == nilptr) break;
	assert(hcdt != nilptr);
        fiber* gr = hcdt->gr;
        if (gr == nilptr) {
            break;
        }
        crn_hcdata_woke_set(hcdt, mygr, hc, caseClose, nilptr);
        crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
        // hcdata_free(hcdt);
    }
    if (hc->recvq != nilptr) szqueue_dispose(hc->recvq);

    qsz = hc->sendq->size;
    if (qsz > 0) { linfo("discard sendq %d\n", qsz); }
    while(hc->sendq != nilptr) {
        crn_hcdata* hcdt = (crn_hcdata*)szqueue_remove(hc->sendq);
	if (hcdt == nilptr) break;
	assert(hcdt != nilptr);
        fiber* gr = hcdt->gr;
        if (gr == nilptr) {
            break;
        }
        crn_hcdata_woke_set(hcdt, mygr, hc, caseClose, nilptr);
        crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
        // hcdata_free(hcdt);
    }
    if (hc->sendq != nilptr) szqueue_dispose(hc->sendq);

    int bufsz = chan_size(hc->c);
    if (bufsz > 0) { lwarn("discard bufsz %d\n", bufsz); }
    chan_dispose(hc->c);
    hc->c = nilptr;
    bzero(hc, sizeof(crn_hchan));
    atomic_setint(&hc->closed, 1);
    pmutex_unlock(&hc->lock);
    crn_gc_free(hc); // gc_free is later
    return true;
}
int crn_hchan_is_closed(crn_hchan* hc) {
    return atomic_getint(&hc->closed);
}
int crn_hchan_cap(crn_hchan* hc) { return hc->cap; }
int crn_hchan_len(crn_hchan* hc) { return chan_size(hc->c); }

// TODO when sending/recving, hchan closed case
// TODO non fiber thread should works also
int crn_hchan_send(crn_hchan* hc, void* data) {
    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    pmutex_lock(&hc->lock);
    if (hc->cap == 0) {
        // if any fiber waiting, put data to it elem and then wakeup
        // else put self to sendq and then parking self

        crn_hcdata* hcdt = (crn_hcdata*)szqueue_remove(hc->recvq);
        if (hcdt != nilptr) {
            fiber* gr = hcdt->gr;
            // assert(gr->id == hcdt->grid);
            crn_hcdata_woke_set(hcdt, mygr, hc, caseRecv, data);
            linfo("resume recver %d/%d by %d/%d\n", hcdt->grid, hcdt->mcid, mygr->id, mygr->mcid);
            pmutex_unlock(&hc->lock);
            crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
            return 1;
        }

        // cannot send directly
        if (hcdt == nilptr) {
            // put data to my hcelem, put self to sendq, then parking self
            crn_hcdata* hcdt = crn_hcdata_new(mygr);
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
            crn_hcdata* hcdt = (crn_hcdata*)szqueue_remove(hc->recvq);
	    if (hcdt != nilptr) {
		fiber* gr = hcdt->gr;
		if (gr != nilptr) {
		    assert(gr->id == hcdt->grid);
		    crn_hcdata_woke_set(hcdt, mygr, hc, caseRecv, data);
		    crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
		}
	    }
            pmutex_unlock(&hc->lock);
            return 1;
        }else{
            // if has recvq, put to peer hcelem, wakeup peer and return
            // put data to my hcelem, put self to sendq, then parking self
            crn_hcdata* hcdt = (crn_hcdata*)szqueue_remove(hc->recvq);
            fiber* gr = hcdt->gr;
            if (gr != nilptr) {
                // assert(gr->id == hcdt->grid);
                crn_hcdata_woke_set(hcdt, mygr, hc, caseRecv, data);
                crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
                pmutex_unlock(&hc->lock);
                return 1;
            }

            hcdt = crn_hcdata_new(mygr);
            hcdt->sdelem = data;
            int rv = szqueue_add(hc->sendq, hcdt);
            assert(rv != -1);

            pmutex_unlock(&hc->lock);
            crn_procer_yield(-1, YIELD_TYPE_CHAN_SEND);
            return 1;
        }
    }
}

int crn_hchan_recv(crn_hchan* hc, void** pdata) {
    fiber* mygr = crn_fiber_getcur();
    assert(mygr != nilptr);

    pmutex_lock(&hc->lock);
    if (hc->cap == 0) {
        // if have elem not nil, get it
        // else if any sendq, wakeup them,
        // else parking

        crn_hcdata* hcdt = (crn_hcdata*)szqueue_remove(hc->sendq);
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
            crn_hcdata* hcdt = crn_hcdata_new(mygr);
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

        crn_hcdata* hcdt = (crn_hcdata*)szqueue_remove(hc->sendq);
        fiber* gr = hcdt->gr;
        if (gr != nilptr) {
            // assert(gr->id == hcdt->grid);
            *pdata = hcdt->sdelem;
            pmutex_unlock(&hc->lock);
            crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
            return 1;
        }

        hcdt = crn_hcdata_new(mygr);
        hcdt->rvelem = pdata;
        int rv = szqueue_add(hc->recvq, hcdt);
        assert(rv != -1);
        pmutex_unlock(&hc->lock);
        crn_procer_yield(-1, YIELD_TYPE_CHAN_RECV);
        return 1;
    }
}

// https://ninokop.github.io/2017/11/07/Go-Channel%E7%9A%84%E5%AE%9E%E7%8E%B0/

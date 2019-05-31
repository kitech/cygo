
#include "noropriv.h"
#include "hchan.h"

enum {
      caseNil = 0,
      caseRecv,
      caseSend,
      caseDefault,
};

typedef struct scase {
    hchan* hc;
    void* hcelem;
    uint16_t kind;
    uintptr_t pc;
    int64_t reltime;
} scase;

static
void sellock(scase** cas0, uint16_t* lockorder, int ncases) {
    for (int i = 0; i < ncases; i++) {
        scase* cas = cas0[lockorder[i]];
        mtx_lock(&cas->hc->lock);
    }
}
static
void selunlock(scase** cas0, uint16_t* lockorder, int ncases) {
    for (int i = ncases-1; i >= 0; i--) {
        scase* cas = cas0[lockorder[i]];
        mtx_unlock(&cas->hc->lock);
    }
}
static
bool selparkcommit() {
    return false;
}
static
void selblock() {
}

static
bool selectgo(int* rcasi, scase** cas0, uint16_t* order0, int ncases) {
    goroutine* mygr = noro_goroutine_getcur();
    sellock(cas0, order0, ncases);

    hchan* hc;
    scase* sk;
    goroutine* gr;

    int dfti;
    scase* dftv;
    int casi;
    scase* cas;
    bool recvok;

 loop:
    // pass 1 - look for something already waiting
    for (int i = 0; i < ncases; i ++) {
        casi = order0[i];
        cas = cas0[casi];
        hc = cas->hc;

        switch (cas->kind) {
        case caseNil:
            assert(1==2); break;
        case caseRecv:
            gr = queue_remove(hc->sendq);
            if (gr != nilptr) goto recv;
            if (hchan_len(hc)>0) goto bufrecv;
            if (hchan_is_closed(hc)) goto rclose;
            break;

        case caseSend:
            if (hchan_is_closed(hc)) goto sclose;
            gr = queue_remove(hc->recvq);
            if (gr != nilptr) goto send;
            if (hchan_len(hc) < hchan_cap(hc)) goto bufsend;
            break;
        case caseDefault:
            dfti = casi;
            dftv = cas;
            break;
        default:
            assert(1==2); break;
        }
    }

    if (dftv != nilptr) {
        selunlock(cas0, order0, ncases);
        casi = dfti;
        cas = dftv;
        goto retc;
    }

    // pass 2 - enqueue on all chans
    for (int i = 0; i < ncases; i ++) {
        casi = i;
        cas = cas0[casi];
        if (cas->kind == caseNil) continue;

        hc = cas->hc;

        switch (cas->kind) {
        case caseRecv:
            queue_add(hc->recvq, mygr);
            break;
        case caseSend:
            queue_add(hc->sendq, mygr);
            break;
        default:
            assert(1==2); break;
        }
    }

    // wait for someone to wake us up
    selunlock(cas0, order0, ncases);
    noro_processor_yield(-1, YIELD_TYPE_CHAN_SELECT);

    sellock(cas0, order0, ncases);

    // pass 3  - dequeue from unsuccessful chans
    casi = -1;
    cas = nilptr;

    for (int i = 0; i < ncases; i ++) {
        sk = cas0[i];
        if (sk->kind == caseNil) continue;

        if (sk == mygr->wokeby) {
            casi = i;
            cas = sk;
        }else{
            hc = sk->hc;
            if (sk->kind == caseSend) {
                gr = queue_remove(hc->sendq);
            }else{
                gr = queue_remove(hc->recvq);
            }
            assert(gr == mygr);
            gr = nilptr;
        }
    }

    if (cas == nilptr) {
        goto loop;
    }

    hc = cas->hc;
    linfo("wait-return: cas0=%p hc=%p cas=%p kind=%d\n", cas0, hc, cas, cas->kind);

    if (cas->kind == caseRecv) {
        recvok = true;
    }

    selunlock(cas0, order0, ncases);
    goto retc;

 bufrecv:
 bufsend:
 recv:
 send:
 rclose:
 retc:
    rcasi = casi;
    return recvok;
 sclose:
    return false;
}

bool goselect(int* rcasi, scase** cas0, int ncases) {
    if (ncases == 0) {
        // parking forever
    }

    assert(ncases <= 32);
    uint16_t order0[32] = {0};
    for (int i = 0; i < ncases; i ++) {
        order0[i] = i;
    }
    return selectgo(rcasi, cas0, order0, ncases);
}


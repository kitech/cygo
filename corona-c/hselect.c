
#include "coronapriv.h"
#include "hchan.h"

typedef struct crn_scase {
    crn_hchan* hc;
    uint16_t kind;
    void* hcelem;
    crn_hcdata* hcdt;
    uintptr_t pc;
    int64_t reltime;
} crn_scase;

// TODO some cases hcdata is not need
crn_scase* crn_scase_new(crn_hchan* hc, uint16_t kind, void* elem) {
    crn_scase* cas = (crn_scase*)crn_gc_malloc(sizeof(crn_scase));
    fiber* mygr = crn_fiber_getcur();
    cas->hc = hc; // was never set: crn_sellock() locked a NULL hc afterwards
    cas->kind = kind; // was never set either: every case looked like caseNil
    crn_hcdata* hcdt= crn_hcdata_new(mygr);
    if (kind == caseRecv) {
        hcdt->rvelem = &cas->hcelem;
    }else if(kind == caseSend){
        cas->hcelem = elem;
        hcdt->sdelem = elem;
    }else{
    }
    cas->hcdt = hcdt;
    return cas;
}
void crn_scase_free(crn_scase* cas) {
    crn_hcdata_free(cas->hcdt);
    crn_gc_free(cas);
}

static
void crn_sellock(crn_scase** cas0, uint16_t* lockorder, int ncases) {
    for (int i = 0; i < ncases; i++) {
        crn_scase* cas = cas0[lockorder[i]];
        if (cas->hc == nilptr) continue; // caseDefault/nil case: nothing to lock
        int dup = 0;
        for (int j = 0; j < i; j++) { // dedup: same hc appears once in a select
            if (cas0[lockorder[j]]->hc == cas->hc) { dup = 1; break; }
        }
        if (dup) continue;
        pmutex_lock(&cas->hc->lock);
    }
}
static
void crn_selunlock(crn_scase** cas0, uint16_t* lockorder, int ncases) {
    for (int i = ncases-1; i >= 0; i--) {
        crn_scase* cas = cas0[lockorder[i]];
        if (cas->hc == nilptr) continue; // caseDefault/nil case: nothing locked
        int dup = 0;
        for (int j = i - 1; j >= 0; j--) { // unlock only the first occurrence
            if (cas0[lockorder[j]]->hc == cas->hc) { dup = 1; break; }
        }
        if (dup) continue;
        pmutex_unlock(&cas->hc->lock);
    }
}
static
bool crn_selparkcommit() {
    return false;
}
static
void crn_selblock() {
}

static
bool crn_selectgo(int* rcasi, crn_scase** cas0, uint16_t* order0, int ncases) {
    fiber* mygr = crn_fiber_getcur();
    crn_sellock(cas0, order0, ncases);
    linfo("rcasi=%d cas0=%p order0=%p ncases=%d\n", *rcasi, cas0, order0, ncases);
    crn_hchan* hc = nilptr;
    crn_scase* sk = nilptr;
    crn_hcdata* hcdt = nilptr;
    fiber* gr = nilptr;
    fiber* wkgr = nilptr;
    int casewk = 0;
    crn_hchan* wkhc = nilptr;

    int dfti = 0;
    crn_scase* dftv = nilptr;
    int casi = 0;
    crn_scase* cas = nilptr;
    bool recvok = false;
    int retline = 0;

    // TODO order and dedup

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
            hcdt = szqueue_remove(hc->sendq);
            if (hcdt == nilptr) break; // empty queue: no sender waiting
            gr = hcdt->gr;
            if (gr != nilptr) { assert(gr->id == hcdt->grid); }
            if (gr != nilptr) goto recv;
            if (crn_hchan_len(hc)>0) goto bufrecv;
            if (crn_hchan_is_closed(hc)) goto rclose;
            break;

        case caseSend:
            if (crn_hchan_is_closed(hc)) goto sclose;
            hcdt = szqueue_remove(hc->recvq);
            if (hcdt == nilptr) break; // empty queue: no receiver waiting
            gr = hcdt->gr;
            if (gr != nilptr) { assert(gr->id == hcdt->grid); }
            if (gr != nilptr) goto send;
            if (crn_hchan_len(hc) < crn_hchan_cap(hc)) goto bufsend;
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
        crn_selunlock(cas0, order0, ncases);
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

        int addrv = -1;
        switch (cas->kind) {
        case caseRecv:
            hcdt = cas->hcdt;
            addrv = szqueue_add(hc->recvq, hcdt);
            hcdt = nilptr;
            break;
        case caseSend:
            hcdt = cas->hcdt;
            addrv = szqueue_add(hc->sendq, hcdt);
            hcdt = nilptr;
            break;
        default:
            assert(1==2); break;
        }

        if (addrv != 0) {
            /*
             * 队列满（szqueue 上限 256）。旧代码忽略返回值：这个 case 的
             * hcdt 没有进入任何队列，fiber 却照样 yield，之后没有任何人
             * 能唤醒它（永久 parked，destroy 永远不会被调用）。
             * 这里回滚已经入队的 case，解锁后返回失败，避免静默挂死。
             */
            for (int j = i - 1; j >= 0; j--) {
                crn_scase* rj = cas0[j];
                if (rj->kind == caseNil || rj->kind == caseDefault) continue;
                if (rj->kind == caseSend) {
                    szqueue_remove_value(rj->hc->sendq, rj->hcdt);
                } else if (rj->kind == caseRecv) {
                    szqueue_remove_value(rj->hc->recvq, rj->hcdt);
                }
            }
            crn_selunlock(cas0, order0, ncases);
            lerror("select enqueue full hc=%p grid=%d ncases=%d\n",
                   hc, mygr->id, ncases);
            *rcasi = -1;
            return false;
        }
    }

    // wait for someone to wake us up
    crn_selunlock(cas0, order0, ncases);
    linfo("should here %d\n", 0);
    crn_procer_yield(-1, YIELD_TYPE_CHAN_SELECT);
    linfo("should here %d\n", 0);
    crn_sellock(cas0, order0, ncases);

    // pass 3  - dequeue from unsuccessful chans
    casi = -1;
    cas = nilptr;

    /* wkgr = mygr->wokeby; */
    /* wkhc = mygr->wokehc; */
    /* casewk = mygr->wokecase; */
    for (int i = 0; i < ncases; i ++) {
        sk = cas0[i];
        if (sk->kind == caseNil || sk->kind == caseDefault) continue;

        // cas was set to nilptr before this loop; use this case's own hcdt
        wkgr = sk->hcdt->wokeby;
        wkhc = sk->hcdt->wokehc;
        casewk = sk->hcdt->wokecase;

        // try match which case woke
        // caseClose wakes parked senders AND receivers: match it so the
        // woken fiber still runs the dequeue pass for its other cases,
        // instead of goto loop and leaving its hcdt stranded in queues
        if ((casewk == sk->kind || casewk == caseClose) && sk->hc == wkhc) {
            casi = i;
            cas = sk;
            linfo("case woke i=%d direction=%d by=%p val=%p\n", i, casewk, wkgr, sk->hcelem);
        }
        else{
            /*
             * 摘除本 case 自己的 hcdt，而不是盲取队头。
             * 旧实现按队头 szqueue_remove：如果队头是别的等待者的 hcdt，
             * 会被取走且不放回，那个 fiber 就永远等不到唤醒（永久 parked，
             * destroy 永远不会被调用）。hcdt 也可能已被 waker/close 提前
             * 取走，此时 remove_value 返回 -1，属于正常情况。
             */
            hc = sk->hc;
            if (sk->kind == caseSend) {
                szqueue_remove_value(hc->sendq, sk->hcdt);
            }else{
                szqueue_remove_value(hc->recvq, sk->hcdt);
            }
            continue;
        }
    }

    if (cas == nilptr) {
        goto loop;
    }

    hc = cas->hc;
    linfo("wait-return: cas0=%p hc=%p cas=%p kind=%d\n", cas0, hc, cas, cas->kind);

    // woken by close: recv reports EOF (false); send on closed chan is
    // invalid (Go panics, we report false). selunlock happens in the labels.
    if (casewk == caseClose) {
        if (cas->kind == caseRecv) { goto rclose; }
        goto sclose;
    }

    if (cas->kind == caseRecv) {
        recvok = true;
    }

    crn_selunlock(cas0, order0, ncases);
    retline = __LINE__;
    goto retc;

 bufrecv:
    recvok = true;
    chan_recv(hc->c, &cas->hcelem);
    crn_selunlock(cas0, order0, ncases);
    retline = __LINE__;
    goto retc;

 bufsend:
    chan_send(hc->c, cas->hcelem);
    crn_selunlock(cas0, order0, ncases);
    retline = __LINE__;
    goto retc;

 recv:
    crn_selunlock(cas0, order0, ncases);
    crn_procer_resume_one(gr, 0, hcdt->grid, hcdt->mcid);
    linfo("syncrecv: cas0=%p hc=%p val=%p\n", cas0, hc, cas->hcelem);
    recvok = true;
    retline = __LINE__;
    goto retc;

 rclose:
    crn_selunlock(cas0, order0, ncases);
    recvok = false;
    retline = __LINE__;
    goto retc;

 send:
    crn_selunlock(cas0, order0, ncases);
    linfo("syncsend: cas0=%p hc=%p val=%p\n", cas0, hc, cas->hcelem);
    retline = __LINE__;
    goto retc;

 retc:
    linfo("return casi=%d recvok=%d retline=%d\n", casi, recvok, retline);
    *rcasi = casi;
    return recvok;

 sclose:
    linfo("send closed chan %d", 0);
    // reachable both from pass1 (locks held) and close-wake: unlock first
    crn_selunlock(cas0, order0, ncases);
    return false;
}

static void crn_blocknocase() {
    crn_procer_yield(-1, YIELD_TYPE_CHAN_SELECT_NOCASE);
}

bool crn_goselect(int* rcasi, crn_scase** cas0, int ncases) {
    if (ncases == 0) {
        // parking forever
        crn_blocknocase();
        assert(1==2); // not reachable
    }

    assert(ncases <= 32);
    uint16_t order0[32] = {0};
    for (int i = 0; i < ncases; i ++) {
        order0[i] = i;
    }
    return crn_selectgo(rcasi, cas0, order0, ncases);
}

// go 1.12.5

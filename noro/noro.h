
#ifndef _NORO_H_
#define _NORO_H_


typedef struct noro noro;

noro* noro_get();
noro* noro_new();
void noro_init(noro* nr);
void noro_free(noro* lnr);
void noro_wait_init_done(noro* nr);

void noro_post(void(*fn)(void*arg), void*arg);

#endif

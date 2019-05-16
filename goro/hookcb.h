#ifndef _HOOK_CB_H_
#define _HOOK_CB_H_

#include <stdbool.h>

typedef struct fdcontext fdcontext;
typedef struct hookcb hookcb;

int fdcontext_set_nonblocking(fdcontext*fdctx, bool isNonBlocking) ;
bool fdcontext_is_socket(fdcontext*fdctx);
bool fdcontext_is_tcpsocket(fdcontext*fdctx);
bool fdcontext_is_nonblocking(fdcontext*fdctx);

hookcb* hookcb_get();
void hookcb_oncreate(int fd, int fdty, bool isNonBlocking, int domain, int sockty, int protocol) ;
void hookcb_onclose(int fd) ;
void hookcb_ondup(int from, int to) ;
fdcontext* hookcb_get_fdcontext(int fd);

#endif


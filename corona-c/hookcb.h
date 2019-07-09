#ifndef _HOOK_CB_H_
#define _HOOK_CB_H_

#include <stdbool.h>

#define FDISSOCKET  1
#define FDISPIPE  2
#define FDISFILE  3

typedef struct fdcontext fdcontext;
typedef struct hookcb hookcb;

int fdcontext_set_nonblocking(fdcontext*fdctx, bool isNonBlocking) ;
bool fdcontext_is_socket(fdcontext*fdctx);
bool fdcontext_is_tcpsocket(fdcontext*fdctx);
bool fdcontext_is_nonblocking(fdcontext*fdctx);
bool fd_is_nonblocking(int fd);
int hookcb_fd_set_nonblocking(int fd, bool isNonBlocking);

hookcb* hookcb_get();
void hookcb_oncreate(int fd, int fdty, bool isNonBlocking, int domain, int sockty, int protocol) ;
void hookcb_onclose(int fd) ;
void hookcb_ondup(int from, int to) ;
fdcontext* hookcb_get_fdcontext(int fd);

#endif


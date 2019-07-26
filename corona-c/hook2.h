#ifndef HOOK2_H
#define HOOK2_H

#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>

/*
typedef int (*pcond_signal_t)(pthread_cond_t *cond);
extern pcond_signal_t pcond_signal_f;
*/

typedef int (*pthread_create_t)(pthread_t *thread, const pthread_attr_t *attr,
                              void *(*start_routine) (void *), void *arg);
extern pthread_create_t pthread_create_f;

typedef int (*getaddrinfo_t)(const char *node, const char *service,
                const struct addrinfo *hints,
                struct addrinfo **res);
extern getaddrinfo_t getaddrinfo_f;


#endif /* HOOK2_H */

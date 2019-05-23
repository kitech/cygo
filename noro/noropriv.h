#ifndef _NORO_PRIV_H_
#define _NORO_PRIV_H_

// std
#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <errno.h>

// sys
#include <pthread.h>
#include <sys/epoll.h>
#include <sys/timerfd.h>

// third
#include <collectc/hashtable.h>
#include <collectc/array.h>

// project
#include "yieldtypes.h"
#include "hookcb.h"
#include "noro_util.h"
#include "norogc.h"


// for netpoller.c
typedef struct netpoller netpoller;
netpoller* netpoller_new();
void netpoller_loop();
void netpoller_yieldfd(int fd, int ytype, void* gr);

#endif


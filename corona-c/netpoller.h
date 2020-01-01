#ifndef NETPOLLER_H
#define NETPOLLER_H

#define CXEV_IO (0x1<<5)
#define CXEV_TIMER (0x1<<6)
#define CXEV_DNS_RESOLV (0x1<<7)

static const int CRN_SEC = 1;
static const int CRN_MSEC = 1000;
static const int CRN_USEC = 1000000;
static const int CRN_NSEC = 1000000000;

#endif /* NETPOLLER_H */


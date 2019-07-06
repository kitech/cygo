A library that implemention golang's core features,
goroutines, schedulers, channels, goselect and garbage collect.

### Features

* [x] stackful coroutine
* [x] multiple threads
* [x] channels
* [x] golang select semantic
* [x] garbage collect
* [x] syscall hook
* [ ] explict syscall functions wrapper

### Usage

All exported functions is `crnpub.h`

### Todos

[x] channel select semantic
[x] wait reason
[ ] goroutines stats, count, memory
[ ] goroutines stack info
[ ] scheduler switch to goroutine
[ ] native main function switch to goroutine
[ ] improve send/recv bool flag
[ ] mutex lock/unlock yield?
[ ] dynamic increase/decrease processor(P)
[x] sockfd timeout support

### Difference with Go
* Ours gosched is a sleep, Go's Gosched is long parking and wait resched
* Go have Sudog, we haven't

### Thirdpartys

Thanks all the contributors.

* libcoro 
* libchan https://github.com/tylertreat/chan
* libcollectc
* rixlog
* libgc >= 8.2.0
* libevent >= 2.1

### BUGS

[x] SIGABRT: coro\_init () at /noro/coro.c:104
    when resume a finished fiber, this crash occurs
[ ] SIGSEGV: bdwgc: mark.c:1581 GC\_push\_all\_eager
    https://www.mail-archive.com/ecls-list@lists.sourceforge.net/msg00161.html
[ ] ASSERT: bdwgc: misc.c: 1986 GC\_disable
[ ] gopackany:goroutine\_post, seems many times just after this: Assertion failure: extra/../misc.c:1986
[ ] ==17317==  Access not within mapped region at address 0x0
    ==17317==    at 0x168C14: kind\_j8p3h8iojolY0RbF1nkaZgxmltree (stdlib\_xmltree.nim.c:310)
[ ] hang forever on ppoll ()
[ ] sometimes GC will stop work
[x] hang forever on __lll_lock_wait_private () from /usr/lib/libc.so.6
    occurs when call linfo/log write in push_other_roots callback
    says that linfo/log has some where not safe point
    related with signal handler
[ ] GC_clear_fl_marks infinite loop

### Note
* A program entering an infinite recursion or running out of space in the stack memory is known as a stack overflow

GC_NPROCS=1 ./prog to set gc thread count


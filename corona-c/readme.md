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

[ ] SIGABRT: coro\_init () at /noro/coro.c:104
[ ] SIGSEGV: bdwgc: mark.c:1581 GC\_push\_all\_eager
[ ] ASSERT: bdwgc: misc.c: 1986 GC\_disable
[ ] gopackany:goroutine\_post, seems many times just after this: Assertion failure: extra/../misc.c:1986
[ ] ==17317==  Access not within mapped region at address 0x0
    ==17317==    at 0x168C14: kind\_j8p3h8iojolY0RbF1nkaZgxmltree (stdlib\_xmltree.nim.c:310)
[ ] hang forever on ppoll ()

### Note
* A program entering an infinite recursion or running out of space in the stack memory is known as a stack overflow

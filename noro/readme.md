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

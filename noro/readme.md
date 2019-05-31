A library that implemention golang's core features,
goroutines, schedulers, channels, goselect and garbage collect.

### Features

* [x] stackful coroutine
* [x] multiple threads
* [x] channels
* [ ] golang select semantic
* [x] garbage collect
* [x] syscall hook
* [ ] explict syscall functions wrapper

### Todos

[ ] channel select semantic
[ ] wait reason
[ ] goroutines stats, count, memory
[ ] goroutines stack info


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

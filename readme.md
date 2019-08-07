Go compiler to C, with a generic library contains Go core features, like goroutine,channel,GC.

That's will generate minimal binary. The farther plan is compile any Go package to C.

### Features
* goroutine
* channel
* GC
* CGO
* interface

### Supported important syntax
* defer
* closure
* select

### Todos
* [ ] dynamic stack resize
* [ ] correct and more safe point for GC

### Supported original Go packages
* unsafe
* errors

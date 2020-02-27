Compile Go to C, with a generic library contains Go core features, like goroutine,channel,GC.

That's will generate minimal binary. The farther plan is compile any Go package to C.

### The pain of Go
* Too large binary size
* Not friendly with C
* Builtin string/array/map no methods
* Too verbosity error handling, not like the Go2 `try` error handling proposal

### Features
* goroutine
* channel
* defer
* GC
* CGO
* interface
* closure
* string/array/map with lot builtin methods
* `catch` statement error handling

### Install

```
cd $GOPATH
git clone https://github.com/kitech/cygo
cd cygo/bysrc
go build -o cygo
```

### Example

```
./cygo ./tpkgs/hello
cmake .
make
```

### Supported important syntax
* defer
* closure
* select

### Todos
* [ ] dynamic stack resize
* [ ] correct and more safe point for GC
* [ ] support more OS/platforms
* [ ] so much to do

### Supported original Go packages
* unsafe
* errors

### 资料
* minigo
* tinygo
* Let's Build A Simple Interpreter  https://github.com/rspivak/lsbasi
* dwarf https://github.com/gimli-rs/gimli


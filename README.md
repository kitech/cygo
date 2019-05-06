# go2ll: A toy go compiler to LLVM

## What is this?

`go2ll` translates Go's
[x/tools/go/ssa](https://godoc.org/golang.org/x/tools/go/ssa) representation
into LLVM. It's written in pure Go (no linking to LLVM), and emits [LLVM
assembly](http://llvm.org/docs/LangRef.html) as output.

To implement various runtime functionality, it uses a few libc functions (such
as `malloc` for memory allocation), so that it's easy to compile the resulting
bit-code to an ordinary executable you can run with
[`llc`](http://llvm.org/docs/CommandGuide/llc.html).

This is the basis of a 30-minute [live-coding
session](https://github.com/pwaller/go2ll-talk) showing how to write a program
from scratch which translates a simple Go program in this manner. This
live-coding session was presented at Go-Sheffield on 7th March 2019.

## Getting started

This is an early release, I have not yet prepared instructions.

Please feel free to poke around. Don't expect anything to work unless you craft
it to do so.

## Other similar projects

`go2ll` is a toy project, and as such is only useful for small programs in limited
circumstances (at the moment, itch scratching). You probably want one of these
other projects which have had more manpower invested:

* [tinygo](https://github.com/tinygo-org/tinygo/) is an impressive and more
  complete job at doing a similar thing as `go2ll`. It is also written in Go and
  directly links to LLVM to invoke the optimizers. As of when I last checked, it
  had some similar limitations as `go2ll`.
* [llgo](https://github.com/llvm/llvm-project/tree/a2e23f682afb040755f93b824c0edf317115d1eb/llgo), now maintained as part of the LLVM project, but as far as I can tell hasn't had much love in a number of years.
* [gollvm](https://go.googlesource.com/gollvm/) is a relatively recent (started May 2017) effort to build a Go compiler using LLVM, by some Googlers. As I heard somewhere it started out as a 20% time project. I don't know much about it. It's written in C++, links directly to LLVM, and seems to be maintained recently (as of May 2019).

I'm sure there are other similar efforts out there, please file an issue if you
think I should add to this list, or modify the description of any of the above
projects I will!

## What works?

So far, `strconv.ParseFloat` works, as does computing a SHA1. Both run faster
than equivalent code when compiled with the standard Go compiler.

I have only made a cursory check that my code hasn't been fully optimized away,
it's possible that this result is incorrect or not useful.

## Limitations (non-exhaustive list):

* No garbage collection, so no long lived programs unless you avoid allocation.
* No goroutines.
* Can't yet use much of the standard library.
* Interfaces don't yet work.
* Closures don't yet work.

I would love to partially lift some of the above limitations - in particular
enable I/O for some other interesting benchmarks. I'm uncertain if I will get to
this.

Benefits:

* Pure go
  * No need to link to LLVM (but you may need the LLVM executables around).
  * Sub-second compile times for compiling-the-compiler.

TODO:

* Complete limitations list
* Describe some more benefits
* Example invocations

## Aspirations / Experiment ideas

* Use [Boehm](https://www.hboehm.info/gc/) for GC. Perhaps this gives at least a
  partially functioning GC with low implementation cost. Remains to be seen.
* Teach LLVM about Go's calling convention, so that fragments of code which are
  faster using LLVM as a compiler can be included in ordinary Go programs
  without requiring CGo.
* Enable syscalls to work, particularly to make I/O work well.

## Why would you do this?

I had a program whose running time was bottlenecked on parsing floating point
numbers in a CSV. The question arose in my mind "What if I used C to parse these
floats?". So I wrote a micro-benchmark (I know, micro-benchmarking has
limitations) parsing floats. I discovered that C was slower than Go(!).

This came as a surprise, because I believe C's optimizers have been in existence
longer and make a greater tradeoff towards execution speed at the expense of
compilation time. That Go both faster to compile and faster to execute than C
is, well, interesting.

Part of the speed difference can be explained with the fact floating point
parsing is implemented differently in Go than in C. So I wondered - what if I
used a powerful optimizer, such as LLVM's easy-to-use tooling, with Go's
floating point implementation?

I didn't want to sit around and re-implement Go's floating point parsing in C,
that would be too dull. Especially since I had another idea at my fingertips.

In a previous job I worked on a Go-To-Verilog compiler which used LLVM IR as an
intermediate representation. With permission, from scratch, I recreated and
extended the frontend for fun.

## Results

So, what came of the float parser?



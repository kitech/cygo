
type
    chan*[T] = ref object
        hc: pointer
        val: T
        dir: int
        born: float

# for each object of T, this will called
proc hchan_finalizer[T](x : T) =
    var hc = x.hc
    var dtime = epochTime() - x.born
    x.hc = nil
    var br = hchan_close(hc)
    linfo("chan GCed", hc, dtime, br)
    return

## public chan API
proc makechan*(T: typedesc, cap:int) : chan[T] =
    var c : chan[T]
    c.new(hchan_finalizer)
    c.hc = hchan_new(cap)
    var val : T
    c.val = val
    c.born = epochTime()
    return c

proc send*[T](c: chan[T], v : T) : bool {.discardable.} =
    c.val = v  # ref it, but when unref?
    return hchan_send(c.hc, cast[pointer](v))

proc recv*[T](c: chan[T]) : T {.discardable.} =
    var ret : T
    var dat : pointer
    var rv = hchan_recv(c.hc, dat.addr)
    ret = cast[T](dat)
    return ret

proc cap*[T](c: chan[T]) : int = hchan_cap(c.hc)
proc len*[T](c: chan[T]) : int = hchan_len(c.hc)
proc closed*[T](c: chan[T]) : bool = hchan_is_closed(c.hc)
proc `$`*[T](c : chan[T]) : string =
    return "chan[$#; $#]@$#" % [T.name, $(c.cap()), c.hc.repr]

type noimplerr = ref CatchableError

# alias of send
# c <- v
proc `<-`*[T](c: chan[T], v: T) =
    c.send(v)
    return

# alias of recv
# var v = <- c
proc `<-`*[T](c : chan[T]) : T {.discardable.} =
    return c.recv()

{.push hint[XDeclaredButNotUsed]: off.}


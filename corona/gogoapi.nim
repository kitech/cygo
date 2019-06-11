# implemention public user callable api

import macros
import typeinfo

include "ffi.nim"

type
    # mirror of C TGenericSeq
    PNSeq = ptr NSeq
    NSeq = object
        len: int
        reserved: int
        data: pointer

# proc toaddr(v: proc) : pointer = cast[pointer](v)
# proc ntocstr(v: string) : cstring {.exportc.} = return v
# proc ctonstr(v: cstring) : string {.exportc.} = $v
proc todptr[T](v: seq[T]) : pointer = ((cast[PNSeq](v)).data).addr
proc todptr(v: string) : pointer = ((cast[PNSeq](v)).data).addr

proc setupForeignThreadGc2() =
    when not defined(setupForeignThreadGc): discard
    else: setupForeignThreadGc()

proc gogorunnerenter(arg:pointer) =
    setupForeignThreadGc2()
    # var thrh = pthread_self()
    # var sbp = thrmapp[][thrh]
    # var stksize : uint32
    # var stkbase : pointer #libgo_currtask_stack(stksize.addr)
    # var stkbottom = cast[pointer](cast[uint64](stkbase) + stksize.uint64)
    # if stkbase != nil: # still nolucky let nim GC work
    #     sbp.sb1.membase = stkbottom
    #     sbp.sb1.gchandle = sbp.sb0.gchandle
    #     GC_call_with_alloc_lock(gcsetbottom1.toaddr, sbp.sb1.addr)
    # else: linfo("wtf nil stkbase")
    discard

proc gogorunnerleave(arg:pointer) =
    # GC_call_with_alloc_lock(gcsetbottom0.toaddr, sbp.sb0.addr)
    discard

proc goroutine_post(fnptr: pointer; args: pointer) =
    noro_post(fnptr, args)
    return

# begin gogo2
type
    # mirror of typeinfo.Any
    SysAny = object
        value: pointer
        rawTypePtr: pointer

# nim typeinfo.AnyKind to ffi_type*
proc nak2ffipty(ak: AnyKind) : pffi_type =
    if ak == akInt:
        return ffi_type_sint64.addr
    elif ak == akString:
        return ffi_type_pointer.addr
    elif ak == akCString:
        return ffi_type_pointer.addr
    elif ak == akPointer:
        return ffi_type_pointer.addr
    elif ak == akRef:
        return ffi_type_pointer.addr
    elif ak == akPtr:
        return ffi_type_pointer.addr
    elif ak == akSequence:
        return ffi_type_pointer.addr
    else: echo "unknown", ak
    return nil

proc gogorunner_cleanup(arg :pointer) =
    linfo "gogorunner_cleanup", repr(arg)
    var argc = cast[int](pointer_array_get(arg, 1))
    for idx in 0..argc-1:
        let tyidx = 2 + idx*2
        let validx = tyidx + 1
        let akty = cast[AnyKind](pointer_array_get(arg, tyidx.cint))
        let akval = pointer_array_get(arg, validx.cint)
        if akty == akString:
            deallocShared(akval)
            discard
        elif akty == akCString:
            deallocShared(akval)
            discard
        elif akty == akInt:
            deallocShared(akval)
            discard
        elif akty == akPointer: discard
        else: linfo "unknown", akty
    #deallocShared(arg)
    pointer_array_free(arg)
    return

import tables

# pack struct, seq[pointer], which, [0]=fnptr, 1=argc, 2=a0ty, 3=a0val, 4=a1ty, 5=a1val ...
proc gogorunner(arg : pointer) =
    setupForeignThreadGc2()
    gogorunnerenter(arg)

    linfo "gogorunner", arg
    var fnptr = pointer_array_get(arg, 0)
    var argc = cast[int](pointer_array_get(arg, 1))
    assert(fnptr != nil)
    assert(argc >= 0, $argc)

    var atypes = newseq[pffi_type](argc+1)
    var avalues = newseq[pointer](argc+1)
    var ptrtmps = newseq[pointer](argc+1)
    for idx in 0..argc-1:
        let tyidx = 2 + idx*2
        let validx = tyidx + 1
        let akty = cast[AnyKind](pointer_array_get(arg, tyidx.cint))
        var akval = pointer_array_get(arg, validx.cint)
        atypes[idx] = nak2ffipty(akty)
        if akty == akString:
            var cs = cast[cstring](akval)
            var ns = $cs
            ptrtmps[idx] = cast[pointer](ns)
            avalues[idx] = ptrtmps[idx].addr
        elif akty == akCString:
            var cs = cast[cstring](akval)
            ptrtmps[idx] = akval
            avalues[idx] = ptrtmps[idx].addr
        elif akty == akInt:
            ptrtmps[idx] = akval
            avalues[idx] = ptrtmps[idx]
        elif akty == akPointer:
            ptrtmps[idx] = akval
            avalues[idx] = ptrtmps[idx].addr
        else: linfo "unknown", akty, akval
        discard

    var cif : ffi_cif
    var rvalue : uint64
    # dump_pointer_array(argc.cint, atypes.todptr())
    var ret = ffi_prep_cif(cif.addr, FFI_DEFAULT_ABI, argc.cuint, ffi_type_pointer.addr, atypes.todptr)

    # dump_pointer_array(argc.cint, avalues.todptr())
    ffi_call(cif.addr, fnptr, rvalue.addr, avalues.todptr)
    gogorunner_cleanup(arg)
    gogorunnerleave(arg)
    return

# packed to passby format
proc gopackany*(fn:proc, args:varargs[Any, toany]) =
    var ecnt = (2+args.len()*2+2)
    var pargs = pointer_array_new(ecnt.cint)
    pointer_array_set(pargs, 0, cast[pointer](fn))
    pointer_array_set(pargs, 1, cast[pointer](args.len()))

    for idx in 0..args.len-1:
        var arg = args[idx]
        var tyidx = 2+idx*2
        var validx = tyidx+1
        pointer_array_set(pargs, tyidx.cint, cast[pointer](arg.kind))
        var sarg = cast[SysAny](arg)
        if arg.kind == akInt:
            var v = allocShared0(sizeof(int))
            copyMem(v, sarg.value, sizeof(int))
            pointer_array_set(pargs, validx.cint, v)
        elif arg.kind == akString:
            var ns = arg.getString()
            var cs : cstring = $ns
            var v = allocShared0(ns.len()+1)
            copyMem(v, cs, ns.len())
            pointer_array_set(pargs, validx.cint, v)
        elif arg.kind == akCString:
            var cs = arg.getCString()
            var v = allocShared0(cs.len()+1)
            copyMem(v, cs, cs.len())
            pointer_array_set(pargs, validx.cint, v)
        elif arg.kind == akPointer:
            var v = arg.getPointer()
            pointer_array_set(pargs, validx.cint, v)
        else: linfo "unknown", arg.kind

    linfo "copy margs", 2+args.len*2, pargs # why refc=1
    goroutine_post(gogorunner.toaddr(), pargs)
    return

# just like a spawn: gogo2 somefunc(a0, a1, a2)
macro gogo2*(stmt:untyped) : untyped =
    var nstmt = newStmtList()
    for idx, s in stmt:
        if idx == 0: continue
        #linfo "aaa ", repr(s), " ", s.kind
        #logecho2 "aaa ", repr(s), " ", s.kind
        nstmt.add(newVarStmt(ident("locarg" & $idx), s))
    var packanycall = newCall(ident("gopackany"), stmt[0])
    for idx, s in stmt:
        if idx == 0: continue
        packanycall.add(ident("locarg" & $idx))

    nstmt.add(packanycall)
    var topstmt = newIfStmt((ident("true"), nstmt))
    #linfo repr(topstmt)
    #if true: logecho2("aaa", 123)
    result = topstmt

# end gogo2

# 这个pragma好友只在isMainModule生效
# 而且还只能声明一次，pragma already present
{.push hint[XDeclaredButNotUsed]:off.}

{.pop.}


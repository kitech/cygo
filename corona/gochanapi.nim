import times
import strutils

type
    chan*[T] = ref object
        hc: pointer
        val: T       ## send temp ref
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

# public chan API
proc newchan*(T: typedesc, cap:int) : chan[T] =
    var c : chan[T]
    c.new(hchan_finalizer)
    c.hc = hchan_new(cap)
    var val : T
    c.val = val
    c.born = epochTime()
    return c
proc close*[T](c: chan[T]) =
    ## TODO
    discard
proc send*[T](c: chan[T], v : T) : bool {.discardable.} =
    c.val = v  # ref it, but when unref?
    return hchan_send(c.hc, cast[pointer](v))

proc recv*[T](c: chan[T]) : T {.discardable.} =
    var ret : T
    var dat : pointer
    var rv = hchan_recv(c.hc, dat.addr)
    ret = cast[T](dat)
    return ret
import typetraits
proc cap*[T](c: chan[T]) : int = hchan_cap(c.hc)
proc len*[T](c: chan[T]) : int = hchan_len(c.hc)
proc closed*[T](c: chan[T]) : bool = hchan_is_closed(c.hc)
proc `$`*[T](c : chan[T]) : string =
    return "chan[$#; $#]@$#" % [T.name, $(c.cap()), $(c.hc)]
proc toelem[T](c: chan[T], v:pointer) : T =
    var ret : T = cast[T](v)
    return ret

type noimplerr = ref CatchableError

proc `<-`*[T](c: chan[T], v: T) =
    ## Alias of send to chan: `c <- v`
    c.send(v)
    return

proc `<-`*[T](c : chan[T]) : T {.discardable.} =
    ## Alias of recv from chan: `var v = <- c`
    return c.recv()

const caseNil : uint16 = 0
const caseRecv : uint16 = 1
const caseSend : uint16 = 2
const caseDefault : uint16 = 3

type scase = ref object
    hc*: pointer # c hchan*
    hcelem: pointer
    kind*: uint16
    pc: pointer
    reltime: int64

proc newscase*[T](c:chan[T], kind: uint16) : scase =
    var sc = scase()
    case kind:
        of caseNil: pass
        of caseDefault: pass
        else: sc.hc = c.hc
    sc.kind = kind
    return sc

# return -1 on nothing
# if recv, value is chanvec[casi].toelem(casvec[casi].hcelem)
proc goselect1*(casvec: openArray[scase]) : int =
    for idx, cas in casvec:
        linfo(idx, cas.hc, cas.kind)

    var casi : cint = -1
    var sok = goselect(casi.addr, casvec.dtaddr, casvec.len().cint)
    linfo sok, casi, casvec.len
    if sok: linfo("val=", casvec[casi].hcelem)
    return casi

import strformat
import macros
# depcreated one
macro goselectv2(chans: varargs[untyped]) : untyped =
    echo "aaa",treeRepr(chans)
    result = newStmtList()
    echo treeRepr(result)

# TODO golib-nim syntax
macro goselectv3(select_case_expr: untyped) : untyped =
    var s = select_case_expr
    echo treeRepr(s)
    result = newStmtList()
    # basic syntax check
    var rw_scases, default_scases: int
    for scase in s.children:
        if (scase.kind in {nnkCommand, nnkCall} and $(scase[0]) == "scase") or (scase.kind == nnkInfix and $(scase[1]) == "scase"):
            inc rw_scases
        elif scase.kind == nnkCall and $(scase[0]) == "default":
            inc default_scases
        else:
            error(scase.lineinfo & ": Unsupported scase in select.")
    if rw_scases == 0:
        warning(s.lineinfo & ": No send or receive scases.")
    if default_scases > 1:
        error(s.lineinfo & ": More than one 'default' scase.")
    if rw_scases == 0 and default_scases == 0:
        warning(s.lineinfo & ": Empty select blocking forever.")

    # macro expand
    var scase_no = -1
    var codetxt1 = "var selidx = -1\nvar scases = newseq[scase](0)\n"
    var codetxt3 = "if unlikely(selidx < 0): discard\n"

    for scase in s.children:
        inc scase_no
        if (scase.kind in {nnkCommand, nnkCall} and $(scase[0]) == "scase") or
            (scase.kind == nnkInfix and $(scase[1]) == "scase"):
            if scase[1].kind == nnkInfix and $(scase[1][0]) == "<-":
                # send to channel
                # var chidt = $(scase[1][1])
                echo $scase_no, " send "
                codetxt1 &= "scases.add newscase[int](nil, caseSend)\n"
                codetxt1 &= "scases[scases.len-1].hcelem = nil\n"
                discard
            else:
                echo $scase_no, " recv "
                # receive from channel
                codetxt1 &= "scases.add newscase[int](nil, caseRecv)\n"
                discard
            discard
        elif scase.kind == nnkCall and $(scase[0]) == "default":
            echo $(scase[0])
            discard
        codetxt3 &= "elif selidx == " & $scase_no & ": discard\n"

    echo "code text ====="
    echo codetxt1 & "\n" & codetxt3
    result = parseStmt(codetxt1 & "\n" & codetxt3)
    echo "result ststs ====="
    echo treeRepr(result)

# copy src list and insert to dst, just before first empty discard
# both dst and src are StmtList
proc stmtinsertv1(dst:NimNode, src:NimNode) =

    var inspos = -1
    for stmt in dst.children:
        inc inspos
        if stmt.kind == nnkDiscardStmt and stmt[0].kind == nnkEmpty:
            break
    #echo "found inspos ", inspos
    if inspos == -1: inspos = 0

    for idx, stmt in src: dst.insert idx+inspos, stmt
    return

# pure go syntax
# mixed with parseStmt/parseExpr/newStmt.../insert...
macro goselectv5(select_case_expr: untyped) : untyped =
    var s = select_case_expr
    # echo treeRepr(s)
    result = newStmtList()
    var scaseno = -1

    # basic syntax check
    var rw_scases, default_scases: int
    for scase in s.children:
        inc scaseno
        if (scase.kind in {nnkCommand, nnkCall} and $(scase[0]) == "scase") or
            (scase.kind == nnkInfix and $(scase[1]) == "scase"):
            inc rw_scases
        elif scase.kind == nnkAsgn and $(scase[0][0]) == "scase":
            inc rw_scases
        elif scase.kind == nnkCall and $(scase[0]) == "default":
            inc default_scases
        else:
            error(scase.lineinfo & ": Unsupported scase in select." & $scaseno)
    if rw_scases == 0:
        warning(s.lineinfo & ": No send or receive scases.")
    if default_scases > 1:
        error(s.lineinfo & ": More than one 'default' scase.")
    if rw_scases == 0 and default_scases == 0:
        warning(s.lineinfo & ": Empty select blocking forever.")

    # macro expand
    scaseno = -1
    var codetxt1 = "var selidx = -1\nvar scases = newseq[scase](0)\n"
    var codetxt3 = "if unlikely(selidx < 0):\n    echo \"Invalid selidx\"\n    discard\n"

    # pass 1, do if code structure
    for scase in s.children:
        inc scaseno
        codetxt1 &= "# scase $#\n" % [$scaseno]
        codetxt3 &= "elif selidx == " & $scase_no & ":\n"
        if (scase.kind in {nnkCommand, nnkCall} and $(scase[0]) == "scase") or
            (scase.kind == nnkInfix and $(scase[1]) == "scase"):
            if scase[1].kind == nnkInfix and $(scase[1][0]) == "<-":
                # send to channel
                var valexpr = scase[1][2]
                if valexpr.kind in {nnkIntLit, nnkNilLit, nnkFloatLit}:
                    echo $scaseno, " send ", "ch=", $(scase[1][1]), " val=", valexpr.repr
                else: # nnkIdent
                    echo $scaseno, " send ", "ch=", $(scase[1][1]), " val=", $(valexpr)
                codetxt1 &= "scases.add newscase($#, caseSend)\n" % [$(scase[1][1])]
                if valexpr.kind in {nnkIntLit, nnkStrLit, nnkNilLit, nnkFloatLit}:
                    codetxt1 &= "scases[scases.len-1].hcelem = cast[pointer]($#)\n" % [valexpr.repr]
                else: # nnkIdent
                    codetxt1 &= "scases[scases.len-1].hcelem = cast[pointer]($#)\n" % [$(valexpr)]
                discard
            else:
                echo $scaseno, " recv nosave", " ch=", scase[2]
                # receive from channel
                codetxt1 &= "scases.add newscase($#, caseRecv)\n" % [$(scase[2])]
                discard
            discard
        elif scase.kind == nnkAsgn and $(scase[0][0]) == "scase":
            # receive from channel and assign
            echo $scaseno, " recv save", " val=", scase[0][1].repr, " ch=", scase[1][1]
            codetxt1 &= "scases.add newscase($#, caseRecv)\n" % [$scase[1][1]]
            codetxt3 &= "    $# = $#.toelem(scases[selidx].hcelem)\n" % [scase[0][1].repr, $(scase[1][1])]
        elif scase.kind == nnkCall and $(scase[0]) == "default":
            echo $scaseno, " ", $(scase[0])
            codetxt1 &= "scases.add newscase[pointer](nil, caseDefault)\n"
            discard
        codetxt3 &= "    discard\n"

    #echo "code text ====="
    var codetxt2 = "selidx = goselect1(scases)"
    #echo codetxt1 & "\n" & codetxt2 & "\n" & codetxt3
    result = parseStmt(codetxt1 & "\n" & codetxt2 & "\n" & codetxt3)
    #echo "result ststs ====="
    #echo treeRepr(result)

    # pass 2, fill scase body stmts
    var stmtcnt = 0
    for stmt in result: stmtcnt += 1
    var scasecnt = scaseno
    scaseno = -1
    # echo "stmtcnt=", $stmtcnt, " scasecnt=", $scasecnt
    var iftopstmt = result[stmtcnt-1]
    for scase in s.children:
        inc scaseno
        var ifcurbody = iftopstmt[scaseno+1][1]
        if (scase.kind in {nnkCommand, nnkCall} and $(scase[0]) == "scase") or
            (scase.kind == nnkInfix and $(scase[1]) == "scase"):
            if scase[1].kind == nnkInfix and $(scase[1][0]) == "<-":
                # send to channel
                #echo $scaseno, " send ", "ch=", $(scase[1][1]), " val=", $(scase[1][2])
                stmtinsertv1(ifcurbody, scase[2])
                discard
            else:
                #echo $scaseno, " recv nosave", " ch=", scase[2]
                # receive from channel
                stmtinsertv1(ifcurbody, scase[3])
                discard
            discard
        elif scase.kind == nnkAsgn and $(scase[0][0]) == "scase":
            # receive from channel and assign
            #echo $scaseno, " recv save", " val=", scase[0][1].repr, " ch=", scase[1][1]
            stmtinsertv1(ifcurbody, scase[1][2])
        elif scase.kind == nnkCall and $(scase[0]) == "default":
            #echo $scaseno, " ", $(scase[0])
            stmtinsertv1(ifcurbody, scase[1])
            discard

    # echo "result ststs pass 2 ====="
    # echo treeRepr(result)

#[
select:
  caze <- c0:
  caze v1 = <- c0:
  caze c2 <- v2:
  caze c3 <- 123:
  caze c4 <- nil:
  caze c5 <- "foo":
  caze c6 <- 5.678:
  caze <- after(): # TODO
  default:
]#

#macro select3 expand result should be like this:
#[
var selidx = -1
var scases = newseq[scase](0)

scases.add(newscase(c0, caseRecv))
scases.add(newscase(c0, caseRecv))
scases.add(newscase(c1, caseRecv))
scases.add(newscase(c2, caseSend))
scases[scases.len-1].hcelem = cast[pointer](v2)
scases.add(newscase(c3, caseSend))
scases[scases.len-1].hcelem = cast[pointer](123)
scases.add(newscase(c4, caseSend))
scases[scases.len-1].hcelem = cast[pointer](nil)
scases.add(newscase(c5, caseSend))
scases[scases.len-1].hcelem = cast[pointer]("foo")
scases.add(newscase(c6, caseSend))
scases[scases.len-1].hcelem = cast[pointer](5.678)
scases.add(newscase(nil, caseDefault))

selidx = goselect1(scases)

if selidx < 0:
  echo "Invalid selidx"
elif selidx == 0:
  do left stmt0
elif selidx == 1:
  v1 = c1.toelem(scases[selidx].hcelem)
  do left stmt1
elif selidx == 2:
  v2 = c2.toelem(scases[selidx].hcelem)
  do left stmt2
elif selidx == 3:
  do left stmt3
elif selidx == 4:
  do left stmt4
elif selidx == 5:
  do left stmt5
elif selidx == 6:
  do left stmt6
elif selidx == 7: # really else/default branch
  do default scase stmt
]#

{.push hint[XDeclaredButNotUsed]:off.}

{.pop.}

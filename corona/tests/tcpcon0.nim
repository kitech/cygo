
import net
import times
import random
import nativesockets

var ip = "192.30.253.112"
var port = Port(443)

proc connsock() =
    var sock = newSocket()
    linfo("sock", repr(sock.getFd()), now(), ip, port)
    var btime = times.now()
    sock.connect(ip, port)
    linfo("connect done", (times.now()-btime).hmstr, sock.getSocketError())
    sock.close()
    return

# test hook __poll/recvmsg/sendmsg???
proc connsock1() =
    var btime = times.now()
    var info = getAddrInfo("www.github.com", 443.Port)
    linfo("done", (times.now()-btime).hmstr)
    freeAddrInfo(info)
    return

# test connect host name
proc connsock2() =
    var sock = newSocket()
    linfo("sock", repr(sock.getFd()), now())
    var btime = times.now()
    sock.connect("www.github.com", port)
    linfo("connect done", (times.now()-btime).hmstr, sock.getSocketError())
    sock.close()
    return

# test send/recv hook
proc test_http_get1() =
    linfo("begin http get1")
    var ip = "39.156.66.14" # "www.baidu.com"
    var port = 80.Port

    var btime = times.now()
    # need buffered=false, or have last several byte coming wait too long
    var sock = newSocket(buffered=false)
    sock.connect(ip, port)
    linfo("connect done", (times.now()-btime).hmstr)
    let reqtxt = "GET / HTTP/1.1\r\n\r\n"
    var rv = sock.send(reqtxt.cstring, reqtxt.len)
    linfo("rv=", rv, reqtxt.len, sock.getSocketError())
    while true:
        var buf = sock.recv(123)
        if buf.len != 123: linfo("recved", buf.len, "/", 123)
        if buf == "": break

    linfo("done")

proc test_accept1s() =
    var sock = newSocket(buffered=false)
    sock.setSockOpt(OptReusePort, true)
    sock.bindAddr(5678.Port)
    sock.listen()
    while true:
        linfo("accepting ...")
        var sk : Socket
        sock.accept(sk)
        linfo("newconn fd=", sk.getFd().repr)
        sk.close()
        break
    sock.close()
    return
proc test_accept1c() =
    var sock = newSocket(buffered=false)
    sock.connect("127.0.0.1", 5678.Port)
    return
proc test_accept1() =
    noro_post(test_accept1s, nil)
    sleep(100)
    noro_post(test_accept1c, nil)
    return

# echo server/client test.
# 32 个client 发起请求并接收
var echo_srv_got0 = 0
var echo_cli_got0 = 0
proc test_echo_srv_handle0(conn:Socket) =
    sleep(rand(123))
    var buf = newStringOfCap(32)
    var rn = conn.recv(buf, buf.cap)
    sleep(123)
    conn.send(buf)
    # conn.close()
    echo_srv_got0 += 1
    return
proc test_echo_srv0(cnt:int) =
    var sock = newSocket(buffered=false)
    sock.setSockOpt(OptReuseAddr, true)
    sock.setSockOpt(OptReusePort, true)
    sock.bindAddr(5678.Port)
    sock.listen()
    var i = 0
    while true:
        var sk : Socket
        sock.accept(sk)
        noro_post(test_echo_srv_handle0, cast[pointer](sk))
        i += 1
        if i == cnt: break
    sock.close()
    linfo("echo srv done")
    return

proc test_echo_cli_worker0(no:int) =
    var sock = newSocket(buffered=false)
    try:
        sock.connect("127.0.0.1", 5678.Port)
    except:
        linfo("cli err", getCurrentExceptionMsg())
    for i in 0..< 1:
        sock.send("this is echo lelelele cli " & $no)
        sleep(rand(300)+123)
        var buf = newStringOfCap(64)
        var rv = sock.recv(buf, buf.cap)
        assert(rv == buf.len)
    echo_cli_got0 += 1
    sleep(1234)
    sock.close()
    return
proc test_echo_cli0() =
    for i in 0..< 32:
        sleep(rand(500))
        noro_post(test_echo_cli_worker0, cast[pointer](i))
    var i = 0
    while true:
        i += 1
        usleepc(100000) # 100ms
        if echo_srv_got0 == echo_cli_got0 and echo_cli_got0 == 32: break
        assert(i < 20, "wait cli done timedout" & $echo_srv_got0 & " " & $echo_cli_got0)
    linfo("echo cli done",  echo_srv_got0, echo_cli_got0)
    return

proc runtest_tcpconm() =
    connsock()
    connsock1()
    connsock2()
    test_http_get1()

proc runtest_tcpcon0() =
    # noro_post(runtest_tcpconm, nil)
    # test_accept1()
    noro_post(test_echo_srv0, cast[pointer](32))
    sleep(300)
    noro_post(test_echo_cli0, nil)
    return


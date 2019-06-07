
import net
import times
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

proc runtest_tcpconm() =
    connsock()
    connsock1()
    connsock2()
    test_http_get1()

proc runtest_tcpcon0() =
    noro_post(runtest_tcpconm, nil)

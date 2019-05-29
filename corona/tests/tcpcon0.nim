
import net
import times

var ip = "192.30.253.112"
var port = Port(443)

proc connsock() =
    var sock = newSocket()
    linfo("sock", repr(sock.getFd()), now(), ip, port)
    var btime = times.now()
    sock.connect(ip, port)
    linfo("connect done", times.now()-btime)
    return

proc runtest_tcpcon0() =
    noro_post(connsock, nil)

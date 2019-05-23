
import net
import times

var ip = "192.30.253.112"
var port = Port(443)

proc connsock() =
    var sock = newSocket()
    linfo("sock", repr(sock.getFd()), now(), ip, port)
    sock.connect(ip, port)
    linfo("connect done", now())
    return

noro_post(connsock, nil)

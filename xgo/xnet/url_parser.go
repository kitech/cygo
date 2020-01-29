package xnet

/*
// #include "url_parser.h"

*/
import "C"

// http://draft.scyphus.co.jp/lang/c/url_parser.html

type cparsed_url struct {
	scheme   byteptr /* mandatory */
	host     byteptr /* mandatory */
	port     byteptr /* optional */
	path     byteptr /* optional */
	query    byteptr /* optional */
	fragment byteptr /* optional */
	username byteptr /* optional */
	password byteptr /* optional */
}

type Url struct {
	Scheme   string /* mandatory */
	Host     string /* mandatory */
	Port     string /* optional */
	Path     string /* optional */
	Query    string /* optional */
	Fragment string /* optional */
	Username string /* optional */
	Password string /* optional */
}

func cpurl2(cu *cparsed_url) *Url {
	if cu == nil {
		return nil
	}
	uo := &Url{}
	uo.Scheme = gostring_clone(cu.scheme)
	uo.Host = gostring_clone(cu.host)
	uo.Port = gostring_clone(cu.port)
	uo.Path = gostring_clone(cu.path)
	uo.Query = gostring_clone(cu.query)
	uo.Fragment = gostring_clone(cu.fragment)
	uo.Username = gostring_clone(cu.username)
	uo.Password = gostring_clone(cu.password)

	return uo
}
func (uo *Url) Portno0() int {
	if uo.Port != 0 {
		return uo.Port.toint()
	}
	scheme := uo.Scheme.tolower()
	switch scheme {
	case "http":
		return 80
	case "https":
		return 443
	case "ftp":
		return 21
	}
	return uo.Port
}

func ParseUrl(u string) *Url {
	uo1 := C.xnet_parse_url(u.cstr())
	if uo1 == nil {
		return nil
	}
	uo := cpurl2(uo1)
	C.xnet_parsed_url_free(uo1)

	return uo
}

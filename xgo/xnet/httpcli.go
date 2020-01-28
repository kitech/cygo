package xnet

type Request struct {
	Method  string
	Headers map[string]string
	Uri     string
}

type Response struct {
	Stcode   int
	Stline   string
	Ctlength i64
	Headers  map[string]string
	Data     string
}

type Client struct {
}

func NewRequest(method string, uri string, data voidptr) *Request {
	req := &Request{}
	req.Method = method
	req.Uri = uri

	return req
}

func (c *Client) Do(req *Request) (*Response, error) {
	return nil, nil
}

func Get() {

}

func Post() {

}

func Put() {

}

func Delete() {

}

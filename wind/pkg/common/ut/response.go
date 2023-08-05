package ut

import (
	"bytes"

	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
)

// ResponseRecorder 记录处理器的响应以供稍后测试。
type ResponseRecorder struct {
	Code        int
	header      *protocol.ResponseHeader
	Body        *bytes.Buffer
	Flushed     bool
	result      *protocol.Response
	wroteHeader bool
}

// NewRecorder 返回一个实例化的响应记录器。
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Code:   consts.StatusOK,
		header: new(protocol.ResponseHeader),
		Body:   new(bytes.Buffer),
	}
}

// Header 返回响应标头以便在处理器中修改（mutate）。
// 要想测试在处理器完成后写入的标头，使用 Result 方法并查看返回的响应值的 Header。
func (r *ResponseRecorder) Header() *protocol.ResponseHeader {
	m := r.header
	if m == nil {
		m = new(protocol.ResponseHeader)
		r.header = m
	}
	return m
}

func (r *ResponseRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(consts.StatusOK)
	}
	if r.Body != nil {
		r.Body.Write(p)
	}
	return len(p), nil
}

func (r *ResponseRecorder) WriteString(s string) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(consts.StatusOK)
	}
	if r.Body != nil {
		r.Body.WriteString(s)
	}
	return len(s), nil
}

func (r *ResponseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	if r.header == nil {
		r.header = new(protocol.ResponseHeader)
	}
	r.header.SetStatusCode(code)
	r.Code = code
	r.wroteHeader = true
}

func (r *ResponseRecorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(consts.StatusOK)
	}
	r.Flushed = true
}

func (r *ResponseRecorder) Result() *protocol.Response {
	if r.result != nil {
		return r.result
	}

	res := new(protocol.Response)
	h := r.Header()
	h.CopyTo(&res.Header)
	if r.Body != nil {
		b := r.Body.Bytes()
		res.SetBody(b)
		res.Header.SetContentLength(len(b))
	}

	r.result = res
	return res
}

package protocol

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
	"sync"

	"github.com/favbox/gobox/hertz/internal/bytesconv"
	"github.com/favbox/gobox/hertz/internal/bytestr"
	"github.com/favbox/gobox/hertz/internal/nocopy"
	"github.com/favbox/gobox/hertz/pkg/common/bytebufferpool"
	"github.com/favbox/gobox/hertz/pkg/common/compress"
	"github.com/favbox/gobox/hertz/pkg/common/config"
	"github.com/favbox/gobox/hertz/pkg/common/errors"
	"github.com/favbox/gobox/hertz/pkg/common/utils"
	"github.com/favbox/gobox/hertz/pkg/network"
	"github.com/favbox/gobox/hertz/pkg/protocol/consts"
)

var (
	errMissingFile = errors.NewPublic("http: 无此文件")

	// 减少 GC 的请求体缓冲池
	requestBodyPool bytebufferpool.Pool

	// 减少 GC 的请求实例池
	requestPool sync.Pool
)

// NoBody 是一个无字节的 io.ReadCloser。
// Read 始终返回 EOF，Close 总是返回 nil。
// NoBody 可用于发送客户端请求时，明确请求的消息体为零字节。
// 另一种实现：只需将 Request.Body 设置为 nil。
var NoBody = noBody{}

type noBody struct{}

func (noBody) Read([]byte) (int, error) { return 0, nil }
func (noBody) Close() error             { return nil }

type Request struct {
	noCopy nocopy.NoCopy

	Header RequestHeader

	uri      URI
	postArgs Args

	bodyRaw         []byte
	bodyStream      io.Reader
	body            *bytebufferpool.ByteBuffer
	maxKeepBodySize int
	w               requestBodyWriter

	multipartForm         *multipart.Form
	multipartFormBoundary string
	multipartFiles        []*File
	multipartFields       []*MultipartField

	// URI 是否已解析
	parsedURI bool
	// Post Args 是否已解析
	parsedPostArgs bool

	isTLS bool

	options *config.RequestOptions
}

// File 表示 multipart 请求的文件信息结构体。
type File struct {
	io.Reader

	Name      string // 文件路径
	ParamName string // 文件名称
}

// MultipartField 表示 multipart 请求中自定义数据部分的结构体。
type MultipartField struct {
	io.Reader

	Param       string
	FileName    string
	ContentType string
}

type requestBodyWriter struct {
	r *Request
}

func (w *requestBodyWriter) Write(p []byte) (int, error) {
	w.r.AppendBody(p)
	return len(p), nil
}

// AppendBody 追加 p 至请求体的字节缓冲区。
//
// 函数返回后，对 p 的复用是安全的。
func (req *Request) AppendBody(p []byte) {
	req.RemoveMultipartFormFiles()
	_ = req.CloseBodyStream()
	_, _ = req.BodyBuffer().Write(p)
}

// AppendBodyString 追加 s 至请求体的字节缓冲区。
func (req *Request) AppendBodyString(s string) {
	req.RemoveMultipartFormFiles()
	_ = req.CloseBodyStream()
	_, _ = req.BodyBuffer().WriteString(s)
}

// BasicAuth 如果请求使用了 HTTP 基本验证，可以返回请求 Authorization 中的用户名密码。
func (req *Request) BasicAuth() (username, password string, ok bool) {
	// 使用 Peek 降低类型转换成本
	auth := req.Header.Peek(consts.HeaderAuthorization)
	if auth == nil {
		return
	}

	return parseBasicAuth(auth)
}

var prefix = []byte{'B', 'a', 's', 'i', 'c', ' '}

// 可解析 base64 编码的 HTTP 基本验证字符串。
// 例如： "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" 返回 ("Aladdin", "open sesame", true)。
func parseBasicAuth(auth []byte) (username, password string, ok bool) {
	if len(auth) < len(prefix) || !bytes.EqualFold(auth[:len(prefix)], prefix) {
		return
	}

	decodeLen := base64.StdEncoding.DecodedLen(len(auth[len(prefix):]))
	decodeData := make([]byte, decodeLen)
	num, err := base64.StdEncoding.Decode(decodeData, auth[len(prefix):])
	if err != nil {
		return
	}

	cs := bytesconv.B2s(decodeData[:num])
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}

	return cs[:s], cs[s+1:], true
}

// Body 返回请求体。
// 如果获取失败则返回 nil。
func (req *Request) Body() []byte {
	body, _ := req.BodyE()
	return body
}

// BodyE 返回请求体和错误。
// 如果获取失败则返回 nil。
func (req *Request) BodyE() ([]byte, error) {
	if req.bodyRaw != nil {
		return req.bodyRaw, nil
	}
	if req.IsBodyStream() {
		bodyBuf := req.BodyBuffer()
		bodyBuf.Reset()
		zw := network.NewWriter(bodyBuf)
		_, err := utils.CopyZeroAlloc(zw, req.bodyStream)
		_ = req.CloseBodyStream()
		if err != nil {
			return nil, err
		}
		return req.BodyBytes(), nil
	}
	if req.OnlyMultipartForm() {
		body, err := MarshalMultipartForm(req.multipartForm, req.multipartFormBoundary)
		if err != nil {
			return nil, err
		}
		return body, nil
	}
	return req.BodyBytes(), nil
}

func (req *Request) BodyBytes() []byte {
	if req.bodyRaw != nil {
		return req.bodyRaw
	}
	if req.body == nil {
		return nil
	}
	return req.body.B
}

func (req *Request) BodyBuffer() *bytebufferpool.ByteBuffer {
	if req.body == nil {
		req.body = requestBodyPool.Get()
	}
	req.bodyRaw = nil
	return req.body
}

func (req *Request) BodyStream() io.Reader {
	if req.bodyStream == nil {
		req.bodyStream = NoBody
	}
	return req.bodyStream
}

// BodyWriter 返回用于填充请求体的编写器。
func (req *Request) BodyWriter() io.Writer {
	req.w.r = req
	return &req.w
}

// BodyWriteTo 将请求体写入 w。
func (req *Request) BodyWriteTo(w io.Writer) error {
	if req.IsBodyStream() {
		zw := network.NewWriter(w)
		_, err := utils.CopyZeroAlloc(zw, req.bodyStream)
		_ = req.CloseBodyStream()
		return err
	}
	if req.OnlyMultipartForm() {
		return WriteMultipartForm(w, req.multipartForm, req.multipartFormBoundary)
	}
	_, err := w.Write(req.BodyBytes())
	return err
}

// CloseBodyStream 关闭请求体数据流。
func (req *Request) CloseBodyStream() error {
	if req.bodyStream == nil {
		return nil
	}

	var err error
	if bsc, ok := req.bodyStream.(io.Closer); ok {
		err = bsc.Close()
	}
	req.bodyStream = nil
	return err
}

// ConnectionClose 若标头 'Connection: close' 已设定则返回真。
func (req *Request) ConnectionClose() bool {
	return req.Header.ConnectionClose()
}

func (req *Request) ConstructBodyStream(body *bytebufferpool.ByteBuffer, bodyStream io.Reader) {
	req.body = body
	req.bodyStream = bodyStream
}

// CopyTo 拷贝正文流之外的请求内容到 dst。
func (req *Request) CopyTo(dst *Request) {
	req.CopyToSkipBody(dst)
	if req.bodyRaw != nil {
		dst.bodyRaw = append(dst.bodyRaw[:0], req.bodyRaw...)
		if dst.body != nil {
			dst.body.Reset()
		}
	} else if req.body != nil {
		dst.BodyBuffer().Set(req.body.B)
	} else if dst.body != nil {
		dst.body.Reset()
	}
}

func (req *Request) CopyToSkipBody(dst *Request) {
	dst.Reset()
	req.Header.CopyTo(&dst.Header)

	req.uri.CopyTo(&dst.uri)
	dst.parsedURI = req.parsedURI

	req.postArgs.CopyTo(&dst.postArgs)
	dst.parsedPostArgs = req.parsedPostArgs
	dst.isTLS = req.isTLS

	if req.options != nil {
		dst.options = &config.RequestOptions{}
		req.options.CopyTo(dst.options)
	}

	// 无需拷贝 multipartForm - 它会在第一次被调用时自动重建。
}

// FormFile 返回指定表单 name 的第一个文件。
func (req *Request) FormFile(name string) (*multipart.FileHeader, error) {
	mf, err := req.MultipartForm()
	if err != nil {
		return nil, err
	}
	if mf.File == nil {
		return nil, err
	}
	fhh := mf.File[name]
	if fhh == nil {
		return nil, errMissingFile
	}
	return fhh[0], nil
}

func (req *Request) HasMultipartForm() bool {
	return req.multipartForm != nil
}

// Host 返回指定请求的主机。
func (req *Request) Host() []byte {
	return req.URI().Host()
}

// IsURIParsed 返回 URI 是否已解析。
func (req *Request) IsURIParsed() bool {
	return req.parsedURI
}

// IsBodyStream 判断请求体是否由 SetBodyStream 设置。
func (req *Request) IsBodyStream() bool {
	return req.bodyStream != nil && req.bodyStream != NoBody
}

// MayContinue 若请求头包含 'Expect: 100-continue' 则返回真。
//
// 若返回真，调用者必须执行一个如下动作：
//
//   - 若请求头不满足调用方的要求，则发送 StatusExpectationFailed 响应。
//   - 或者在用 ContinueReadBody 读取请求正文之前发送 StatusContinue 响应。
//   - 再或者关闭连接。
func (req *Request) MayContinue() bool {
	return bytes.Equal(req.Header.peek(bytestr.StrExpect), bytestr.Str100Continue)
}

// Method 返回请求方法。
func (req *Request) Method() []byte {
	return req.Header.Method()
}

// MultipartFields 返回表单字段切片。
func (req *Request) MultipartFields() []*MultipartField {
	return req.multipartFields
}

// MultipartFiles 返回表单文件切片。
func (req *Request) MultipartFiles() []*File {
	return req.multipartFiles
}

// MultipartForm 返回请求表单。
//
// 若请求的内容类型不是 'multipart/form-data' 则返回 errors.ErrNoMultipartForm。
//
// 在返回的 multipart 表单被处理后，一定要调用 RemoveMultipartFormFiles。
func (req *Request) MultipartForm() (*multipart.Form, error) {
	if req.multipartForm != nil {
		return req.multipartForm, nil
	}
	req.multipartFormBoundary = string(req.Header.MultipartFormBoundary())
	if len(req.multipartFormBoundary) == 0 {
		return nil, errors.ErrNoMultipartForm
	}

	ce := req.Header.peek(bytestr.StrContentEncoding)
	var err error
	var f *multipart.Form

	if !req.IsBodyStream() {
		body := req.BodyBytes()
		if bytes.Equal(ce, bytestr.StrGzip) {
			// 这里不关心内存使用情况
			var err error
			if body, err = compress.AppendGunzipBytes(nil, body); err != nil {
				return nil, fmt.Errorf("无法解压缩请求体：%s", err)
			}
		} else if len(ce) > 0 {
			return nil, fmt.Errorf("不支持的内容编码：%q", ce)
		}
		f, err = ReadMultipartForm(bytes.NewReader(body), req.multipartFormBoundary, len(body), len(body))
	} else {
		bodyStream := req.bodyStream
		if req.Header.contentLength > 0 {
			bodyStream = io.LimitReader(bodyStream, int64(req.Header.contentLength))
		}
		if bytes.Equal(ce, bytestr.StrGzip) {
			// Do not care about memory usage here.
			if bodyStream, err = gzip.NewReader(bodyStream); err != nil {
				return nil, fmt.Errorf("无法解压缩请求体：%w", err)
			}
		} else if len(ce) > 0 {
			return nil, fmt.Errorf("不支持的内容编码：%q", ce)
		}

		mr := multipart.NewReader(bodyStream, req.multipartFormBoundary)

		f, err = mr.ReadForm(8 * 1024)
	}

	if err != nil {
		return nil, err
	}
	req.multipartForm = f
	return f, nil
}

func (req *Request) MultipartFormBoundary() string {
	return req.multipartFormBoundary
}

func (req *Request) OnlyMultipartForm() bool {
	return req.multipartForm != nil && (req.body == nil || len(req.body.B) == 0)
}

// Options 返回请求选项。
func (req *Request) Options() *config.RequestOptions {
	if req.options == nil {
		req.options = config.NewRequestOptions(nil)
	}
	return req.options
}

// Path 返回请求路径。
func (req *Request) Path() []byte {
	return req.URI().Path()
}

// ParseURI 解析请求的完全限定 URI。
func (req *Request) ParseURI() {
	if req.parsedURI {
		return
	}
	req.parsedURI = true

	req.uri.parse(req.Header.Host(), req.Header.RequestURI(), req.isTLS)
}

// PostArgs 返回 POST 参数。
func (req *Request) PostArgs() *Args {
	req.parsePostArgs()
	return &req.postArgs
}

// PostArgString 返回 POST 参数的查询字符串。
func (req *Request) PostArgString() []byte {
	return req.postArgs.QueryString()
}

func (req *Request) parsePostArgs() {
	if req.parsedPostArgs {
		return
	}
	req.parsedPostArgs = true

	if !bytes.HasPrefix(req.Header.ContentType(), bytestr.StrPostArgsContentType) {
		return
	}
	req.postArgs.ParseBytes(req.Body())
}

// QueryString 返回请求的查询字符串。
func (req *Request) QueryString() []byte {
	return req.URI().QueryString()
}

// RemoveMultipartFormFiles 移除该请求关联的 multipart/form-data 临时文件。
func (req *Request) RemoveMultipartFormFiles() {
	if req.multipartForm != nil {
		// 忽略错误。因为这些文件可能被用户删除或移到他处。
		_ = req.multipartForm.RemoveAll()
		req.multipartForm = nil
	}
	req.multipartFormBoundary = ""
	req.multipartFiles = nil
	req.multipartFields = nil
}

// RequestURI 返回完全限定的请求网址。
func (req *Request) RequestURI() []byte {
	return req.Header.RequestURI()
}

// Reset 重置请求。
func (req *Request) Reset() {
	req.Header.Reset()
	req.ResetSkipHeader()
	_ = req.CloseBodyStream()

	req.options = nil
}

// ResetSkipHeader 重置请求（跳过标头）。
func (req *Request) ResetSkipHeader() {
	req.ResetBody()
	req.uri.Reset()
	req.parsedURI = false
	req.parsedPostArgs = false
	req.postArgs.Reset()
	req.isTLS = false
}

// ResetBody 重置请求体。
func (req *Request) ResetBody() {
	req.bodyRaw = nil
	req.RemoveMultipartFormFiles()
	_ = req.CloseBodyStream()
	if req.body != nil {
		if req.body.Len() <= req.maxKeepBodySize {
			req.body.Reset()
			return
		}
		requestBodyPool.Put(req.body)
		req.body = nil
	}
}

// Scheme 返回请求架构。
func (req *Request) Scheme() []byte {
	return req.uri.Scheme()
}

// SetAuthSchemeToken 设置身份验证架构和令牌。例如：
//
//	Authorization: <auth-scheme-value-set-here> <auth-token-value>
func (req *Request) SetAuthSchemeToken(scheme, token string) {
	req.SetHeader(consts.HeaderAuthorization, scheme+" "+token)
}

// SetAuthToken 设置身份验证的令牌（默认 Scheme 方案：Bearer）。例如：
//
//	Authorization: Bearer <auth-token-value-comes-here>
func (req *Request) SetAuthToken(token string) {
	req.SetHeader(consts.HeaderAuthorization, "Bearer "+token)
}

// SetBasicAuth 设置基本身份验证标头（默认 Scheme 方案：Basic）。例如：
//
// Authorization: Basic <username>:<password>
func (req *Request) SetBasicAuth(username, password string) {
	encodeStr := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	req.SetHeader(consts.HeaderAuthorization, "Basic "+encodeStr)
}

// SetBody 设置请求体。
//
// 在函数返回后，重新使用 body 参数是安全的。
func (req *Request) SetBody(body []byte) {
	req.RemoveMultipartFormFiles()
	_ = req.CloseBodyStream()
	req.BodyBuffer().Set(body)
}

// SetBodyString 设置请求体。
func (req *Request) SetBodyString(body string) {
	req.RemoveMultipartFormFiles()
	_ = req.CloseBodyStream()
	req.BodyBuffer().SetString(body)
}

// SetBodyRaw 设置请求体，但不复制它。
//
// 基于此，内容体不可改变。
func (req *Request) SetBodyRaw(body []byte) {
	req.ResetBody()
	req.bodyRaw = body
}

// SetBodyStream 设置请求体的流和大小（可选）。
//
// 若 bodySize >= 0，那么在返回 io.EOF 之前，bodyStream 必须提供确切的 bodySize 字节。
//
// 若 bodySize < 0，那么, 则读取 bodyStream 直至 io.EOF。
//
// 若 bodyStream 实现了 io.Closer，则读取完所有请求体数据后调用 bodyStream.Close()。
//
// 注意：GET 和 HEAD 请求不能有请求体。
func (req *Request) SetBodyStream(bodyStream io.Reader, bodySize int) {
	req.ResetBody()
	req.bodyStream = bodyStream
	req.Header.SetContentLength(bodySize)
}

// SetConnectionClose 设置连接关闭 'Connection: close' 标头。
func (req *Request) SetConnectionClose() {
	req.Header.SetConnectionClose(true)
}

// SetCookie 附加单个 cookie 到当前请求。
func (req *Request) SetCookie(key, value string) {
	req.Header.SetCookie(key, value)
}

// SetCookies 附加多个 cookie 到当前请求。
func (req *Request) SetCookies(hc map[string]string) {
	for k, v := range hc {
		req.Header.SetCookie(k, v)
	}
}

// SetFile 附加单个文件名称和路径到文件上传表单中。
func (req *Request) SetFile(param, filePath string) {
	req.multipartFiles = append(req.multipartFiles, &File{
		Name:      filePath,
		ParamName: param,
	})
}

// SetFiles 附加多个文件名称和路径到文件上传表单中。
func (req *Request) SetFiles(files map[string]string) {
	for f, fp := range files {
		req.multipartFiles = append(req.multipartFiles, &File{
			Name:      fp,
			ParamName: f,
		})
	}
}

// SetFormData 设置 x-www-form-urlencoded 请求参数。
func (req *Request) SetFormData(data map[string]string) {
	for k, v := range data {
		req.postArgs.Add(k, v)
	}
	req.parsedPostArgs = true
	req.Header.SetContentTypeBytes(bytestr.StrPostArgsContentType)
}

// SetFormDataFromValues 根据 url.Values 设置 x-www-form-urlencoded 请求参数。
func (req *Request) SetFormDataFromValues(data url.Values) {
	for k, v := range data {
		for _, kv := range v {
			req.postArgs.Add(k, kv)
		}
	}
	req.parsedPostArgs = true
	req.Header.SetContentTypeBytes(bytestr.StrPostArgsContentType)
}

// SetHeader 在当前请求头中设置单个标头字段和值。
func (req *Request) SetHeader(header, value string) {
	req.Header.Set(header, value)
}

// SetHeaders 在当前请求头中设置多个标头字段和值。
func (req *Request) SetHeaders(headers map[string]string) {
	for h, v := range headers {
		req.Header.Set(h, v)
	}
}

// SetHost 设置请求的主机。
func (req *Request) SetHost(host string) {
	req.URI().SetHost(host)
}

// SetIsTLS 用于 TLS 服务器标记该请求是否为 TLS 请求。
// 客户端不应使用该方法，应依赖于 URI 的 schema。
func (req *Request) SetIsTLS(isTLS bool) {
	req.isTLS = isTLS
}

// SetMaxKeepBodySize 设置请求体的最大保留字节数。
func (req *Request) SetMaxKeepBodySize(n int) {
	req.maxKeepBodySize = n
}

// SetMethod 设置请求方法。
func (req *Request) SetMethod(method string) {
	req.Header.SetMethod(method)
}

// SetMultipartField 根据 io.Reader 附加自定义表单字段。
func (req *Request) SetMultipartField(param, fileName, contentType string, reader io.Reader) {
	req.multipartFields = append(req.multipartFields, &MultipartField{
		Param:       param,
		FileName:    fileName,
		ContentType: contentType,
		Reader:      reader,
	})
}

// SetMultipartFields 附加多个自定义表单字段。
func (req *Request) SetMultipartFields(fields ...*MultipartField) {
	req.multipartFields = append(req.multipartFields, fields...)
}

// SetMultipartFormBoundary 设置表单边界值。
func (req *Request) SetMultipartFormBoundary(b string) {
	req.multipartFormBoundary = b
}

// SetMultipartFormData 根据 map 附加自定义表单字段。
func (req *Request) SetMultipartFormData(data map[string]string) {
	for k, v := range data {
		req.SetMultipartField(k, "", "", strings.NewReader(v))
	}
}

// SetOptions 自定义请求选项。
// 用于在中间件中执行某些操作，如服务发现。
func (req *Request) SetOptions(opts ...config.RequestOption) {
	req.Options().Apply(opts)
}

// SetQueryString 设置查询字符串。
func (req *Request) SetQueryString(queryString string) {
	req.URI().SetQueryString(queryString)
}

// SetRequestURI 设置完全限定的网址，即包含 scheme 和 host。
func (req *Request) SetRequestURI(requestURI string) {
	req.Header.SetRequestURI(requestURI)
	req.parsedURI = false
}

// SwapBody 将请求主体与指定主体交换，并返回请求主体。
//
// 禁止在函数返回后修改传递给 SwapBody 的主体。
func (req *Request) SwapBody(body []byte) []byte {
	bb := req.BodyBuffer()
	zw := network.NewWriter(bb)

	if req.IsBodyStream() {
		bb.Reset()
		_, err := utils.CopyZeroAlloc(zw, req.bodyStream)
		req.CloseBodyStream() //nolint:errcheck
		if err != nil {
			bb.Reset()
			bb.SetString(err.Error())
		}
	}

	req.bodyRaw = nil

	oldBody := bb.B
	bb.B = body
	return oldBody
}

// URI 返回请求的 URI
func (req *Request) URI() *URI {
	req.ParseURI()
	return &req.uri
}

// AcquireRequest 从请求池返回一个空的请求实例。
//
// 当实例不再用时，调用 ReleaseRequest 进行释放，以减少 GC，提高性能。
func AcquireRequest() *Request {
	v := requestPool.Get()
	if v == nil {
		return &Request{}
	}
	return v.(*Request)
}

// ReleaseRequest 将通过 AcquireRequest 获取的请求实例放回池中。
//
// 放回池后禁止再调该实例或成员。
func ReleaseRequest(req *Request) {
	req.Reset()
	requestPool.Put(req)
}

// NewRequest 根据给定方法、网址和可选正文构造新的请求实例。
//
// # 方法默认值是 GET。
//
// 网址必须为完全限定的 URI，即带有 scheme 和 host，若省略 scheme 则假定为 http。
//
// 该方法的协议版本固定为 HTTP/1.1
func NewRequest(method, url string, body io.Reader) *Request {
	if method == "" {
		method = consts.MethodGet
	}

	req := new(Request)
	req.SetRequestURI(url)
	req.SetIsTLS(bytes.HasPrefix(bytesconv.S2b(url), bytestr.StrHTTPS))
	req.ParseURI()
	req.SetMethod(method)
	req.Header.SetHostBytes(req.URI().Host())
	req.Header.SetRequestURIBytes(req.URI().RequestURI())

	if !req.Header.IgnoreBody() {
		req.SetBodyStream(body, -1)
		switch v := req.BodyStream().(type) {
		case *bytes.Buffer:
			req.Header.SetContentLength(v.Len())
		case *bytes.Reader:
			req.Header.SetContentLength(v.Len())
		case *strings.Reader:
			req.Header.SetContentLength(v.Len())
		default:
		}
	}

	return req
}

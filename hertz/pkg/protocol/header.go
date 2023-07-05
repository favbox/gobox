package protocol

import (
	"bytes"

	"github.com/favbox/gobox/hertz/internal/bytesconv"
	"github.com/favbox/gobox/hertz/internal/bytestr"
	"github.com/favbox/gobox/hertz/internal/nocopy"
	errs "github.com/favbox/gobox/hertz/pkg/common/errors"
	"github.com/favbox/gobox/hertz/pkg/common/utils"
	"github.com/favbox/gobox/hertz/pkg/protocol/consts"
)

// RequestHeader 表示 HTTP 请求头。
//
// 禁止复制 RequestHeader 实例，而是通过创建新实例并 CopyTo 来替代。
//
// RequestHeader 实例 *不能* 用于多协程，并发不是安全的。
type RequestHeader struct {
	noCopy nocopy.NoCopy

	disableNormalizing   bool
	connectionClose      bool
	noDefaultContentType bool

	cookiesCollected bool

	contentLength      int
	contentLengthBytes []byte

	method      []byte
	requestURI  []byte
	host        []byte
	contentType []byte

	userAgent []byte
	mulHeader [][]byte
	protocol  string

	h       []argsKV
	bufKV   argsKV
	trailer *Trailer

	cookies []argsKV

	// 存储从 wire 接收的不可变副本。
	rawHeaders []byte
}

// Add 添加一对请求头KV。
// Add 可以添加相同键的多个值。使用 Set 设置单个键值对。
//
// Content-Type, Content-Length, Connection, Cookie,
// Transfer-Encoding, Host 和 User-Agent 只能设置一次，新值会覆盖旧值。
func (h *RequestHeader) Add(key, value string) {
	if h.setSpecialHeader(bytesconv.S2b(key), bytesconv.S2b(value)) {
		return
	}

	k := getHeaderKeyBytes(&h.bufKV, key, h.disableNormalizing)
	h.h = appendArg(h.h, bytesconv.B2s(k), value, ArgsHasValue)
}

// AddArgBytes 添加一对请求头KV。
func (h *RequestHeader) AddArgBytes(key, value []byte, noValue bool) {
	h.h = appendArgBytes(h.h, key, value, noValue)
}

// AppendBytes 将请求头关键信息附加到 dst 并返回。
func (h *RequestHeader) AppendBytes(dst []byte) []byte {
	dst = append(dst, h.Method()...)
	dst = append(dst, ' ')
	dst = append(dst, h.RequestURI()...)
	dst = append(dst, ' ')
	dst = append(dst, bytestr.StrHTTP11...)
	dst = append(dst, bytestr.StrCRLF...)

	userAgent := h.UserAgent()
	if len(userAgent) > 0 {
		dst = appendHeaderLine(dst, bytestr.StrUserAgent, userAgent)
	}

	host := h.Host()
	if len(host) > 0 {
		dst = appendHeaderLine(dst, bytestr.StrHost, host)
	}

	contentType := h.ContentType()
	if len(contentType) == 0 && !h.IgnoreBody() && !h.noDefaultContentType {
		contentType = bytestr.StrPostArgsContentType
	}
	if len(contentType) > 0 {
		dst = appendHeaderLine(dst, bytestr.StrContentType, contentType)
	}
	if len(h.contentLengthBytes) > 0 {
		dst = appendHeaderLine(dst, bytestr.StrContentLength, h.contentLengthBytes)
	}

	for i, n := 0, len(h.h); i < n; i++ {
		kv := &h.h[i]
		dst = appendHeaderLine(dst, kv.key, kv.value)
	}

	if !h.Trailer().Empty() {
		dst = appendHeaderLine(dst, bytestr.StrTrailer, h.Trailer().GetBytes())
	}

	// 这里不能用 h.collectCookies() 因为 cookies 尚未收集，直接处理 h.cookies 键值对切片。
	n := len(h.cookies)
	if n > 0 {
		dst = append(dst, bytestr.StrCookie...)
		dst = append(dst, bytestr.StrColonSpace...)
		dst = appendRequestCookieBytes(dst, h.cookies)
		dst = append(dst, bytestr.StrCRLF...)
	}

	if h.connectionClose {
		dst = appendHeaderLine(dst, bytestr.StrConnection, bytestr.StrClose)
	}

	return append(dst, bytestr.StrCRLF...)
}

// 从 h.h 中收集 Cookie 值并解析至 h.cookies 切片。
//
// 若已解析则直接返回。
func (h *RequestHeader) collectCookies() {
	if h.cookiesCollected {
		return
	}

	for i, n := 0, len(h.h); i < n; i++ {
		kv := &h.h[i]
		if bytes.Equal(kv.key, bytestr.StrCookie) {
			h.cookies = parseRequestCookies(h.cookies, kv.value)
			tmp := *kv
			copy(h.h[i:], h.h[i+1:])
			n--
			i--
			h.h[n] = tmp
			h.h = h.h[:n]
		}
	}
	h.cookiesCollected = true
}

// ConnectionClose 返回标头中是否设置了 'Connection: close'。
func (h *RequestHeader) ConnectionClose() bool {
	return h.connectionClose
}

// ContentLength 返回内容长度标头值。
//
// 值可能为负数
// -1 意为 Transfer-Encoding: chunked，即传输编码被设置为分块传输。
func (h *RequestHeader) ContentLength() int {
	return h.contentLength
}

// ContentLengthBytes 返回内容长度的字节切片形式。
func (h *RequestHeader) ContentLengthBytes() []byte {
	return h.contentLengthBytes
}

// ContentType 返回内容类型标头值。
func (h *RequestHeader) ContentType() []byte {
	return h.contentType
}

// Cookie 返回指定键的 cookie。
func (h *RequestHeader) Cookie(key string) []byte {
	h.collectCookies()
	return peekArgStr(h.cookies, key)
}

// Cookies 返回全部请求 cookies。
//
// 事后调用 protocol.ReleaseCookie 可有效减少 GC 负载。
func (h *RequestHeader) Cookies() []*Cookie {
	var cookies []*Cookie
	h.VisitAllCookie(func(key, value []byte) {
		cookie := AcquireCookie()
		cookie.SetKeyBytes(key)
		cookie.SetValueBytes(value)
		cookies = append(cookies, cookie)
	})
	return cookies
}

// CopyTo 拷贝所有标头至 dst。
func (h *RequestHeader) CopyTo(dst *RequestHeader) {
	dst.Reset()

	dst.disableNormalizing = h.disableNormalizing
	dst.connectionClose = h.connectionClose
	dst.noDefaultContentType = h.noDefaultContentType

	dst.contentLength = h.contentLength
	dst.contentLengthBytes = append(dst.contentLengthBytes[:0], h.contentLengthBytes...)
	dst.method = append(dst.method[:0], h.method...)
	dst.requestURI = append(dst.requestURI[:0], h.requestURI...)
	dst.host = append(dst.host[:0], h.host...)
	dst.contentType = append(dst.contentType[:0], h.contentType...)
	dst.userAgent = append(dst.userAgent[:0], h.userAgent...)
	h.Trailer().CopyTo(dst.Trailer())
	dst.h = copyArgs(dst.h, h.h)
	dst.cookies = copyArgs(dst.cookies, h.cookies)
	dst.cookiesCollected = h.cookiesCollected
	dst.rawHeaders = append(dst.rawHeaders[:0], h.rawHeaders...)
	dst.protocol = h.protocol
}

// DelAllCookies 删除请求头中的所有 cookies。
func (h *RequestHeader) DelAllCookies() {
	h.collectCookies()
	h.cookies = h.cookies[:0]
}

// DelBytes 删除指定键 key 对应的标头。
func (h *RequestHeader) DelBytes(key []byte) {
	h.bufKV.key = append(h.bufKV.key[:0], key...)
	utils.NormalizeHeaderKey(h.bufKV.key, h.disableNormalizing)
	h.del(h.bufKV.key)
}

// 删除指定键 key 对应的标头。
func (h *RequestHeader) del(key []byte) {
	switch string(key) {
	case consts.HeaderHost:
		h.host = h.host[:0]
	case consts.HeaderContentType:
		h.contentType = h.contentType[:0]
	case consts.HeaderUserAgent:
		h.userAgent = h.userAgent[:0]
	case consts.HeaderCookie:
		h.cookies = h.cookies[:0]
	case consts.HeaderContentLength:
		h.contentLength = 0
		h.contentLengthBytes = h.contentLengthBytes[:0]
	case consts.HeaderConnection:
		h.connectionClose = false
	case consts.HeaderTrailer:
		h.Trailer().ResetSkipNormalize()
	}
	h.h = delAllArgsBytes(h.h, key)
}

// DelCookie 删除指定键 key 对应的 cookie。
func (h *RequestHeader) DelCookie(key string) {
	h.collectCookies()
	h.cookies = delAllArgs(h.cookies, key)
}

// DisableNormalizing 禁用标头名称的规范化。
//
// 默认情况下，第一个字母和所有破折号后面的第一个字母会大写，其他字母小写。
// 例如：
//
//   - CONNECTION -> Connection
//   - conteNT-tYPE -> Content-Type
//   - foo-bar-baz -> Foo-Bar-Baz
//
// 如非必要，不要禁用标头名称的规范化。
func (h *RequestHeader) DisableNormalizing() {
	h.disableNormalizing = true
}

// FullCookie 返回完整的 cookie 字节切片。
func (h *RequestHeader) FullCookie() []byte {
	return h.Peek(consts.HeaderCookie)
}

// Get 返回指定 key 的标头值字符串。
//
// 返回的值在下一次调用 RequestHeader 之前一直有效。
// 不要存储对返回值的引用，可以拷贝副本。
func (h *RequestHeader) Get(key string) string {
	return string(h.Peek(key))
}

// GetAll 返回指定 key 的所有标头值的字符串切片。
// 所以该方法也是并发安全且长期可用的。
func (h *RequestHeader) GetAll(key string) []string {
	res := make([]string, 0)
	headers := h.PeekAll(key)
	for _, header := range headers {
		res = append(res, string(header))
	}
	return res
}

// GetBufValue 获取缓冲键值对的值切片。
func (h *RequestHeader) GetBufValue() []byte {
	return h.bufKV.value
}

// GetProtocol 获取请求协议。
func (h *RequestHeader) GetProtocol() string {
	return h.protocol
}

// HasAcceptEncodingBytes 返回标头是否包含指定的 Accept-Encoding 可接受编码值。
func (h *RequestHeader) HasAcceptEncodingBytes(acceptEncoding []byte) bool {
	ae := h.peek(bytestr.StrAcceptEncoding)
	n := bytes.Index(ae, acceptEncoding)
	if n < 0 {
		return false
	}
	b := ae[n+len(acceptEncoding):]
	if len(b) > 0 && b[0] != ',' {
		return false
	}
	if n == 0 {
		return true
	}
	return ae[n-1] == ' '
}

// Header 返回请求头的字节切片形式。
//
// 返回的表示形式在下一次调用 RequestHeader 方法之前一直有效。
func (h *RequestHeader) Header() []byte {
	h.bufKV.value = h.AppendBytes(h.bufKV.value[:0])
	return h.bufKV.value
}

// Host 返回主机标头值的字节切片。
func (h *RequestHeader) Host() []byte {
	return h.host
}

// IgnoreBody 返回请求方法是否为 GET 或 HEAD，若是则忽略 Body 部分。
func (h *RequestHeader) IgnoreBody() bool {
	return h.IsGet() || h.IsHead()
}

// InitBufValue 按 size 初始化缓冲键值对的值。
func (h *RequestHeader) InitBufValue(size int) {
	if size > cap(h.bufKV.value) {
		h.bufKV.value = make([]byte, 0, size)
	}
}

// InitContentLengthWithValue 按指定内容长度初始化 contentLength。
func (h *RequestHeader) InitContentLengthWithValue(contentLength int) {
	h.contentLength = contentLength
}

// IsDisableNormalizing 返回是否禁用了标头名称的规范化。
func (h *RequestHeader) IsDisableNormalizing() bool {
	return h.disableNormalizing
}

// IsHTTP11 返回是否为 HTTP/1.1 请求。
func (h *RequestHeader) IsHTTP11() bool {
	return h.protocol == consts.HTTP11
}

// IsGet 返回请求方法是否为 GET。
func (h *RequestHeader) IsGet() bool {
	return bytes.Equal(h.Method(), bytestr.StrGet)
}

// IsPost 返回请求方法是否为 POST。
func (h *RequestHeader) IsPost() bool {
	return bytes.Equal(h.Method(), bytestr.StrPost)
}

// IsOptions 返回请求方法是否为 OPTIONS。
func (h *RequestHeader) IsOptions() bool {
	return bytes.Equal(h.Method(), bytestr.StrOptions)
}

// IsTrace 返回请求方法是否为 TRACE。
func (h *RequestHeader) IsTrace() bool {
	return bytes.Equal(h.Method(), bytestr.StrTrace)
}

// IsPut 返回请求方法是否为 PUT。
func (h *RequestHeader) IsPut() bool {
	return bytes.Equal(h.Method(), bytestr.StrPut)
}

// IsHead 返回请求方法是否为 HEAD。
func (h *RequestHeader) IsHead() bool {
	return bytes.Equal(h.Method(), bytestr.StrHead)
}

// IsDelete 返回请求方法是否为 DELETE。
func (h *RequestHeader) IsDelete() bool {
	return bytes.Equal(h.Method(), bytestr.StrDelete)
}

// IsConnect 返回请求方法是否为 CONNECT。
func (h *RequestHeader) IsConnect() bool {
	return bytes.Equal(h.Method(), bytestr.StrConnect)
}

// Len 返回设置的标头数量。
// 即在 VisitAll 中 f 的调用次数。
func (h *RequestHeader) Len() int {
	n := 0
	h.VisitAll(func(k, v []byte) { n++ })
	return n
}

// Method 返回 HTTP 请求方法。默认为 GET。
func (h *RequestHeader) Method() []byte {
	if len(h.method) == 0 {
		return bytestr.StrGet
	}
	return h.method
}

// MultipartFormBoundary 从 Content-Type 中获取 'multipart/form-data; boundary=...' 后的边界部分。
func (h *RequestHeader) MultipartFormBoundary() []byte {
	b := h.ContentType()
	if !bytes.HasPrefix(b, bytestr.StrMultipartFormData) {
		return nil
	}
	b = b[len(bytestr.StrMultipartFormData):]
	if len(b) == 0 || b[0] != ';' {
		return nil
	}

	var n int
	for len(b) > 0 {
		n++
		for len(b) > n && b[n] == ' ' {
			n++
		}
		b = b[n:]
		if !bytes.HasPrefix(b, bytestr.StrBoundary) {
			if n = bytes.IndexByte(b, ';'); n < 0 {
				return nil
			}
			continue
		}

		b = b[len(bytestr.StrBoundary):]
		if len(b) == 0 || b[0] != '=' {
			return nil
		}
		b = b[1:]
		if n = bytes.IndexByte(b, ';'); n >= 0 {
			b = b[:n]
		}
		if len(b) > 1 && b[0] == '"' && b[len(b)-1] == '"' {
			b = b[1 : len(b)-1]
		}
		return b
	}
	return nil
}

// Peek 返回指定 key 按需规格化后的标头值字节切片。
//
// 返回的值在下一次调用 RequestHeader 之前一直有效。
// 不要存储对返回值的引用，可以拷贝副本。
func (h *RequestHeader) Peek(key string) []byte {
	k := getHeaderKeyBytes(&h.bufKV, key, h.disableNormalizing)
	return h.peek(k)
}

// 返回指定 key （不考虑规格化）对应的标头值字节切片。
// 同时对常用几个 key 提前判断并返回。
func (h *RequestHeader) peek(key []byte) []byte {
	switch string(key) {
	case consts.HeaderHost:
		return h.Host()
	case consts.HeaderContentType:
		return h.ContentType()
	case consts.HeaderUserAgent:
		return h.UserAgent()
	case consts.HeaderConnection:
		if h.ConnectionClose() {
			return bytestr.StrClose
		}
		return peekArgBytes(h.h, key)
	case consts.HeaderContentLength:
		return h.contentLengthBytes
	case consts.HeaderCookie:
		if h.cookiesCollected {
			return appendRequestCookieBytes(nil, h.cookies)
		}
		return peekArgBytes(h.h, key)
	case consts.HeaderTrailer:
		return h.Trailer().GetBytes()
	default:
		return peekArgBytes(h.h, key)
	}
}

// PeekAll 返回指定 key 按需规格化后的所有标头值切片。
//
// 返回值在 ReleaseRequest 之前一直有效，且可修改。
//
// 不要存储对返回值的引用，请改用 RequestHeader.GetAll(key)。
func (h *RequestHeader) PeekAll(key string) [][]byte {
	k := getHeaderKeyBytes(&h.bufKV, key, h.disableNormalizing)
	return h.peekAll(k)
}

func (h *RequestHeader) peekAll(key []byte) [][]byte {
	h.mulHeader = h.mulHeader[:0]
	switch string(key) {
	case consts.HeaderHost:
		if host := h.Host(); len(host) > 0 {
			h.mulHeader = append(h.mulHeader, host)
		}
	case consts.HeaderContentType:
		if contentType := h.ContentType(); len(contentType) > 0 {
			h.mulHeader = append(h.mulHeader, contentType)
		}
	case consts.HeaderUserAgent:
		if ua := h.UserAgent(); len(ua) > 0 {
			h.mulHeader = append(h.mulHeader, ua)
		}
	case consts.HeaderConnection:
		if h.ConnectionClose() {
			h.mulHeader = append(h.mulHeader, bytestr.StrClose)
		} else {
			h.mulHeader = peekAllArgBytesToDst(h.mulHeader, h.h, key)
		}
	case consts.HeaderContentLength:
		h.mulHeader = append(h.mulHeader, h.contentLengthBytes)
	case consts.HeaderCookie:
		if h.cookiesCollected {
			h.mulHeader = append(h.mulHeader, appendRequestCookieBytes(nil, h.cookies))
		} else {
			h.mulHeader = peekAllArgBytesToDst(h.mulHeader, h.h, key)
		}
	default:
		h.mulHeader = peekAllArgBytesToDst(h.mulHeader, h.h, key)
	}
	return h.mulHeader
}

// PeekArgBytes 返回指定 key （不考虑规格化）对应的标头值字节切片。
func (h *RequestHeader) PeekArgBytes(key []byte) []byte {
	return peekArgBytes(h.h, key)
}

func (h *RequestHeader) PeekContentEncoding() []byte {
	return h.peek(bytestr.StrContentEncoding)
}

func (h *RequestHeader) PeekIfModifiedSinceBytes() []byte {
	return h.peek(bytestr.StrIfModifiedSince)
}

func (h *RequestHeader) PeekRange() []byte {
	return h.peek(bytestr.StrRange)
}

// RawHeaders 返回原始标头键值对字节切片。
//
// 取决于服务端配置，标头键名称可能会被规范化为大写。
//
// 原始标头和请求行类似，这些副本在解析过程中不存储也不能返回。
//
// 在返回后使用切片是不安全的。
func (h *RequestHeader) RawHeaders() []byte {
	return h.rawHeaders
}

// RequestURI 返回 HTTP 的 RequestURI。
func (h *RequestHeader) RequestURI() []byte {
	requestURI := h.requestURI
	if len(requestURI) == 0 {
		requestURI = bytestr.StrSlashSlash
	}
	return requestURI
}

// Reset 清除请求标头。
func (h *RequestHeader) Reset() {
	h.disableNormalizing = false
	h.Trailer().disableNormalizing = false
	h.ResetSkipNormalize()
}

// ResetConnectionClose 如果标头中有 'Connection: close' 则进行清除。
func (h *RequestHeader) ResetConnectionClose() {
	if h.connectionClose {
		h.connectionClose = false
		h.h = delAllArgsBytes(h.h, bytestr.StrConnection)
	}
}

// ResetSkipNormalize 清除请求标头，但不重置 disableNormalizing 字段。
func (h *RequestHeader) ResetSkipNormalize() {
	h.connectionClose = false
	h.protocol = ""
	h.noDefaultContentType = false

	h.contentLength = 0
	h.contentLengthBytes = h.contentLengthBytes[:0]

	h.method = h.method[:0]
	h.requestURI = h.requestURI[:0]
	h.host = h.host[:0]
	h.contentType = h.contentType[:0]
	h.userAgent = h.userAgent[:0]

	h.h = h.h[:0]
	h.cookies = h.cookies[:0]
	h.cookiesCollected = false

	h.rawHeaders = h.rawHeaders[:0]
	h.mulHeader = h.mulHeader[:0]
	h.Trailer().ResetSkipNormalize()
}

// Set 设置给定的 'key: value' 标头。
//
// 使用 Add 给同一个标头键设置多个值。
func (h *RequestHeader) Set(key, value string) {
	initHeaderKV(&h.bufKV, key, value, h.disableNormalizing)
	h.SetCanonical(h.bufKV.key, h.bufKV.value)
}

// SetArgBytes 设置给定的 'key: value' 标头。
func (h *RequestHeader) SetArgBytes(key, value []byte, noValue bool) {
	h.h = setArgBytes(h.h, key, value, noValue)
}

// SetByteRange 设置 'Range: bytes=startPos-endPos' 标头。
//
//   - 若 startPos 为负值，则值设为 'bytes=-startPos'
//   - 若 endPos 为负值，则值设为 'bytes=startPos-'
func (h *RequestHeader) SetByteRange(startPos, endPos int) {
	b := h.bufKV.value[:0]
	b = append(b, bytestr.StrBytes...)
	b = append(b, '=')
	if startPos >= 0 {
		b = bytesconv.AppendUint(b, startPos)
	} else {
		endPos = -startPos
	}
	b = append(b, '-')
	if endPos >= 0 {
		b = bytesconv.AppendUint(b, endPos)
	}
	h.bufKV.value = b

	h.SetCanonical(bytestr.StrRange, h.bufKV.value)
}

// SetBytesKV 设置指定字节切片类型的 'key: value' 标头。
//
// 使用 AddBytesKV 为相同标头键设置多个值。
func (h *RequestHeader) SetBytesKV(key, value []byte) {
	h.bufKV.key = append(h.bufKV.key[:0], key...)
	utils.NormalizeHeaderKey(h.bufKV.key, h.disableNormalizing)
	h.SetCanonical(h.bufKV.key, value)
}

// SetCanonical 设置指定的 'key: value' 标头，假定该键为规范形式。
func (h *RequestHeader) SetCanonical(key, value []byte) {
	if h.setSpecialHeader(key, value) {
		return
	}

	h.h = setArgBytes(h.h, key, value, ArgsHasValue)
}

// SetConnectionClose 设置连接的关闭状态。
func (h *RequestHeader) SetConnectionClose(close bool) {
	h.connectionClose = close
}

// SetContentLength 根据设置内容长度整数值设置内容长度字节切片。
//
// 若值为负数，会同时设置 'Transfer-Encoding: chunked' 标头。
func (h *RequestHeader) SetContentLength(contentLength int) {
	h.contentLength = contentLength
	if contentLength >= 0 {
		h.contentLengthBytes = bytesconv.AppendUint(h.contentLengthBytes[:0], contentLength)
		h.h = delAllArgsBytes(h.h, bytestr.StrTransferEncoding)
	} else {
		h.contentLengthBytes = h.contentLengthBytes[:0]
		h.h = setArgBytes(h.h, bytestr.StrTransferEncoding, bytestr.StrChunked, ArgsHasValue)
	}
}

// SetContentLengthBytes 直接设置内容长度标头字节切片。
func (h *RequestHeader) SetContentLengthBytes(contentLength []byte) {
	h.contentLengthBytes = append(h.contentLengthBytes[:0], contentLength...)
}

// SetContentTypeBytes 设置 Content-Type 标头值字节切片。
func (h *RequestHeader) SetContentTypeBytes(contentType []byte) {
	h.contentType = append(h.contentType[:0], contentType...)
}

// SetCookie 设置 'key: value' cookies。
func (h *RequestHeader) SetCookie(key, value string) {
	h.collectCookies()
	h.cookies = setArg(h.cookies, key, value, ArgsHasValue)
}

// SetHost 设置主机标头值。
func (h *RequestHeader) SetHost(host string) {
	h.host = append(h.host[:0], host...)
}

// SetHostBytes 设置主机标头值。
func (h *RequestHeader) SetHostBytes(host []byte) {
	h.host = append(h.host[:0], host...)
}

// SetMethod 设置 HTTP 请求方法。
func (h *RequestHeader) SetMethod(method string) {
	h.method = append(h.method[:0], method...)
}

// SetMethodBytes 设置 HTTP 请求方法。
func (h *RequestHeader) SetMethodBytes(method []byte) {
	h.method = append(h.method[:0], method...)
}

// SetMultipartFormBoundary 设置 MultipartFormData 后的边界值。
func (h *RequestHeader) SetMultipartFormBoundary(boundary string) {

	b := h.bufKV.value[:0]
	b = append(b, bytestr.StrMultipartFormData...)
	b = append(b, ';', ' ')
	b = append(b, bytestr.StrBoundary...)
	b = append(b, '=')
	b = append(b, boundary...)
	h.bufKV.value = b

	h.SetContentTypeBytes(h.bufKV.value)
}

// SetNoDefaultContentType 控制默认内容类型的标头行为。
//
// 若设为 false，则在未指定内容类型时，使用默认内容类型标头。
// 若设为 true，则在未指定内容类型时，不发送内容类型标头。
func (h *RequestHeader) SetNoDefaultContentType(b bool) {
	h.noDefaultContentType = b
}

// SetProtocol 设置请求协议。
func (h *RequestHeader) SetProtocol(p string) {
	h.protocol = p
}

// SetRawHeaders 设置原始标头字节切片。
func (h *RequestHeader) SetRawHeaders(r []byte) {
	h.rawHeaders = r
}

// SetRequestURI 设置第一个 HTTP 请求行的 RequestURI。
// RequestURI 必须正确编码。
// 若不确定，请使用 URI.RequestURI 构造正确的 RequestURI。
func (h *RequestHeader) SetRequestURI(requestURI string) {
	h.requestURI = append(h.requestURI[:0], requestURI...)
}

// SetRequestURIBytes 设置第一个 HTTP 请求行的 RequestURI。
// RequestURI 必须正确编码。
// 若不确定，请使用 URI.RequestURI 构造正确的 RequestURI。
func (h *RequestHeader) SetRequestURIBytes(requestURI []byte) {
	h.requestURI = append(h.requestURI[:0], requestURI...)
}

// 处理特殊标头，若已处理则返回 true。
func (h *RequestHeader) setSpecialHeader(key, value []byte) bool {
	if len(key) == 0 {
		return false
	}

	switch key[0] | 0x20 {
	case 'c': // ContentType, ContentLength, Connection, Cookie
		if utils.CaseInsensitiveCompare(bytestr.StrContentType, key) {
			h.SetContentTypeBytes(value)
			return true
		} else if utils.CaseInsensitiveCompare(bytestr.StrContentLength, key) {
			if contentLength, err := ParseContentLength(value); err == nil {
				h.contentLength = contentLength
				h.contentLengthBytes = append(h.contentLengthBytes[:0], value...)
			}
			return true
		} else if utils.CaseInsensitiveCompare(bytestr.StrConnection, key) {
			if bytes.Equal(bytestr.StrClose, value) {
				h.SetConnectionClose(true)
			} else {
				h.ResetConnectionClose()
				h.h = setArgBytes(h.h, key, value, ArgsHasValue)
			}
			return true
		} else if utils.CaseInsensitiveCompare(bytestr.StrCookie, key) {
			h.collectCookies()
			h.cookies = parseRequestCookies(h.cookies, value)
			return true
		}
	case 't': // TransferEncoding, Trailer
		if utils.CaseInsensitiveCompare(bytestr.StrTransferEncoding, key) {
			// 传输编码是自动管理的。
			return true
		} else if utils.CaseInsensitiveCompare(bytestr.StrTrailer, key) {
			// 拷贝值以免恐慌
			value = append(h.bufKV.value[:0], value...)
			h.Trailer().SetTrailers(value)
			return true
		}
	case 'h': // Host
		if utils.CaseInsensitiveCompare(bytestr.StrHost, key) {
			h.SetHostBytes(value)
			return true
		}
	case 'u': // User-Agent
		if utils.CaseInsensitiveCompare(bytestr.StrUserAgent, key) {
			h.SetUserAgentBytes(value)
			return true
		}
	}

	return false
}

// SetUserAgent 设置用户代理标头值。
func (h *RequestHeader) SetUserAgent(userAgent string) {
	h.userAgent = append(h.userAgent[:0], userAgent...)
}

// SetUserAgentBytes 设置用户代理标头值。
func (h *RequestHeader) SetUserAgentBytes(userAgent []byte) {
	h.userAgent = append(h.userAgent[:0], userAgent...)
}

// String 返回请求头的字符串形式。
func (h *RequestHeader) String() string {
	return string(h.Header())
}

// Trailer 返回 HTTP 标头的挂车。
func (h *RequestHeader) Trailer() *Trailer {
	if h.trailer == nil {
		h.trailer = new(Trailer)
	}
	return h.trailer
}

// UserAgent 返回用户代理标头值。
func (h *RequestHeader) UserAgent() []byte {
	return h.userAgent
}

// VisitAll 对每个标头应用函数 f。
//
// f 在返回后不得保留对键或值的引用，以防数据竞赛。
// 如果需要保留密钥和/或值内容，请在返回之前复制它们。
func (h *RequestHeader) VisitAll(f func(key, value []byte)) {
	host := h.Host()
	if len(host) > 0 {
		f(bytestr.StrHost, host)
	}
	if len(h.contentLengthBytes) > 0 {
		f(bytestr.StrContentLength, h.contentLengthBytes)
	}
	contentType := h.ContentType()
	if len(contentType) > 0 {
		f(bytestr.StrContentType, contentType)
	}
	userAgent := h.UserAgent()
	if len(userAgent) > 0 {
		f(bytestr.StrUserAgent, userAgent)
	}
	if !h.Trailer().Empty() {
		f(bytestr.StrTrailer, h.Trailer().GetBytes())
	}

	h.collectCookies()
	if len(h.cookies) > 0 {
		h.bufKV.value = appendRequestCookieBytes(h.bufKV.value[:0], h.cookies)
		f(bytestr.StrCookie, h.bufKV.value)
	}
	visitArgs(h.h, f)
	if h.ConnectionClose() {
		f(bytestr.StrConnection, bytestr.StrClose)
	}
}

// VisitAllCookie 对每个请求 cookie 应用函数 f。
//
// f 在返回后不得保留对键或值的引用，以防数据竞赛。
func (h *RequestHeader) VisitAllCookie(f func(key, value []byte)) {
	h.collectCookies()
	visitArgs(h.cookies, f)
}

func parseRequestCookies(cookies []argsKV, src []byte) []argsKV {
	var s cookieScanner
	s.b = src
	var kv *argsKV
	cookies, kv = allocArg(cookies)
	for s.next(kv) {
		if len(kv.key) > 0 || len(kv.value) > 0 {
			cookies, kv = allocArg(cookies)
		}
	}
	return releaseArg(cookies)
}

// 将 key 加入 kv 后按需规格化并返回。
func getHeaderKeyBytes(kv *argsKV, key string, disableNormalizing bool) []byte {
	kv.key = append(kv.key[:0], key...)
	utils.NormalizeHeaderKey(kv.key, disableNormalizing)
	return kv.key
}

// 初始化一个指定 key 和 value 的标头键值对 kv。
func initHeaderKV(kv *argsKV, key, value string, disableNormalizing bool) {
	kv.key = getHeaderKeyBytes(kv, key, disableNormalizing)
	kv.value = append(kv.value[:0], value...)
}

// 附加一个标头行。
// 形如 "key: value;\r\n"
func appendHeaderLine(dst, key, value []byte) []byte {
	dst = append(dst, key...)
	dst = append(dst, bytestr.StrColonSpace...)
	dst = append(dst, value...)
	return append(dst, bytestr.StrCRLF...)
}

func ParseContentLength(b []byte) (int, error) {
	v, n, err := bytesconv.ParseUintBuf(b)
	if err != nil {
		return -1, err
	}
	if n != len(b) {
		return -1, errs.NewPublic("Content-Length末尾出现非数字字符")
	}
	return v, nil
}

// ResponseHeader 表示 HTTP 响应头。
//
// 禁止复制 ResponseHeader 实例，而是通过创建新实例并 CopyTo 来替代。
//
// ResponseHeader 实例 *不能* 用于多协程，并发不是安全的。
type ResponseHeader struct {
	noCopy nocopy.NoCopy

	disableNormalizing   bool // 禁用标头名称规范化？
	connectionClose      bool // 连接已关闭？
	noDefaultContentType bool // 不实用默认内容类型？
	noDefaultDate        bool // 不用默认日期？

	statusCode         int
	contentLength      int
	contentLengthBytes []byte
	contentEncoding    []byte

	contentType []byte
	server      []byte
	mulHeader   [][]byte
	protocol    string

	h       []argsKV
	bufKV   argsKV
	trailer *Trailer

	cookies []argsKV

	headerLength int
}

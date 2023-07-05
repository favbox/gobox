package consts

const (
	HeaderDate = "Date"

	HeaderIfModifiedSince = "If-Modified-Since"
	HeaderLastModified    = "Last-Modified"

	HeaderLocation = "Location" // 重定向
)

// 传输编码类
const (
	HeaderTE               = "TE"
	HeaderTrailer          = "Trailer"
	HeaderTrailerLower     = "trailer"
	HeaderTransferEncoding = "Transfer-Encoding"
)

// 控制类
const (
	HeaderCookie         = "Cookie"
	HeaderExpect         = "Expect"
	HeaderMaxForwards    = "Max-Forwards"
	HeaderSetCookie      = "Set-Cookie"
	HeaderSetCookieLower = "set-cookie"
)

// 连接管理类
const (
	HeaderConnection      = "Connection"
	headerKeepAlive       = "Keep-Alive"
	HeaderProxyConnection = "Proxy-Connection"
)

// 鉴权类
const (
	HeaderAuthorization      = "Authorization"
	HeaderProxyAuthenticate  = "Proxy-Authenticate"
	HeaderProxyAuthorization = "Proxy-Authorization"
	HeaderWWWAuthenticate    = "WWW-Authenticate"
)

// 区间请求类
const (
	HeaderAcceptRanges = "Accept-Ranges"
	HeaderContentRange = "Content-Range"
	HeaderIfRange      = "If-Range"
	HeaderRange        = "Range"
)

// 响应上下文类
const (
	HeaderAllow       = "Allow"
	HeaderServer      = "Server"
	HeaderServerLower = "server"
)

// 请求上下文
const (
	HeaderFrom          = "From"
	HeaderHost          = "Host"
	HeaderReferer       = "Referer"
	HeaderRefererPolicy = "Referer-Policy"
	HeaderUserAgent     = "User-Agent"
)

// 消息体信息
const (
	HeaderContentEncoding = "Content-Encoding"
	HeaderContentLanguage = "Content-Language"
	HeaderContentLength   = "Content-Length"
	HeaderContentLocation = "Content-Location"
	HeaderContentType     = "Content-Type"
)

// 内容协商类
const (
	HeaderAccept         = "Accept"
	HeaderAcceptCharset  = "Accept-Charset"
	HeaderAcceptEncoding = "Accept-Encoding"
	HeaderAcceptLanguage = "Accept-Language"
	HeaderAltSvc         = "Alt-Svc"
)

// 协议类
const (
	HTTP11 = "HTTP/1.1"
	HTTP10 = "HTTP/1.0"
	HTTP20 = "HTTP/2.0"
)

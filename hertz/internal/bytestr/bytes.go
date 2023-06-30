// Package bytestr 定义一些常用字节化字符串。
package bytestr

var (
	DefaultServerName  = []byte("hertz")
	DefaultUserAgent   = []byte("hertz")
	DefaultContentType = []byte("text/plain; charset=utf-8")
)

var (
	StrBackSlash        = []byte("\\")
	StrSlash            = []byte("/")
	StrSlashSlash       = []byte("//")
	StrSlashDotDot      = []byte("/..")
	StrSlashDotSlash    = []byte("/./")
	StrSlashDotDotSlash = []byte("/../")
	StrCRLF             = []byte("\r\n")
	StrHTTP             = []byte("http")
	StrHTTP11           = []byte("HTTP/1.1")
	StrColon            = []byte(":")
	StrStar             = []byte("*")
)

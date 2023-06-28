// runtime 包使用 //go:linkname 将函数推送到该包中。但当前依然需要一个 .s 文件，
// 因为 -complete 标志不允许没有主体的函数声明。
// 详见 https://github.com/golang/go/issues/23311
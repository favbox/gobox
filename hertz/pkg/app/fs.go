package app

import "github.com/favbox/gobox/hertz/internal/nocopy"

// FS 表示为本地文件系统中静态文件提供服务的请求处理程序的设置。
//
// 禁止复制 FS 值，而是创建新值。
type FS struct {
	noCopy nocopy.NoCopy

	// 文件根目录路径。
	Root string

	// 目录下索引文件名的列表。
	//
	// 例如：
	//
	//	* index.html
	//	* login.html
	//
	// 默认该列表为空。
	IndexNames []string
}

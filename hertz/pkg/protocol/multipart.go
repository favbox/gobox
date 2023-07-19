package protocol

import (
	"fmt"
	"io"
	"mime/multipart"

	"github.com/favbox/gobox/hertz/pkg/common/bytebufferpool"
	"github.com/favbox/gobox/hertz/pkg/common/utils"
	"github.com/favbox/gobox/hertz/pkg/network"
)

// MarshalMultipartForm 将多部分表单编排为字节切片。
func MarshalMultipartForm(f *multipart.Form, boundary string) ([]byte, error) {
	var buf bytebufferpool.ByteBuffer
	if err := WriteMultipartForm(&buf, f, boundary); err != nil {
		return nil, err
	}
	return buf.B, nil
}

// WriteMultipartForm 将指定的多部分表单 f 和边界值 boundary 写入 w。
func WriteMultipartForm(w io.Writer, f *multipart.Form, boundary string) error {
	// 这里不关心内存分配，因为多部分表单处理很慢。
	if len(boundary) == 0 {
		panic("BUG: 表单边界值 boundary 不能为空")
	}

	mw := multipart.NewWriter(w)
	if err := mw.SetBoundary(boundary); err != nil {
		return fmt.Errorf("无法使用表单边界值 %q: %s", boundary, err)
	}

	// 编排值
	for k, vv := range f.Value {
		for _, v := range vv {
			if err := mw.WriteField(k, v); err != nil {
				return fmt.Errorf("无法写入表单字段 %q 值 %q: %s", k, v, err)
			}
		}
	}

	// 编排文件
	for k, fvv := range f.File {
		for _, fv := range fvv {
			vw, err := mw.CreatePart(fv.Header)
			zw := network.NewWriter(vw)
			if err != nil {
				return fmt.Errorf("无法创建表单文件 %q (%q): %s", k, fv.Filename, err)
			}
			fh, err := fv.Open()
			if err != nil {
				return fmt.Errorf("无法打开表单文件 %q (%q): %s", k, fv.Filename, err)
			}
			if _, err = utils.CopyZeroAlloc(zw, fh); err != nil {
				return fmt.Errorf("拷贝表单文件 %q (%q): %s 发生错误", k, fv.Filename, err)
			}
			if err = fh.Close(); err != nil {
				return fmt.Errorf("无法关闭表单文件 %q (%q): %s", k, fv.Filename, err)
			}
		}
	}

	if err := mw.Close(); err != nil {
		return fmt.Errorf("关闭表单编写器 %s 出现错误", err)
	}

	return nil
}

func ReadMultipartForm(r io.Reader, boundary string, size, maxInMemoryFileSize int) (*multipart.Form, error) {
	// 不用关心此处的内存分派，因为与多部分表单发送的数据（通常几MB）相比，以下内存分配很小。

	if size <= 0 {
		return nil, fmt.Errorf("表单大小必须大于0。给定 %d", size)
	}
	lr := io.LimitReader(r, int64(size))
	mr := multipart.NewReader(lr, boundary)
	f, err := mr.ReadForm(int64(maxInMemoryFileSize))
	if err != nil {
		return nil, fmt.Errorf("无法读取多部分表单数据体: %s", err)
	}
	return f, nil
}

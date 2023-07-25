package ext

import (
	"bytes"
	"io"
	"sync"

	errs "github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/utils"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol"
)

var (
	errChunkedStream = errs.New(errs.ErrChunkedStream, errs.ErrorTypePublic, nil)

	bodyStreamPool = sync.Pool{
		New: func() any {
			return &bodyStream{}
		},
	}
)

type bodyStream struct {
	prefetchedBytes *bytes.Reader
	reader          network.Reader
	trailer         *protocol.Trailer
	offset          int
	contentLength   int
	chunkLeft       int
	chunkEOF        bool // 块是否已触底
}

func (bs *bodyStream) Read(p []byte) (int, error) {
	defer func() {
		if bs.reader != nil {
			bs.reader.Release()
		}
	}()

	if bs.contentLength == -1 {
		if bs.chunkEOF {
			return 0, io.EOF
		}

		if bs.chunkLeft == 0 {
			// TODO
		}
		bytesToRead := len(p)

		if bytesToRead > bs.chunkLeft {
			bytesToRead = bs.chunkLeft
		}

		src, err := bs.reader.Peek(bytesToRead)
		copied := copy(p, src)
		bs.reader.Skip(copied)
		bs.chunkLeft -= copied

		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return copied, err
		}

		if bs.chunkLeft == 0 {
			err = utils.SkipCRLF(bs.reader)
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
		}

		return copied, err
	}

	if bs.offset == bs.contentLength {
		return 0, io.EOF
	}

	var n int
	var err error
	// 从预读缓冲区读取
	if int(bs.prefetchedBytes.Size()) > bs.offset {
		n, err = bs.prefetchedBytes.Read(p)
		bs.offset += n
		if bs.offset == bs.contentLength {
			return n, io.EOF
		}
		if err != nil || len(p) == n {
			return n, err
		}
	}

	// 从 wire 读取
	m := len(p) - n
	remain := bs.contentLength - bs.offset
	if m > remain {
		m = remain
	}

	if conn, ok := bs.reader.(io.Reader); ok {
		m, err = conn.Read(p[n:])
	} else {
		var tmp []byte
		tmp, err = bs.reader.Peek(m)
		m = copy(p[n:], tmp)
		bs.reader.Skip(m)
	}
	bs.offset += m
	n += m

	if err != nil {
		// 流数据可能不完整
		if err == io.EOF {
			if bs.offset != bs.contentLength && bs.contentLength != -2 {
				err = io.ErrUnexpectedEOF
			}
			// 确保 skipREset 好使
			bs.offset = bs.contentLength
		}
		return n, err
	}

	if bs.offset == bs.contentLength {
		err = io.EOF
	}
	return n, err
}

// ReadBodyWithStreaming 将网络读取器 r 按指定参数流式读取到 dst 并返回。
func ReadBodyWithStreaming(zr network.Reader, contentLength, maxBodySize int, dst []byte) (b []byte, err error) {
	if contentLength == -1 {
		return b, errChunkedStream
	}
	dst = dst[:0]

	if maxBodySize <= 0 {
		maxBodySize = maxContentLengthInStream
	}

	readN := maxBodySize
	if readN > contentLength {
		readN = contentLength
	}
	if readN > maxContentLengthInStream {
		readN = maxContentLengthInStream
	}

	if contentLength >= 0 && maxBodySize >= contentLength {
		b, err = appendBodyFixedSize(zr, dst, readN)
	} else {
		b, err = readBodyIdentity(zr, readN, dst)
	}

	if err != nil {
		return b, err
	}
	if contentLength > maxBodySize {
		return b, errBodyTooLarge
	}
	return b, nil
}

package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"time"

	"github.com/favbox/gosky/wind/internal/bytestr"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
)

func SetupProxy(conn network.Conn, addr string, proxyURI *protocol.URI, tlsConfig *tls.Config, isTLS bool, dialer network.Dialer) (network.Conn, error) {
	var err error
	if bytes.Equal(proxyURI.Scheme(), bytestr.StrHTTPS) {
		conn, err = dialer.AddTLS(conn, tlsConfig)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case proxyURI == nil:
		// 啥也不干。没用代理。
	case isTLS:
		connectReq, connectResp := protocol.AcquireRequest(), protocol.AcquireResponse()
		defer func() {
			protocol.ReleaseRequest(connectReq)
			protocol.ReleaseResponse(connectResp)
		}()

		SetProxyAuthHeader(&connectReq.Header, proxyURI)
		connectReq.SetMethod(consts.MethodConnect)
		connectReq.SetHost(addr)

		// 发送 CONNECT 请求时，跳过响应体
		connectResp.SkipBody = true

		// 设置超时时长，以免永久阻塞造成协程泄露。
		connectCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		didReadResponse := make(chan struct{}) // 关闭于 CONNECT 请求读写完成或失败之后

		// 写入 CONNECT 请求，并读取响应。
		go func() {
			defer close(didReadResponse)

			reqI
		}()
		select {}
	}

	return conn, nil
}

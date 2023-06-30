package dialer

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/favbox/gobox/hertz/pkg/common/test/assert"
	"github.com/favbox/gobox/hertz/pkg/network"
)

func TestDialer(t *testing.T) {
	defaultDialer = &mockDialer{}
	dialer := DefaultDialer()
	assert.DeepEqual(t, &mockDialer{}, dialer)

	_, err := DialConnection("", "", 0, nil)
	assert.NotNil(t, err)

	_, err = DialTimeout("", "", 0, nil)
	assert.NotNil(t, err)

	_, err = AddTLS(nil, nil)
	assert.NotNil(t, err)
}

type mockDialer struct{}

func (m *mockDialer) DialConnection(network, address string, timeout time.Duration, tlsConfig *tls.Config) (network.Conn, error) {
	return nil, errors.New("方法尚未实现")
}

func (m *mockDialer) DialTimeout(network, address string, timeout time.Duration, tlsConfig *tls.Config) (net.Conn, error) {
	return nil, errors.New("方法尚未实现")
}

func (m *mockDialer) AddTLS(conn network.Conn, tlsConfig *tls.Config) (network.Conn, error) {
	return nil, errors.New("方法尚未实现")
}

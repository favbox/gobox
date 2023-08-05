package client

import (
	"errors"
)

var errTooManyRedirects = errors.New("too many redirects detected when doing the request")

//
//func TestCloseIdleConnections(t *testing.T) {
//	opt := config.NewOptions([]config.Option{})
//	opt.Addr = "unix-test-10000"
//	opt.Network = "unix"
//	engine := route.NewEngine(opt)
//
//	go engine.Run()
//	defer func() {
//		engine.Close()
//	}()
//	time.Sleep(time.Millisecond * 500)
//
//	c, _ := NewClient(WithDialer(newMockDialerWithCustomFunc(opt.Network, opt.Addr, 1*time.Second, nil)))
//
//	if _, _, err := c.Get(context.Background(), nil, "http://google.com"); err != nil {
//		t.Fatal(err)
//	}
//
//	connsLen := func() int {
//		c.mLock.Lock()
//		defer c.mLock.Unlock()
//
//		if _, ok := c.m["google.com"]; !ok {
//			return 0
//		}
//		return c.m["google.com"].ConnectionCount()
//	}
//
//	if conns := connsLen(); conns > 1 {
//		t.Errorf("expected 1 conns got %d", conns)
//	}
//
//	c.CloseIdleConnections()
//
//	if conns := connsLen(); conns > 0 {
//		t.Errorf("expected 0 conns got %d", conns)
//	}
//}

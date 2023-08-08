package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/favbox/gosky/wind/internal/bytestr"
	"github.com/favbox/gosky/wind/pkg/common/test/assert"
	"github.com/favbox/gosky/wind/pkg/common/test/mock"
	"github.com/favbox/gosky/wind/pkg/common/testdata/proto"
	"github.com/favbox/gosky/wind/pkg/common/utils"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
)

func TestRequestContext_ClientIP(t *testing.T) {
	c := NewContext(0)
	c.conn = mock.NewConn("")

	// 0.0.0.0 模拟可信代理服务器
	c.Request.Header.Set("X-Forwarded-For", "126.0.0.2, 0.0.0.0")
	ip := c.ClientIP()
	assert.DeepEqual(t, "126.0.0.2", ip)

	// 无代理服务器
	c = NewContext(0)
	c.conn = mock.NewConn("")
	c.Request.Header.Set("X-Real-Ip", "126.0.0.1")
	ip = c.ClientIP()
	assert.DeepEqual(t, "126.0.0.1", ip)

	// 自定义 RemoteIPHeaders 和 TrustedProxies
	opts := ClientIPOptions{
		RemoteIPHeaders: []string{"X-Forwarded-For", "X-Real-IP"},
		TrustedProxies: map[string]bool{
			"0.0.0.0": true,
		},
	}
	c = NewContext(0)
	c.SetClientIPFunc(ClientIPWithOption(opts))
	c.conn = mock.NewConn("")
	c.Request.Header.Set("X-Forwarded-For", " 126.0.0.2, 0.0.0.0")
	ip = c.ClientIP()
	assert.DeepEqual(t, "126.0.0.2", ip)

	// 无可信的代理服务器
	opts = ClientIPOptions{
		RemoteIPHeaders: []string{"X-Forwarded-For", "X-Real-IP"},
		TrustedProxies:  nil,
	}
	c = NewContext(0)
	c.SetClientIPFunc(ClientIPWithOption(opts))
	c.conn = mock.NewConn("")
	c.Request.Header.Set("X-Forwarded-For", " 126.0.0.2, 0.0.0.0")
	ip = c.ClientIP()
	assert.DeepEqual(t, "0.0.0.0", ip)
}

func TestRequestContext_SetClientIPFunc(t *testing.T) {
	fn := func(ctx *RequestContext) string {
		return ""
	}
	SetClientIPFunc(fn)
	assert.DeepEqual(t, reflect.ValueOf(fn).Pointer(), reflect.ValueOf(defaultClientIP).Pointer())
}

func TestProtobuf(t *testing.T) {
	ctx := NewContext(0)
	body := proto.TestStruct{Body: []byte("Hello World")}
	ctx.ProtoBuf(consts.StatusOK, &body)

	expected := string(ctx.Response.Body())
	assert.DeepEqual(t, expected, "\n\vHello World")
}

func TestRequestContext_PureJSON(t *testing.T) {
	ctx := NewContext(0)
	ctx.PureJSON(consts.StatusOK, utils.H{
		"html": "<b>Hello World</b>",
	})
	assert.DeepEqual(t, "{\"html\":\"<b>Hello World</b>\"}\n", string(ctx.Response.Body()))
}

func TestRequestContext_IndentedJSON(t *testing.T) {
	ctx := NewContext(0)
	ctx.IndentedJSON(consts.StatusOK, utils.H{
		"foo":  "bar",
		"html": "h1",
	})
	actual := string(ctx.Response.Body())
	assert.DeepEqual(t, "{\n    \"foo\": \"bar\",\n    \"html\": \"h1\"\n}", actual)
}

func TestNewContext(t *testing.T) {
	reqContext := NewContext(0)
	reqContext.Set("testContextKey", "testValue")
	ctx := reqContext
	assert.DeepEqual(t, "testValue", ctx.Value("testContextKey"))
}

func TestContextNotModified(t *testing.T) {
	ctx := NewContext(0)
	ctx.Response.SetStatusCode(consts.StatusOK)
	assert.DeepEqual(t, consts.StatusOK, ctx.Response.StatusCode())
	ctx.NotModified()
	assert.DeepEqual(t, consts.StatusNotModified, ctx.Response.StatusCode())
}

func TestRequestContext_IfModifiedSince(t *testing.T) {
	ctx := NewContext(0)
	var req protocol.Request
	req.Header.Set(string(bytestr.StrIfModifiedSince), "Mon, 02 Jan 2006 15:04:05 MST")
	req.CopyTo(&ctx.Request)
	assert.True(t, ctx.IfModifiedSince(time.Now()))
	tt, _ := time.Parse(time.RFC3339, "2004-11-12T11:45:26.371Z")
	assert.False(t, ctx.IfModifiedSince(tt))
}

func TestWrite(t *testing.T) {
	ctx := NewContext(0)
	l, err := ctx.WriteString("test body")
	assert.Nil(t, err)
	assert.DeepEqual(t, 9, l)
	assert.DeepEqual(t, "test body", string(ctx.Response.BodyBytes()))
}

func TestRequestContext_SetConnectionClose(t *testing.T) {
	ctx := NewContext(0)
	ctx.SetConnectionClose()
	assert.True(t, ctx.Response.Header.ConnectionClose())
}

func TestRequestContext_NotModified(t *testing.T) {
	ctx := NewContext(0)
	ctx.NotModified()
	assert.True(t, ctx.Response.StatusCode() == consts.StatusNotModified)
}

func TestRequestContext_NotFound(t *testing.T) {
	ctx := NewContext(0)
	ctx.NotFound()
	assert.True(t, ctx.Response.StatusCode() == consts.StatusNotFound)
	assert.DeepEqual(t, "404 Page not found", string(ctx.Response.BodyBytes()))
}

func TestRequestContext_Redirect(t *testing.T) {
	ctx := NewContext(0)
	ctx.Redirect(consts.StatusFound, []byte("/hello"))
	assert.DeepEqual(t, consts.StatusFound, ctx.Response.StatusCode())

	ctx.redirect([]byte("/hello"), consts.StatusMovedPermanently)
	assert.DeepEqual(t, consts.StatusMovedPermanently, ctx.Response.StatusCode())
}

func TestGetRedirectStatusCode(t *testing.T) {
	val := getRedirectStatusCode(consts.StatusMovedPermanently)
	assert.DeepEqual(t, consts.StatusMovedPermanently, val)

	val = getRedirectStatusCode(consts.StatusNotFound)
	assert.DeepEqual(t, consts.StatusFound, val)
}

func TestCookie(t *testing.T) {
	ctx := NewContext(0)
	ctx.Request.Header.SetCookie("cookie", "test cookie")
	if string(ctx.Cookie("cookie")) != "test cookie" {
		t.Fatalf("unexpected cookie: %#v, expected get: %#v", string(ctx.Cookie("cookie")), "test cookie")
	}
}

func TestUserAgent(t *testing.T) {
	ctx := NewContext(0)
	ctx.Request.Header.SetUserAgentBytes([]byte("user agent"))
	if string(ctx.UserAgent()) != "user agent" {
		t.Fatalf("unexpected user agent: %#v, expected get: %#v", string(ctx.UserAgent()), "user agent")
	}
}

func TestStatus(t *testing.T) {
	ctx := NewContext(0)
	ctx.Status(consts.StatusOK)
	if ctx.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("expected get consts.StatusOK, but not")
	}
}

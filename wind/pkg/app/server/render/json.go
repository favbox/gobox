package render

import (
	"bytes"
	"encoding/json"

	hjson "github.com/favbox/gosky/wind/pkg/common/json"
	"github.com/favbox/gosky/wind/pkg/protocol"
)

// JSONMarshaler 自定义 json.Marshal。
type JSONMarshaler func(v any) ([]byte, error)

var jsonMarshalFunc JSONMarshaler

func init() {
	ResetJSONMarshal(hjson.Marshal)
}

// ResetStdJSONMarshal 重置 JSON 编排函数为标准库实现。
func ResetStdJSONMarshal() {
	ResetJSONMarshal(json.Marshal)
}

// ResetJSONMarshal 重置 JSON 编排函数为给定的 fn。
func ResetJSONMarshal(fn JSONMarshaler) {
	jsonMarshalFunc = fn
}

var jsonContentType = "application/json; charset=utf-8"

// JSONRender 表示默认 JSON 渲染器（无缩进、启用 html 转义）。
type JSONRender struct {
	Data any
}

func (r JSONRender) Render(resp *protocol.Response) error {
	r.WriteContentType(resp)
	jsonBytes, err := jsonMarshalFunc(r.Data)
	if err != nil {
		return err
	}

	resp.AppendBody(jsonBytes)
	return nil
}

func (r JSONRender) WriteContentType(resp *protocol.Response) {
	writeContentType(resp, jsonContentType)
}

// PureJSON 表示纯 JSON 渲染器（无缩进、不启用 html 转义）。
type PureJSON struct {
	Data any
}

func (r PureJSON) Render(resp *protocol.Response) error {
	r.WriteContentType(resp)
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(r.Data)
	if err != nil {
		return err
	}
	resp.AppendBody(buf.Bytes())
	return nil
}

func (r PureJSON) WriteContentType(resp *protocol.Response) {
	writeContentType(resp, jsonContentType)
}

// IndentedJSON 表示带缩进的 JSON 渲染器（缩进 4 个空格、启用 html 转义）。
type IndentedJSON struct {
	Data any
}

func (r IndentedJSON) Render(resp *protocol.Response) error {
	r.WriteContentType(resp)
	jsonBytes, err := jsonMarshalFunc(r.Data)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, jsonBytes, "", "    ")
	if err != nil {
		return err
	}
	resp.AppendBody(buf.Bytes())
	return nil
}

func (r IndentedJSON) WriteContentType(resp *protocol.Response) {
	writeContentType(resp, jsonContentType)
}
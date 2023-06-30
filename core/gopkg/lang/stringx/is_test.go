package stringx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIs(t *testing.T) {
	is := assert.New(t)

	is.False(IsNumeric(""))
	is.False(IsNumeric(" "))
	is.False(IsNumeric(" hi "))
	is.True(IsNumeric("123"))

	is.False(IsAlpha(""))
	is.False(IsAlpha(" "))
	is.False(IsAlpha(" hi "))
	is.False(IsAlpha("123"))
	is.True(IsAlpha("hi"))
	is.True(IsAlpha("馫龘飝鱻灥麤靐飍朤淼馫譶龘"))

	is.False(IsAlphaNumber(""))
	is.False(IsAlphaNumber(" "))
	is.False(IsAlphaNumber(" hi "))
	is.True(IsAlphaNumber("hi"))
	is.True(IsAlphaNumber("123"))
	is.True(IsAlphaNumber("a1b2c3"))
	is.False(IsAlphaNumber("a1b2c3,"))
}

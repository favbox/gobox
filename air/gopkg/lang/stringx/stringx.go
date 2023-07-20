package stringx

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/favbox/gosky/air/gopkg/internal/hack"
	"github.com/favbox/gosky/air/gopkg/lang/fastrand"
)

var ErrDecodeRune = errors.New("符文编码出错")

// PadLeftRune 左侧填充符文至字符串使其长度达到指定大小。
// 注意：尺寸是 unicode 大小。
func PadLeftRune(s string, size int, r rune) string {
	return padRuneLeftOrRight(s, size, r, true)
}

// PadLeftSpace 左侧填充空格至字符串使其长度达到指定大小。
func PadLeftSpace(s string, size int) string {
	return PadLeftRune(s, size, ' ')
}

// PadRightRune 右侧填充符文至字符串使其长度达到指定大小。
// 注意：尺寸是 unicode 大小。
func PadRightRune(s string, size int, r rune) string {
	return padRuneLeftOrRight(s, size, r, false)
}

// PadRightSpace 右侧填充空格至字符串使其长度达到指定大小。
func PadRightSpace(s string, size int) string {
	return PadRightRune(s, size, ' ')
}

// PadCenterRune 填充符文至字符串使其居中显示且长度达到指定大小。
func PadCenterRune(s string, size int, r rune) string {
	if size <= 0 {
		return s
	}
	length := utf8.RuneCountInString(s)
	pads := size - length

	// 长度已满足，无需填充
	if pads <= 0 {
		return s
	}

	// 左侧填充
	leftPads := pads / 2
	if leftPads > 0 {
		s = padRawLeftRune(s, r, leftPads)
	}
	// 右侧填充
	rightPads := size - leftPads - length
	if rightPads > 0 {
		s = padRawRightRune(s, r, rightPads)
	}

	return s
}

// PadCenterSpace 填充空格至字符串使其居中显示且长度达到指定大小。
func PadCenterSpace(s string, size int) string {
	return PadCenterRune(s, size, ' ')
}

func padRuneLeftOrRight(s string, size int, r rune, isLeft bool) string {
	if size <= 0 {
		return s
	}

	// 计算候补数量
	pads := size - utf8.RuneCountInString(s)

	// 长度已满足，无需填充
	if pads <= 0 {
		return s
	}

	if isLeft {
		return padRawLeftRune(s, r, pads)
	}
	return padRawRightRune(s, r, pads)
}

func padRawLeftRune(s string, r rune, padSize int) string {
	return RepeatRune(r, padSize) + s
}

func padRawRightRune(s string, r rune, padSize int) string {
	return s + RepeatRune(r, padSize)
}

// RepeatRune 返回长度为n的重复符文。
func RepeatRune(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteRune(r)
	}
	return sb.String()
}

// RemoveRune 移除字符中所有出现的指定符文。
func RemoveRune(s string, r rune) string {
	if s == "" {
		return s
	}
	sb := strings.Builder{}
	sb.Grow(len(s) / 2)

	for _, v := range s {
		if v != r {
			sb.WriteRune(v)
		}
	}
	return sb.String()
}

// RemoveString 移除字符串中所有出现的子串。
func RemoveString(s, substr string) string {
	if s == "" || substr == "" {
		return s
	}
	return strings.ReplaceAll(s, substr, "")
}

// Rotate 按移动位数旋转(循环位移)字符串。
// 若shift为正，则向右旋转。
// 若shift为负，则向左旋转。
func Rotate(s string, shift int) string {
	if shift == 0 {
		return s
	}
	sLen := len(s)
	if sLen == 0 {
		return s
	}

	shiftMod := shift % sLen
	if shiftMod == 0 {
		return s
	}

	offset := -(shiftMod)
	sb := strings.Builder{}
	sb.Grow(sLen)
	_, _ = sb.WriteString(SubStart(s, offset))
	_, _ = sb.WriteString(Sub(s, 0, offset))
	return sb.String()
}

// Sub 返回字符串指定起止位置的子串。
// 起止位置基于 unicode/utf8 计数。
func Sub(s string, start, stop int) string {
	return sub(s, start, stop)
}

// SubStart 返回字符串从起点位置开始的子串。
// 起始位置基于 unicode/utf8 计数。
func SubStart(s string, start int) string {
	return sub(s, start, math.MaxInt64)
}

func sub(s string, start, stop int) string {
	if s == "" {
		return ""
	}

	unicodeLen := utf8.RuneCountInString(s)

	// 终点纠偏
	if stop < 0 {
		stop += unicodeLen
	}
	if stop > unicodeLen {
		stop = unicodeLen
	}

	// 起点纠偏
	if start < 0 {
		start += unicodeLen
	}
	if start > stop {
		return ""
	}

	// 负值纠偏
	if start < 0 {
		start = 0
	}
	if stop < 0 {
		stop = 0
	}

	if start == 0 && stop == unicodeLen {
		return s
	}

	sb := strings.Builder{}
	sb.Grow(stop - start)
	runeIndex := 0
	for _, r := range s {
		if runeIndex >= stop {
			break
		}
		if runeIndex >= start {
			sb.WriteRune(r)
		}
		runeIndex++
	}
	return sb.String()
}

// MustReverse 翻转字符串，出错则传播恐慌。
func MustReverse(s string) string {
	result, err := Reverse(s)
	if err != nil {
		panic(err)
	}
	return result
}

// Reverse 反转字符串，返回结果和潜在错误。
func Reverse(s string) (string, error) {
	if s == "" {
		return s, nil
	}
	src := hack.StringToBytes(s)
	dst := make([]byte, len(s))
	srcIndex := len(s)
	dstIndex := 0
	for srcIndex > 0 {
		r, n := utf8.DecodeLastRune(src[:srcIndex])
		if r == utf8.RuneError {
			return hack.BytesToString(dst), ErrDecodeRune
		}
		utf8.EncodeRune(dst[dstIndex:], r)
		srcIndex -= n
		dstIndex += n
	}
	return hack.BytesToString(dst), nil
}

// Shuffle 随机打乱字符串并返回新串。
func Shuffle(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	index := 0
	for i := len(runes) - 1; i > 0; i-- {
		index = fastrand.Intn(i + 1)
		if i != index {
			runes[i], runes[index] = runes[index], runes[i]
		}
	}
	return string(runes)
}

// ContainsAnySubstrings 判断字符串中是否包含指定的任一子串。
func ContainsAnySubstrings(s string, subs []string) bool {
	for _, substr := range subs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

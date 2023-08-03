package utils

import (
	"testing"

	"github.com/favbox/gosky/wind/pkg/common/test/assert"
)

// 函数 AddMissingPort 只添加丢失的端口，不考虑其他错误情况。
func TestPathAddMissingPort(t *testing.T) {
	ipList := []string{"127.0.0.1", "111.111.1.1", "[0:0:0:0:0:ffff:192.1.56.10]", "[0:0:0:0:0:ffff:c0a8:101]", "www.foobar.com"}
	for _, ip := range ipList {
		assert.DeepEqual(t, ip+":443", AddMissingPort(ip, true))
		assert.DeepEqual(t, ip+":80", AddMissingPort(ip, false))
		customizedPort := ":8080"
		assert.DeepEqual(t, ip+customizedPort, AddMissingPort(ip+customizedPort, true))
		assert.DeepEqual(t, ip+customizedPort, AddMissingPort(ip+customizedPort, false))
	}
}

func TestCleanPath(t *testing.T) {
	normalPath := "/Foo/Bar/go/src/github.com/gosky/wind/pkg/common/utils/path_test.go"
	expectedNormalPath := "/Foo/Bar/go/src/github.com/gosky/wind/pkg/common/utils/path_test.go"
	cleanNormalPath := CleanPath(normalPath)
	assert.DeepEqual(t, expectedNormalPath, cleanNormalPath)

	singleDotPath := "/Foo/Bar/./././go/src"
	expectedSingleDotPath := "/Foo/Bar/go/src"
	cleanSingleDotPath := CleanPath(singleDotPath)
	assert.DeepEqual(t, expectedSingleDotPath, cleanSingleDotPath)

	doubleDotPath := "../../.."
	expectedDoubleDotPath := "/"
	cleanDoublePotPath := CleanPath(doubleDotPath)
	assert.DeepEqual(t, expectedDoubleDotPath, cleanDoublePotPath)

	// 多点可作文件名
	multiDotPath := "/../...."
	expectedMultiDotPath := "/...."
	cleanMultiDotPath := CleanPath(multiDotPath)
	assert.DeepEqual(t, expectedMultiDotPath, cleanMultiDotPath)

	nullPath := ""
	expectedNullPath := "/"
	cleanNullPath := CleanPath(nullPath)
	assert.DeepEqual(t, expectedNullPath, cleanNullPath)

	relativePath := "/Foo/Bar/../go/src/../../github.com/gosky/wind"
	expectedRelativePath := "/Foo/github.com/gosky/wind"
	cleanRelativePath := CleanPath(relativePath)
	assert.DeepEqual(t, expectedRelativePath, cleanRelativePath)

	multiSlashPath := "///////Foo//Bar////go//src/github.com/gosky/wind//.."
	expectedMultiSlashPath := "/Foo/Bar/go/src/github.com/gosky"
	cleanMultiSlashPath := CleanPath(multiSlashPath)
	assert.DeepEqual(t, expectedMultiSlashPath, cleanMultiSlashPath)
}

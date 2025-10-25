package tests

import (
	"testing"
)

// TestAll 运行所有测试的主函数
func TestAll(t *testing.T) {
	// 这个文件用于组织和运行所有测试
	// 每个功能模块的测试已经在各自的文件中实现
	// 这里我们只是确保测试套件能被正确识别
	t.Log("All tests will be run by go test command")
}

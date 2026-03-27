//go:build ignore

package main

import (
	"fmt"
	"strings"

	"github.com/XavierMary56/automatic_review/go-server/internal/service"
)

func main() {
	// 测试新的微信变体屏蔽规则
	tests := []struct {
		content     string
		shouldBlock bool
	}{
		{"加薇看后01", true},
		{"加Ⅴ看后01", true},
		{"加v看后01", true},
		{"加V看后01", true},
		{"加vx看后01", true},
		{"加VX看后01", true},
		{"加q看后01", true},
		{"加Q看后01", true},
		{"加qq看后01", true},
		{"加QQ看后01", true},
	}

	fmt.Println("测试微信/QQ变体屏蔽规则:")
	fmt.Println(strings.Repeat("=", 50))

	allPass := true
	for _, test := range tests {
		isBlocked := service.TestLooksLikeAdOrContactExternal(test.content)

		status := "PASS"
		if isBlocked != test.shouldBlock {
			status = "FAIL"
			allPass = false
		}
		fmt.Printf("%s | %s | 被屏蔽: %v (期望: %v)\n", status, test.content, isBlocked, test.shouldBlock)
	}

	fmt.Println(strings.Repeat("=", 50))
	if allPass {
		fmt.Println("所有测试通过！")
	} else {
		fmt.Println("有测试失败！")
	}
}

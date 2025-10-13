package main

import (
	"log"

	"github.com/alpen/alpen-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// 使用 log 输出是为了未来可以统一接入结构化日志
		log.Fatalf("命令执行失败: %v", err)
	}
}

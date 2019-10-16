package main

import (
	"fmt"
	"os/exec"
)

func main() {
	var(
		cmd *exec.Cmd
		err error
	)
	cmd = exec.Command("/bin/bash","-c","echo 1;echo 2;")
	// 创建fork子进程
	// pipe管道
	// 捕获子进程输出
	// Run 子进程创建 命令执行
	if err = cmd.Run(); err != nil {
		fmt.Println(err)
	}
}

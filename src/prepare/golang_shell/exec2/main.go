package main

import (
	"fmt"
	"os/exec"
)

func main() {
	var(
		cmd *exec.Cmd
		// 二进制切片
		output []byte
		err error
	)
	cmd = exec.Command("/bin/bash","-c","sleep 5;ls -l;sleep 1;echo hello world!")
	// 执行子进程 并且 捕获子进程输出(pipe)
	if output,err = cmd.CombinedOutput(); err != nil {
		fmt.Println(err)
		return
	}
	// 打印子进程输出 output是UTF-8的字符串
	// fmt.Println(output)
	// golang没有隐式类型转换 只有显示类型转换
	fmt.Println(string(output))
}

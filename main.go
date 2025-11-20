package main

import (
	cmd "mimo/cmd"
	_ "mimo/rpclient/rpc"
)

func main() {
	// 启用命令自动补全
	cmd.RootCmd.CompletionOptions.DisableDefaultCmd = false
	cmd.RootCmd.SuggestionsMinimumDistance = 1

	// 执行根命令
	cmd.Execute()
}

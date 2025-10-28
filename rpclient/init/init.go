package cmd

import (
	"fmt"
	cmd "mimo/cmd"
	"mimo/internal/run"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "MIMO system updater",
	Long:  "执行系统资源或目标更新，可选择 --sys 或 --target",
	Run: func(cmd *cobra.Command, args []string) {
		sysFlag, _ := cmd.Flags().GetBool("sys")
		tgtFlag, _ := cmd.Flags().GetBool("target")

		if !sysFlag && !tgtFlag {
			fmt.Println("Specify one of the options: --sys or --target")
			return
		}

		if sysFlag {
			run.RunUpdate()
			return
		}
		if tgtFlag {
			run.RuntgtUpdate()
			return
		}
	},
}

func init() {
	// 为 update 命令添加 flags
	updateCmd.Flags().Bool("sys", false, "执行系统更新")
	updateCmd.Flags().Bool("target", false, "执行target更新")

	// 注册到根命令
	cmd.RootCmd.AddCommand(updateCmd)
}

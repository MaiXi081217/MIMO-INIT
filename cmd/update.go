package cmd

import (
	"fmt"
	"mimo/internal/run"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "MIMO system updater",
	Long:  "执行系统资源或目标更新，可选择 --sys 或 --target",
	RunE: func(cmd *cobra.Command, args []string) error {
		sysFlag, _ := cmd.Flags().GetBool("sys")
		tgtFlag, _ := cmd.Flags().GetBool("target")

		if !sysFlag && !tgtFlag {
			return fmt.Errorf("specify one of the options: --sys or --target")
		}

		if sysFlag {
			return run.RunUpdate()
		}
		if tgtFlag {
			return run.RuntgtUpdate()
		}
		return nil
	},
}

func init() {
	// 为 update 命令添加 flags
	updateCmd.Flags().Bool("sys", false, "执行系统更新")
	updateCmd.Flags().Bool("target", false, "执行target更新")

	// 注册到根命令
	RootCmd.AddCommand(updateCmd)
}

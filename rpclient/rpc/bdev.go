package rpc

import (
	"fmt"
	. "mimo/cmd"
	"mimo/rpclient/goclient"

	"github.com/spf13/cobra"
)

var (
	bdevName string
	timeout  int
)

// bdev_get_bdevs 命令
var bdevGetBdevsCmd = &cobra.Command{
	Use:   "bdev_get_bdevs",
	Short: "获取 bdev 信息",
	Long:  `查询当前系统中注册的所有 bdev 设备，或根据名称查询特定设备。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		params := goclient.BuildParams(map[string]any{
			"name":       bdevName,
			"timeout_ms": timeout * 1000,
		})

		result, err := goclient.Call("bdev_get_bdevs", params)
		if err != nil {
			return fmt.Errorf("获取 bdev 失败: %v", err)
		}

		fmt.Println("返回结果：")
		fmt.Println(string(result))
		return nil
	},
}




func init() {
	bdevGetBdevsCmd.Flags().StringVarP(&bdevName, "bdev", "b", "", "bdev 名称")
	bdevGetBdevsCmd.Flags().IntVarP(&timeout, "timeout", "t", 3, "超时时间（秒）")

	RootCmd.AddCommand(bdevGetBdevsCmd)
}

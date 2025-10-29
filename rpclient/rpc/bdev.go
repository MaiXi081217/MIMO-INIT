package rpc

import (
	"fmt"
	. "mimo/cmd"
	"mimo/rpclient/goclient"

	"github.com/spf13/cobra"
)

// ======================================================
// bdev_get_bdevs
// ======================================================
func bdevGetBdevsCmd() *cobra.Command {
	var (
		bdevName string
		timeout  int
	)

	cmd := &cobra.Command{
		Use:   "bdev_get_bdevs",
		Short: "List or query SPDK block devices.",
		Long: `If no parameters are given, all block devices are listed.
If a name is given, only that bdev is returned.
With a nonzero timeout, waits until the bdev appears or the timeout expires.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			params := goclient.BuildParams(map[string]any{
				"name":       bdevName,
				"timeout_ms": timeout * 1000,
			})

			result, err := goclient.Call("bdev_get_bdevs", params)
			if err != nil {
				return fmt.Errorf("get bdev failed: %v", err)
			}

			fmt.Println("返回结果：")
			fmt.Println(string(result))
			return nil
		},
	}

	cmd.Flags().StringVarP(&bdevName, "bdev", "b", "", "bdev name (optional)")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "timeout in seconds (optional)")
	return cmd
}

// ======================================================
// bdev_nvme_attach_controller
// ======================================================
func bdevNvmeAttachControllerCmd() *cobra.Command {
	var (
		name   string
		trtype string
		traddr string
	)

	cmd := &cobra.Command{
		Use:   "bdev_nvme_attach_controller",
		Short: "Attach a local PCIe NVMe controller.",
		Long:  "Attach a local PCIe NVMe controller to the system using its PCIe address.",
		RunE: func(cmd *cobra.Command, args []string) error {

			params := goclient.BuildParams(map[string]any{
				"name":   name,
				"trtype": trtype,
				"traddr": traddr,
			})

			result, err := goclient.Call("bdev_nvme_attach_controller", params)
			if err != nil {
				return fmt.Errorf("连接 NVMe 控制器失败: %v", err)
			}

			fmt.Println("返回结果：")
			fmt.Println(string(result))
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "bdev", "b", "", "bdev name (required)")
	cmd.MarkFlagRequired("bdev")

	cmd.Flags().StringVarP(&trtype, "trtype", "t", "PCIe", "Transport type, default PCIe")
	cmd.Flags().StringVarP(&traddr, "traddr", "a", "", "PCIe address (required)")
	cmd.MarkFlagRequired("traddr")

	return cmd
}

func init() {
	RootCmd.AddCommand(bdevGetBdevsCmd())
	RootCmd.AddCommand(bdevNvmeAttachControllerCmd())
}

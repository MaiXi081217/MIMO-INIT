package rpc

import (
	"fmt"
	. "mimo/cmd"
	"mimo/rpclient/goclient"
	"strings"

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

	/*
		Example response:
		{
		  "jsonrpc": "2.0",
		  "id": 1,
		  "result": [
		    {
		      "name": "Malloc0",
		      "product_name": "Malloc disk",
		      "block_size": 512,
		      "num_blocks": 20480,
		      "claimed": false,
		      "zoned": false,
		      "supported_io_types": {
		        "read": true,
		        "write": true,
		        "unmap": true,
		        "write_zeroes": true,
		        "flush": true,
		        "reset": true,
		        "nvme_admin": false,
		        "nvme_io": false
		      },
		      "driver_specific": {}
		    }
		  ]
		}
	*/
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
	/*
		Example response:
		{
		  "jsonrpc": "2.0",
		  "id": 1,
		  "result": [
		    "Nvme0n1"
		  ]
		}
	*/

}

// ======================================================
// bdev_malloc_create
// ======================================================
func bdevMallocCreateCmd() *cobra.Command {
	var (
		bdevName  string
		uuid      string
		totalSize float64
		blockSize int
	)

	cmd := &cobra.Command{
		Use:   "bdev_malloc_create",
		Short: "Create a malloc bdev",
		Long:  "Create a malloc bdev with specified total size (MB) and block size (bytes).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if totalSize <= 0 || blockSize <= 0 {
				return fmt.Errorf("total_size and block_size must be positive")
			}

			numBlocks := int((totalSize * 1024 * 1024) / float64(blockSize))

			params := goclient.BuildParams(map[string]any{
				"name":       bdevName,
				"uuid":       uuid,
				"block_size": blockSize,
				"num_blocks": numBlocks,
			})

			result, err := goclient.Call("bdev_malloc_create", params)
			if err != nil {
				return fmt.Errorf("create malloc bdev failed: %v", err)
			}

			fmt.Println("返回结果：")
			fmt.Println(string(result))
			return nil
		},
	}

	cmd.Flags().StringVarP(&bdevName, "bdev", "b", "", "Name of the malloc bdev (optional)")
	cmd.Flags().StringVarP(&uuid, "uuid", "u", "", "UUID of the malloc bdev (optional)")
	cmd.Flags().Float64VarP(&totalSize, "size", "s", 0, "Total size of malloc bdev in MB (required)")
	cmd.Flags().IntVarP(&blockSize, "block", "z", 0, "Block size in bytes (required)")

	cmd.MarkFlagRequired("size")
	cmd.MarkFlagRequired("block")

	return cmd
	/*
		Example response:
		{
		  "jsonrpc": "2.0",
		  "id": 1,
		  "result": "Malloc0"
		}
	*/
}

func bdevRaidCreateCmd() *cobra.Command {
	var (
		name        string
		raidLevel   string
		baseBdevs   string
		stripSizeKB int
		uuid        string
		superblock  bool
	)

	cmd := &cobra.Command{
		Use:   "bdev_raid_create",
		Short: "Create a RAID bdev",
		Long:  "Construct a new RAID bdev from base bdevs with specified RAID level and optional strip size.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || raidLevel == "" || baseBdevs == "" {
				return fmt.Errorf("name, raid_level, and base_bdevs are required")
			}

			// baseBdevs 支持空格分隔字符串
			baseList := strings.Fields(baseBdevs)

			params := goclient.BuildParams(map[string]any{
				"name":          name,
				"raid_level":    raidLevel,
				"base_bdevs":    baseList,
				"strip_size_kb": stripSizeKB,
				"uuid":          uuid,
				"superblock":    superblock,
			})

			result, err := goclient.Call("bdev_raid_create", params)
			if err != nil {
				return fmt.Errorf("create RAID bdev failed: %v", err)
			}

			fmt.Println("返回结果：")
			fmt.Println(string(result))
			return nil
		},
	}

	// 短选项和长选项
	cmd.Flags().StringVarP(&name, "name", "n", "", "RAID bdev name (required)")
	cmd.Flags().StringVarP(&raidLevel, "raid-level", "r", "", "RAID level, e.g., 0, 1, concat (required)")
	cmd.Flags().StringVarP(&baseBdevs, "base-bdevs", "b", "", "Base bdevs, space-separated in quotes (required)")
	cmd.Flags().IntVarP(&stripSizeKB, "strip-size-kb", "z", 64, "Strip size in KB (optional),default 64")
	cmd.Flags().StringVar(&uuid, "uuid", "", "UUID for this RAID bdev (optional)")
	cmd.Flags().BoolVarP(&superblock, "superblock", "s", false, "Store RAID info in superblock on each base bdev(optional)")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("raid-level")
	cmd.MarkFlagRequired("base-bdevs")

	return cmd

	/*
		Example response:
		{
		  "jsonrpc": "2.0",
		  "id": 1,
		  "result": true
		}
	*/
}

func init() {
	RootCmd.AddCommand(bdevGetBdevsCmd())
	RootCmd.AddCommand(bdevNvmeAttachControllerCmd())
	RootCmd.AddCommand(bdevMallocCreateCmd())
	RootCmd.AddCommand(bdevRaidCreateCmd())
}

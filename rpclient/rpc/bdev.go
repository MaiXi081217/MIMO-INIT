package rpc

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mimo/mimo-rpc-service/service"
	"github.com/spf13/cobra"
	. "mimo/cmd"
)

// getBdevService 从命令中获取 socket 地址并创建服务实例
func getBdevService(cmd *cobra.Command) *service.BdevService {
	socketAddr, _ := cmd.Root().PersistentFlags().GetString("socket")
	return service.NewBdevService(socketAddr)
}

// printResult 格式化并打印结果
func printResult(result interface{}) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

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
			result, err := getBdevService(cmd).GetBdevs(bdevName, timeout*1000)
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().StringVarP(&bdevName, "bdev", "b", "", "bdev name (optional)")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "timeout in seconds (optional, default: 0)")
	return cmd
}

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
			result, err := getBdevService(cmd).AttachNvmeController(name, trtype, traddr)
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "b", "", "Name of the NVMe controller (required)")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVarP(&trtype, "trtype", "t", "", "Transport type: e.g., rdma, pcie (required)")
	cmd.Flags().StringVarP(&traddr, "traddr", "a", "", "Transport address: e.g., an ip address or BDF (required)")
	cmd.MarkFlagRequired("trtype")
	cmd.MarkFlagRequired("traddr")

	return cmd
}

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
			result, err := getBdevService(cmd).CreateMallocBdev(bdevName, uuid, totalSize, blockSize)
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().StringVarP(&bdevName, "bdev", "b", "", "Name of the malloc bdev (optional)")
	cmd.Flags().StringVarP(&uuid, "uuid", "u", "", "UUID of the malloc bdev (optional)")
	cmd.Flags().Float64VarP(&totalSize, "size", "s", 0, "Total size of malloc bdev in MB (required)")
	cmd.Flags().IntVarP(&blockSize, "block", "z", 0, "Block size in bytes (required)")

	cmd.MarkFlagRequired("size")
	cmd.MarkFlagRequired("block")

	return cmd
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
			baseList := strings.Fields(baseBdevs)
			req := service.CreateRaidBdevRequest{
				Name:        name,
				RaidLevel:   raidLevel,
				BaseBdevs:   baseList,
				StripSizeKB: stripSizeKB,
				UUID:        uuid,
				Superblock:  superblock,
			}
			result, err := getBdevService(cmd).CreateRaidBdev(req)
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "RAID bdev name (required)")
	cmd.Flags().StringVarP(&raidLevel, "raid-level", "r", "", "RAID level: raid0, raid1, raid10 (or 10), concat (required)")
	cmd.Flags().StringVarP(&baseBdevs, "base-bdevs", "b", "", "Base bdevs, whitespace separated list in quotes (required)")
	cmd.Flags().IntVarP(&stripSizeKB, "strip-size-kb", "z", 0, "Strip size in KB (optional)")
	cmd.Flags().StringVar(&uuid, "uuid", "", "UUID for this RAID bdev (optional)")
	cmd.Flags().BoolVarP(&superblock, "superblock", "s", false, "Store RAID info in superblock on each base bdev (default: false)")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("raid-level")
	cmd.MarkFlagRequired("base-bdevs")

	return cmd
}

func bdevNvmeDetachControllerCmd() *cobra.Command {
	var (
		trtype string
		traddr string
	)

	cmd := &cobra.Command{
		Use:   "bdev_nvme_detach_controller",
		Short: "Detach an NVMe controller and delete any associated bdevs",
		Long:  "Detach an NVMe controller and delete any associated bdevs.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			result, err := getBdevService(cmd).DetachNvmeController(name, trtype, traddr)
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().StringVarP(&trtype, "trtype", "t", "", "NVMe-oF target trtype: e.g., rdma, pcie")
	cmd.Flags().StringVarP(&traddr, "traddr", "a", "", "NVMe-oF target address: e.g., an ip address or BDF")

	return cmd
}

func bdevMallocDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bdev_malloc_delete",
		Short: "Delete a malloc bdev",
		Long:  "Delete a malloc bdev.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := getBdevService(cmd).DeleteMallocBdev(args[0])
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	return cmd
}

func bdevRaidDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bdev_raid_delete",
		Short: "Delete existing RAID bdev",
		Long:  "Delete existing RAID bdev.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := getBdevService(cmd).DeleteRaidBdev(args[0])
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	return cmd
}

func bdevRaidAddBaseBdevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bdev_raid_add_base_bdev",
		Short: "Add base bdev to existing RAID bdev",
		Long:  "Add base bdev to existing RAID bdev.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := getBdevService(cmd).AddRaidBaseBdev(args[0], args[1])
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	return cmd
}

func bdevRaidRemoveBaseBdevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bdev_raid_remove_base_bdev",
		Short: "Remove base bdev from existing RAID bdev",
		Long:  "Remove base bdev from existing RAID bdev.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := getBdevService(cmd).RemoveRaidBaseBdev(args[0])
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	return cmd
}

func bdevWipeSuperblockCmd() *cobra.Command {
	var size int

	cmd := &cobra.Command{
		Use:   "bdev_wipe_superblock",
		Short: "Wipe superblock area of a bdev",
		Long:  "Wipe superblock area (first N bytes) of a bdev. Default size is 1MB.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := getBdevService(cmd).WipeSuperblock(args[0], size)
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().IntVarP(&size, "size", "s", 0, "Size in bytes to wipe (default: 1MB)")

	return cmd
}

func init() {
	RootCmd.AddCommand(bdevGetBdevsCmd())
	RootCmd.AddCommand(bdevNvmeAttachControllerCmd())
	RootCmd.AddCommand(bdevNvmeDetachControllerCmd())
	RootCmd.AddCommand(bdevMallocCreateCmd())
	RootCmd.AddCommand(bdevMallocDeleteCmd())
	RootCmd.AddCommand(bdevRaidCreateCmd())
	RootCmd.AddCommand(bdevRaidDeleteCmd())
	RootCmd.AddCommand(bdevRaidAddBaseBdevCmd())
	RootCmd.AddCommand(bdevRaidRemoveBaseBdevCmd())
	RootCmd.AddCommand(bdevWipeSuperblockCmd())
}

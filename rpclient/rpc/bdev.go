package rpc

import (
	"encoding/json"
	"fmt"
	"strings"
	. "mimo/cmd"
	"github.com/mimo/mimo-rpc-service/service"

	"github.com/spf13/cobra"
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
			var result interface{}
			var err error
			
			// 如果没有指定 bdev 名称，使用便利方法
			if bdevName == "" && timeout == 0 {
				result, err = getBdevService(cmd).GetAllBdevs()
			} else {
				result, err = getBdevService(cmd).GetBdevs(bdevName, timeout*1000)
			}
			
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().StringVarP(&bdevName, "bdev", "b", "", "bdev name (optional)")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "timeout in seconds (optional)")
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
			var result interface{}
			var err error
			
			// 如果 trtype 为空或为 "PCIe"，使用便利方法
			if trtype == "" || trtype == "PCIe" {
				result, err = getBdevService(cmd).AttachNvmeControllerByPCIe(name, traddr)
			} else {
				result, err = getBdevService(cmd).AttachNvmeController(name, trtype, traddr)
			}
			
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

	cmd.Flags().StringVarP(&name, "bdev", "b", "", "bdev name (required)")
	cmd.MarkFlagRequired("bdev")

	cmd.Flags().StringVarP(&trtype, "trtype", "t", "PCIe", "Transport type, default PCIe")
	cmd.Flags().StringVarP(&traddr, "traddr", "a", "", "PCIe address (required)")
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
			var result interface{}
			var err error
			
			// 如果只提供了基本参数，使用简化方法
			baseList := strings.Fields(baseBdevs)
			if stripSizeKB == 64 && uuid == "" && !superblock {
				result, err = getBdevService(cmd).CreateRaidBdevSimple(name, raidLevel, baseList)
			} else {
				// 使用完整方法
				req := service.CreateRaidBdevRequest{
					Name:        name,
					RaidLevel:   raidLevel,
					BaseBdevs:   baseList,
					StripSizeKB: stripSizeKB,
					UUID:        uuid,
					Superblock:  superblock,
				}
				result, err = getBdevService(cmd).CreateRaidBdev(req)
			}
			
			if err != nil {
				return err
			}
			return printResult(result)
		},
	}

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
}

func init() {
	RootCmd.AddCommand(bdevGetBdevsCmd())
	RootCmd.AddCommand(bdevNvmeAttachControllerCmd())
	RootCmd.AddCommand(bdevMallocCreateCmd())
	RootCmd.AddCommand(bdevRaidCreateCmd())
}

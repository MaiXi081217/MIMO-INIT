package cmd

import (
	"fmt"
	"os"

	"mimo/rpclient/goclient"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// RootCmd 根命令
var RootCmd = &cobra.Command{
	Use:   "mimo",
	Short: "MIMO Storage CLI",
	Long:  "MIMO Storage 是一个用于管理高性能存储系统的命令行工具。",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 跳过无需 RPC 初始化的命令
		skip := map[string]bool{
			"completion": true,
			"help":       true,
		}
		if skip[cmd.Name()] {
			return nil
		}

		// 仅对已注册命令初始化 RPC
		if isRegisteredCommand(cmd, args) {
			_ = goclient.GetClient()
		}

		return nil
	},
}

// Execute 执行 CLI
func Execute() {
	// 覆盖默认 help
	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		printHelp(cmd)
	})

	if err := RootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// 判断 args[0] 是否是注册的子命令
func isRegisteredCommand(cmd *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, c := range cmd.Commands() {
		if c.Name() == args[0] || contains(c.Aliases, args[0]) {
			return true
		}
	}
	return false
}

// 辅助函数：检查 slice 是否包含字符串
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// 自定义 help 输出
func printHelp(cmd *cobra.Command) {
	fmt.Printf("%s\n\n", cmd.Long)

	// Usage
	fmt.Println("Usage:")
	fmt.Printf("  %s [command]\n\n", cmd.Use)

	// 子命令
	if len(cmd.Commands()) > 0 {
		fmt.Println("Available Commands:")
		for _, c := range cmd.Commands() {
			fmt.Printf("  %-15s %s\n", c.Name(), c.Short)
		}
	}

	// 当前命令 flags
	printFlags(cmd.Flags(), "Flags")

	// 子命令 flags
	for _, c := range cmd.Commands() {
		if c.Flags().HasFlags() {
			printFlags(c.Flags(), fmt.Sprintf("Command %s Flags", c.Name()))
		}
	}
}

// 输出 flag 列表
func printFlags(flags *pflag.FlagSet, title string) {
	if flags.HasFlags() {
		fmt.Printf("\n%s:\n", title)
		flags.VisitAll(func(f *pflag.Flag) {
			shorthand := ""
			if f.Shorthand != "" {
				shorthand = fmt.Sprintf("-%s, ", f.Shorthand)
			}
			def := ""
			if f.DefValue != "" {
				def = fmt.Sprintf(" (default %s)", f.DefValue)
			}
			fmt.Printf("  %s--%s %s%s\n", shorthand, f.Name, f.Usage, def)
		})
	}
}

// completionCmd 自动补全命令
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long:  "Generate the autocompletion script for bash, zsh, fish, or powershell.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return RootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return RootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return RootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return RootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell type: %s", args[0])
		}
	},
}

func init() {
	RootCmd.AddCommand(completionCmd)
}

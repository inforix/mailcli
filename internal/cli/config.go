package cli

import (
	"fmt"
	"os"
	"os/exec"

	"mailcli/internal/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Config management",
	}
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigEditCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var showPassword bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show effective configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if !showPassword {
				cfg = config.Redact(cfg)
			}
			out, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}

	cmd.Flags().BoolVar(&showPassword, "show-password", false, "Show password in output")

	return cmd
}

func newConfigEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open config file in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigPath()
			if err != nil {
				return err
			}
			editor := os.Getenv("EDITOR")
			if editor == "" {
				return fmt.Errorf("EDITOR not set; config file is %s", path)
			}
			editCmd := exec.Command(editor, path)
			editCmd.Stdout = os.Stdout
			editCmd.Stderr = os.Stderr
			editCmd.Stdin = os.Stdin
			return editCmd.Run()
		},
	}

	return cmd
}

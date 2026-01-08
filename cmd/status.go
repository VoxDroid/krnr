package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/install"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show krnr installation status (user and system)",
	RunE: func(_ *cobra.Command, _ []string) error {
		st, err := install.GetStatus()
		if err != nil {
			return err
		}
		fmt.Printf("krnr status:\n")
		if st.UserInstalled {
			fmt.Printf("- User install: %s (on PATH: %v)\n", st.UserPath, st.UserOnPath)
		} else {
			fmt.Printf("- User install: not found (expected: %s)\n", st.UserPath)
		}
		if st.SystemInstalled {
			fmt.Printf("- System install: %s (on PATH: %v)\n", st.SystemPath, st.SystemOnPath)
		} else {
			fmt.Printf("- System install: not found (expected: %s)\n", st.SystemPath)
		}
		if st.MetadataFound {
			fmt.Printf("- Install metadata: present\n")
		} else {
			fmt.Printf("- Install metadata: not found\n")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

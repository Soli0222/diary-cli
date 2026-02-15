package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "バージョンを表示",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("diary-cli " + Version)
		},
	}
}

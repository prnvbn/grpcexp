package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	_ "embed"
)

//go:embed version.txt
var version string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "grpcexp version",
	Long:  `displays the installed version of the grpcexp cli`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

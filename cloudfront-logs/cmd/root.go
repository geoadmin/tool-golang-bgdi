package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cloudfront-logs command",
	Short: "BGDI CLI tool for cloudfront-logs management",
	Long:  `BGDI CLI tool for cloudfront-logs management`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd:  true, // hides cmd
		DisableDefaultCmd: true, // removes cmd
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func init() {
	rootCmd.PersistentFlags().StringP("profile", "a", "", `AWS account (profile).
	One of ['swisstopo-bgdi', 'swisstopo-bgdi-dev']`)
	rootCmd.PersistentFlags().StringP("bucket", "b", "", "S3 Bucket")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose print output")

	_ = rootCmd.MarkPersistentFlagRequired("profile")
	_ = rootCmd.MarkPersistentFlagRequired("bucket")
}

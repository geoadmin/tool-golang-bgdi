/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/geoadmin/tool-golang-bgdi/lib/fmtc"
	"github.com/spf13/cobra"
)

//-----------------------------------------------------------------------------

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an E2E tests run status",
	Long: `Get an E2E tests run status.

Note that if the tests run is on-going, the command waits until its is finished.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e := initPrint(cmd)
		if e != nil {
			return e
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop() // Ensure cleanup

		client, e := getClient(ctx, cmd)
		if e != nil {
			return e
		}

		id, e := cmd.Flags().GetString("test-id")
		if e != nil {
			return e
		}
		np, e := cmd.Flags().GetBool("no-progress")
		if e != nil {
			return e
		}
		showProgress := !np
		interval, e := cmd.Flags().GetInt("interval")
		if e != nil {
			return e
		}

		input := &codebuild.BatchGetBuildsInput{
			Ids: []string{id},
		}
		r, e := client.BatchGetBuilds(ctx, input)
		if e != nil {
			return fmt.Errorf("failed to get tests run %s: %w", id, e)
		}

		if len(r.Builds) == 0 {
			return fmt.Errorf("failed to get tests run %s: not found", id)
		}
		if !r.Builds[0].BuildComplete {
			cPrintln(fmtc.NoColor, "E2E Tests run found, run in progress waiting to complete...")
			r, e = waitForBuild(ctx, client, id, showProgress, interval)
			if e != nil {
				return e
			}
		} else {
			cPrintf(fmtc.NoColor, "E2E Tests run found and completed at %s\n", r.Builds[0].EndTime.UTC().String())
		}

		return printTestResult(ctx, client, r)
	},
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		// Avoid doing file/folder completion after the command
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

//-----------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringP("test-id", "t", "", "Test ID")
	_ = getCmd.MarkFlagRequired("test-id")
}

//-----------------------------------------------------------------------------

package cmd

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	codebuildCompletions "github.com/geoadmin/tool-golang-bgdi/aws-codebuild/cmd/completions"
	awsCompletions "github.com/geoadmin/tool-golang-bgdi/lib/aws/completions"
	"github.com/spf13/cobra"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Codebuild Project",
	Run: func(_ *cobra.Command, _ []string) {
		if codebuildCompletions.Project == "" {
            log.Fatalf("Project name is required")
        }

        cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsCompletions.Profile))
        if err != nil {
            log.Fatalf("failed to load configuration: %v", err)
        }

        client := codebuild.NewFromConfig(cfg)
        input := &codebuild.StartBuildInput{
            ProjectName: &codebuildCompletions.Project,
        }

        result, err := client.StartBuild(context.TODO(), input)
        if err != nil {
            log.Fatalf("failed to start build: %v", err)
        }

        log.Printf("Build started with ID: %s\n", *result.Build.Id)
	},
}

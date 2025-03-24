package codebuildCompletions

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	awsCompletions "github.com/geoadmin/tool-golang-bgdi/lib/aws/completions"
	"github.com/spf13/cobra"
)

var Project string

//-----------------------------------------------------------------------------

func ProjectCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsCompletions.Profile))
	if err != nil {
		fmt.Printf("failed to load configuration: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	client := codebuild.NewFromConfig(cfg)
	input := &codebuild.ListProjectsInput{}
	result, err := client.ListProjects(context.TODO(), input)
	if err != nil {
		fmt.Printf("failed to list projects: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	return result.Projects, cobra.ShellCompDirectiveNoFileComp
}

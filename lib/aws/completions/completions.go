package awsCompletions

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var Profile string

//-----------------------------------------------------------------------------

func ProfileCompletion(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	configPath := os.ExpandEnv("$HOME/.aws/config")
	cfg, err := ini.Load(configPath)
	if err != nil {
		fmt.Printf("failed to load AWS config file: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	var profiles []string
	for _, section := range cfg.Sections() {
		if section.Name() != "DEFAULT" && section.Name() != "profile default" {
			profiles = append(profiles, section.Name()[8:]) // Remove 'profile ' prefix
		}
	}

	return profiles, cobra.ShellCompDirectiveNoFileComp
}


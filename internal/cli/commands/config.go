package commands

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"gtd/internal/config"
	"gtd/internal/models"
)

// Config prints or updates application configuration.
func Config(cfg *models.Config, args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	defaultContext := fs.String("default-context", "", "Set default context")
	addContext := fs.String("add-context", "", "Add a context")
	autoCommit := fs.String("auto-commit", "", "Set auto-commit (true|false)")
	autoSync := fs.String("auto-sync", "", "Set auto-sync (true|false)")
	syncInterval := fs.Int("sync-interval", -1, "Set sync interval in minutes")
	gitRemote := fs.String("git-remote", "", "Set git remote name")
	gitBranch := fs.String("git-branch", "", "Set git branch name")

	if err := fs.Parse(args); err != nil {
		return err
	}

	changed := false
	if *defaultContext != "" {
		cfg.DefaultContext = *defaultContext
		changed = true
	}
	if *addContext != "" && !containsContext(cfg.Contexts, *addContext) {
		cfg.Contexts = append(cfg.Contexts, *addContext)
		changed = true
	}
	if *autoCommit != "" {
		v, err := strconv.ParseBool(*autoCommit)
		if err != nil {
			return fmt.Errorf("invalid --auto-commit value: %w", err)
		}
		cfg.AutoCommit = v
		changed = true
	}
	if *autoSync != "" {
		v, err := strconv.ParseBool(*autoSync)
		if err != nil {
			return fmt.Errorf("invalid --auto-sync value: %w", err)
		}
		cfg.AutoSync = v
		changed = true
	}
	if *syncInterval > 0 {
		cfg.SyncIntervalMinutes = *syncInterval
		changed = true
	}
	if *gitRemote != "" {
		cfg.GitRemote = *gitRemote
		changed = true
	}
	if *gitBranch != "" {
		cfg.GitBranch = *gitBranch
		changed = true
	}

	if changed {
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("✓ Configuration updated")
	}

	printConfig(cfg)
	return nil
}

func containsContext(contexts []string, target string) bool {
	for _, c := range contexts {
		if strings.EqualFold(c, target) {
			return true
		}
	}
	return false
}

func printConfig(cfg *models.Config) {
	fmt.Println("Current configuration:")
	fmt.Printf("  Data path:      %s\n", cfg.DataPath)
	fmt.Printf("  Default context:%s\n", " "+cfg.DefaultContext)
	fmt.Printf("  Contexts:       %s\n", strings.Join(cfg.Contexts, ", "))
	fmt.Printf("  Auto commit:    %t\n", cfg.AutoCommit)
	fmt.Printf("  Auto sync:      %t\n", cfg.AutoSync)
	fmt.Printf("  Sync interval:  %d minutes\n", cfg.SyncIntervalMinutes)
	fmt.Printf("  Git remote:     %s\n", cfg.GitRemote)
	fmt.Printf("  Git branch:     %s\n", cfg.GitBranch)
}

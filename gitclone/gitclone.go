package gitclone

import (
	"fmt"

	"github.com/bitrise-io/envman/envman"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-git-clone/gitcloneinternal"
)

const (
	trimEnding = "..."
)

// Config ...
type Config struct {
	gitcloneinternal.CheckoutConfig
}

func printLogAndExportEnv(gitCmd git.Git, format, env string, maxEnvLength int) error {
	l, err := gitcloneinternal.Runner.RunForOutput(gitCmd.Log(format))
	if err != nil {
		return err
	}

	if (env == "GIT_CLONE_COMMIT_MESSAGE_SUBJECT" || env == "GIT_CLONE_COMMIT_MESSAGE_BODY") && len(l) > maxEnvLength {
		tv := l[:maxEnvLength-len(trimEnding)] + trimEnding
		log.Printf("Value %s  is bigger than maximum env variable size, trimming", env)
		l = tv
	}

	log.Printf("=> %s\n   value: %s\n", env, l)
	if err := tools.ExportEnvironmentWithEnvman(env, l); err != nil {
		return fmt.Errorf("envman export, error: %v", err)
	}
	return nil
}

func getMaxEnvLength() (int, error) {
	configs, err := envman.GetConfigs()
	if err != nil {
		return 0, err
	}

	return configs.EnvBytesLimitInKB * 1024, nil
}

// Execute is the entry point of the git clone process
func Execute(cfg Config) error {
	maxEnvLength, err := getMaxEnvLength()
	if err != nil {
		return gitcloneinternal.NewStepError(
			"get_max_commit_msg_length_failed",
			fmt.Errorf("failed to set commit message length: %s", err),
			"Getting allowed commit message length failed",
		)
	}

	gitCmd, err := git.New(cfg.CloneIntoDir)
	if err != nil {
		return gitcloneinternal.NewStepError(
			"git_new",
			fmt.Errorf("failed to create git project directory: %v", err),
			"Creating new git project directory failed",
		)
	}

	originPresent, err := gitcloneinternal.IsOriginPresent(gitCmd, cfg.CloneIntoDir, cfg.RepositoryURL)
	if err != nil {
		return gitcloneinternal.NewStepError(
			"check_origin_present_failed",
			fmt.Errorf("checking if origin is present failed: %v", err),
			"Checking wether origin is present failed",
		)
	}

	if originPresent && cfg.ResetRepository {
		if err := gitcloneinternal.ResetRepo(gitCmd); err != nil {
			return gitcloneinternal.NewStepError(
				"reset_repository_failed",
				fmt.Errorf("reset repository failed: %v", err),
				"Resetting repository failed",
			)
		}
	}
	if err := gitcloneinternal.Runner.Run(gitCmd.Init()); err != nil {
		return gitcloneinternal.NewStepError(
			"init_git_failed",
			fmt.Errorf("initializing repository failed: %v", err),
			"Initializing git has failed",
		)
	}
	if !originPresent {
		if err := gitcloneinternal.Runner.Run(gitCmd.RemoteAdd(gitcloneinternal.OriginRemoteName, cfg.RepositoryURL)); err != nil {
			return gitcloneinternal.NewStepError(
				"add_remote_failed",
				fmt.Errorf("adding remote repository failed (%s): %v", cfg.RepositoryURL, err),
				"Adding remote repository failed",
			)
		}
	}

	if err := gitcloneinternal.SetupSparseCheckout(gitCmd, cfg.SparseDirectories); err != nil {
		return err
	}

	if err := gitcloneinternal.CheckoutState(gitCmd, cfg.CheckoutConfig, gitcloneinternal.DefaultPatchSource{}); err != nil {
		return err
	}

	if cfg.UpdateSubmodules {
		if err := gitcloneinternal.UpdateSubmodules(gitCmd, cfg.CheckoutConfig); err != nil {
			return err
		}
	}

	checkoutArg := getCheckoutArg(cfg.Commit, cfg.Tag, cfg.Branch)
	if checkoutArg != "" {
		log.Infof("\nExporting git logs\n")

		for format, env := range map[string]string{
			`%H`:  "GIT_CLONE_COMMIT_HASH",
			`%s`:  "GIT_CLONE_COMMIT_MESSAGE_SUBJECT",
			`%b`:  "GIT_CLONE_COMMIT_MESSAGE_BODY",
			`%an`: "GIT_CLONE_COMMIT_AUTHOR_NAME",
			`%ae`: "GIT_CLONE_COMMIT_AUTHOR_EMAIL",
			`%cn`: "GIT_CLONE_COMMIT_COMMITER_NAME",
			`%ce`: "GIT_CLONE_COMMIT_COMMITER_EMAIL",
		} {
			if err := printLogAndExportEnv(gitCmd, format, env, maxEnvLength); err != nil {
				return gitcloneinternal.NewStepError(
					"export_envs_failed",
					fmt.Errorf("gitCmd log failed: %v", err),
					"Exporting envs failed",
				)
			}
		}

		count, err := gitcloneinternal.Runner.RunForOutput(gitCmd.RevList("HEAD", "--count"))
		if err != nil {
			return gitcloneinternal.NewStepError(
				"count_commits_failed",
				fmt.Errorf("get rev-list failed: %v", err),
				"Counting commits failed",
			)
		}

		log.Printf("=> %s\n   value: %s\n", "GIT_CLONE_COMMIT_COUNT", count)
		if err := tools.ExportEnvironmentWithEnvman("GIT_CLONE_COMMIT_COUNT", count); err != nil {
			return gitcloneinternal.NewStepError(
				"export_envs_commit_count_failed",
				fmt.Errorf("envman export failed: %v", err),
				"Exporting commit count env failed",
			)
		}
	}

	return nil
}

func getCheckoutArg(commit, tag, branch string) string {
	switch {
	case commit != "":
		return commit
	case tag != "":
		return tag
	case branch != "":
		return branch
	default:
		return ""
	}
}

package gitcloneinternal

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
)

// UpdateSubmodules ...
func UpdateSubmodules(gitCmd git.Git, cfg CheckoutConfig) error {
	if err := Runner.Run(gitCmd.SubmoduleUpdate(cfg.LimitSubmoduleUpdateDepth, JobsFlag)); err != nil {
		return NewStepError(
			UpdateSubmodelFailedTag,
			fmt.Errorf("submodule update: %v", err),
			"Updating submodules has failed",
		)
	}

	return nil
}

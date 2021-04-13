package gitcloneinternal

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/log"
)

// CheckoutMethod is the checkout method used
type CheckoutMethod int

const (
	// InvalidCheckoutMethod ...
	InvalidCheckoutMethod CheckoutMethod = iota
	// CheckoutNoneMethod only adds remote, resets repo, updates submodules
	CheckoutNoneMethod
	// CheckoutCommitMethod checks out a given commit
	CheckoutCommitMethod
	// CheckoutTagMethod checks out a given tag
	CheckoutTagMethod
	// CheckoutBranchMethod checks out a given branch
	CheckoutBranchMethod
	// CheckoutPRMergeBranchMethod checks out a MR/PR (when merge branch is available)
	CheckoutPRMergeBranchMethod
	// CheckoutPRDiffFileMethod  checks out a MR/PR (when a diff file is available)
	CheckoutPRDiffFileMethod
	// CheckoutPRManualMergeMethod check out a Merge Request using manual merge
	CheckoutPRManualMergeMethod
	// CheckoutHeadBranchCommitMethod checks out a MR/PR head branch only, without merging into base branch
	CheckoutHeadBranchCommitMethod
	// CheckoutForkCommitMethod checks out a PR source branch, without merging
	CheckoutForkCommitMethod
)

const privateForkAuthWarning = `May fail due to missing authentication as Pull Request opened from a private fork.
A git hosting provider head branch or a diff file is unavailable.`

// ParameterValidationError is returned when there is missing or malformatted parameter for a given parameter set
type ParameterValidationError struct {
	ErrorString string
}

// CheckoutConfig is the git clone step configuration
type CheckoutConfig struct {
	RepositoryURL string `env:"repository_url,required"`
	CloneIntoDir  string `env:"clone_into_dir,required"`

	Commit string `env:"commit"`
	Tag    string `env:"tag"`
	Branch string `env:"branch"`

	PRDestBranch          string `env:"branch_dest"`
	PRID                  int    `env:"pull_request_id"`
	PRSourceRepositoryURL string `env:"pull_request_repository_url"`
	PRMergeBranch         string `env:"pull_request_merge_branch"`
	PRHeadBranch          string `env:"pull_request_head_branch"`

	CloneDepth        int      `env:"clone_depth"`
	FetchTags         bool     `env:"fetch_tags,opt[yes,no]"`
	SparseDirectories []string `env:"sparse_directories,multiline"`
	ShouldMergePR     bool     `env:"merge_pr,opt[yes,no]"`
	ManualMerge       bool     `env:"manual_merge,opt[yes,no]"`

	ResetRepository bool `env:"reset_repository,opt[Yes,No]"`

	UpdateSubmodules          bool `env:"update_submodules,opt[yes,no]"`
	LimitSubmoduleUpdateDepth bool `env:"limit_submodule_update_depth,opt[yes,no]"`

	BuildURL      string `env:"build_url"`
	BuildAPIToken string `env:"build_api_token"`
}

// CheckoutState ...
func CheckoutState(gitCmd git.Git, cfg CheckoutConfig, patch patchSource) error {
	checkoutMethod, diffFile := selectCheckoutMethod(cfg, patch)
	fetchOpts := selectFetchOptions(checkoutMethod, cfg.CloneDepth, cfg.FetchTags, cfg.UpdateSubmodules, len(cfg.SparseDirectories) != 0)

	checkoutStrategy, err := createCheckoutStrategy(checkoutMethod, cfg, diffFile)
	if err != nil {
		return err
	}
	if checkoutStrategy == nil {
		return fmt.Errorf("failed to select a checkout stategy")
	}

	if err := checkoutStrategy.do(gitCmd, fetchOpts, selectFallbacks(checkoutMethod, fetchOpts)); err != nil {
		log.Infof("Checkout strategy used: %T", checkoutStrategy)
		return err
	}

	return nil
}

// Error ...
func (e ParameterValidationError) Error() string {
	return e.ErrorString
}

// NewParameterValidationError returns a new ValidationError
func NewParameterValidationError(msg string) error {
	return ParameterValidationError{ErrorString: msg}
}

// checkoutStrategy is the interface an actual checkout strategy implements
type checkoutStrategy interface {
	do(gitCmd git.Git, fetchOptions fetchOptions, fallback fallbackRetry) error
}

// X: required parameter
// !: used to identify checkout strategy
// _: optional parameter
// |=========================================================================|
// | params\strat| commit | tag | branch | manualMR | headBranch | diffFile  |
// | commit      |  X  !  |     |        |  _/X     |  _/X       |           |
// | tag         |        |  X !|        |          |            |           |
// | branch      |  _     |  _  |  X !   |  X       |            |           |
// | branchDest  |        |     |        |  X  !    |  X !       |  X  !     |
// | PRRepoURL   |        |     |        |  _       |            |           |
// | PRID        |        |     |        |          |            |           |
// | mergeBranch |        |     |        |          |    !       |           |
// | headBranch  |        |     |        |          |  X         |           |
// |=========================================================================|

func selectCheckoutMethod(cfg CheckoutConfig, patch patchSource) (CheckoutMethod, string) {
	isPR := cfg.PRSourceRepositoryURL != "" || cfg.PRDestBranch != "" || cfg.PRMergeBranch != "" || cfg.PRID != 0
	if !isPR {
		if cfg.Commit != "" {
			return CheckoutCommitMethod, ""
		}

		if cfg.Tag != "" {
			return CheckoutTagMethod, ""
		}

		if cfg.Branch != "" {
			return CheckoutBranchMethod, ""
		}

		return CheckoutNoneMethod, ""
	}

	isFork := isFork(cfg.RepositoryURL, cfg.PRSourceRepositoryURL)
	isPrivateSourceRepo := isPrivate(cfg.PRSourceRepositoryURL)
	isPrivateFork := isFork && isPrivateSourceRepo
	isPublicFork := isFork && !isPrivateSourceRepo

	if !cfg.ShouldMergePR {
		if cfg.PRHeadBranch != "" {
			return CheckoutHeadBranchCommitMethod, ""
		}

		if !isFork {
			return CheckoutCommitMethod, ""
		}

		if isPublicFork {
			return CheckoutForkCommitMethod, ""
		}

		if cfg.BuildURL != "" {
			patchFile := getPatchFile(patch, cfg.BuildURL, cfg.BuildAPIToken)
			if patchFile != "" {
				log.Infof("Merging Pull Request despite the option to disable merging, as it is opened from a private fork.")
				return CheckoutPRDiffFileMethod, patchFile
			}
		}

		log.Warnf(privateForkAuthWarning)
		return CheckoutForkCommitMethod, ""
	}

	if !cfg.ManualMerge || isPrivateFork {
		if cfg.PRMergeBranch != "" {
			return CheckoutPRMergeBranchMethod, ""
		}

		if cfg.BuildURL != "" {
			patchFile := getPatchFile(patch, cfg.BuildURL, cfg.BuildAPIToken)
			if patchFile != "" {
				return CheckoutPRDiffFileMethod, patchFile
			}
		}

		log.Warnf(privateForkAuthWarning)
		return CheckoutPRManualMergeMethod, ""
	}

	return CheckoutPRManualMergeMethod, ""
}

func getPatchFile(patch patchSource, buildURL, buildAPIToken string) string {
	if patch != nil {
		patchFile, err := patch.getDiffPath(buildURL, buildAPIToken)
		if err != nil {
			log.Warnf("Diff file unavailable: %v", err)
		} else {
			return patchFile
		}
	}

	return ""
}

func createCheckoutStrategy(checkoutMethod CheckoutMethod, cfg CheckoutConfig, patchFile string) (checkoutStrategy, error) {
	switch checkoutMethod {
	case CheckoutNoneMethod:
		{
			return checkoutNone{}, nil
		}
	case CheckoutCommitMethod:
		{
			branchRef := ""
			if cfg.Branch != "" {
				branchRef = refsHeadsPrefix + cfg.Branch
			}

			params, err := NewCommitParams(cfg.Commit, branchRef, "")
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	case CheckoutTagMethod:
		{
			params, err := NewTagParams(cfg.Tag)
			if err != nil {
				return nil, err
			}

			return checkoutTag{
				params: *params,
			}, nil
		}
	case CheckoutBranchMethod:
		{
			params, err := NewBranchParams(cfg.Branch)
			if err != nil {
				return nil, err
			}

			return checkoutBranch{
				params: *params,
			}, nil
		}
	case CheckoutPRMergeBranchMethod:
		{
			params, err := NewPRMergeBranchParams(cfg.PRDestBranch, cfg.PRMergeBranch)
			if err != nil {
				return nil, err
			}

			return checkoutPRMergeBranch{
				params: *params,
			}, nil
		}
	case CheckoutPRDiffFileMethod:
		{
			prManualMergeStrategy, err := createCheckoutStrategy(CheckoutPRManualMergeMethod, cfg, patchFile)
			if err != nil {
				return nil, err
			}

			params, err := NewPRDiffFileParams(cfg.PRDestBranch, prManualMergeStrategy)
			if err != nil {
				return nil, err
			}

			return checkoutPRDiffFile{
				params:    *params,
				patchFile: patchFile,
			}, nil
		}
	case CheckoutPRManualMergeMethod:
		{
			prRepositoryURL := ""
			if isFork(cfg.RepositoryURL, cfg.PRSourceRepositoryURL) {
				prRepositoryURL = cfg.PRSourceRepositoryURL
			}

			params, err := NewPRManualMergeParams(cfg.Branch, cfg.Commit, prRepositoryURL, cfg.PRDestBranch)
			if err != nil {
				return nil, err
			}

			return checkoutPRManualMerge{
				params: *params,
			}, nil
		}
	case CheckoutHeadBranchCommitMethod:
		{
			headBranchRef := refsPrefix + cfg.PRHeadBranch // ref/pull/2/head
			params, err := NewCommitParams(cfg.Commit, headBranchRef, "")
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	case CheckoutForkCommitMethod:
		{
			sourceBranchRef := refsHeadsPrefix + cfg.Branch
			params, err := NewCommitParams(cfg.Commit, sourceBranchRef, cfg.PRSourceRepositoryURL)
			if err != nil {
				return nil, err
			}

			return checkoutCommit{
				params: *params,
			}, nil
		}
	default:
		return nil, fmt.Errorf("invalid checkout strategy selected")
	}

}

func selectFetchOptions(checkoutStrategy CheckoutMethod, cloneDepth int, fetchTags, fetchSubmodules bool, filterTree bool) fetchOptions {
	opts := fetchOptions{
		depth:           cloneDepth,
		tags:            fetchTags,
		fetchSubmodules: fetchSubmodules,
	}

	opts = selectFilterTreeFetchOption(checkoutStrategy, opts, filterTree)

	return opts
}

func selectFilterTreeFetchOption(checkoutStrategy CheckoutMethod, opts fetchOptions, filterTree bool) fetchOptions {
	if !filterTree {
		return opts
	}

	switch checkoutStrategy {
	case CheckoutCommitMethod,
		CheckoutTagMethod,
		CheckoutBranchMethod,
		CheckoutHeadBranchCommitMethod,
		CheckoutForkCommitMethod:
		opts.filterTree = true
		opts.depth = 0
	default:
	}

	return opts
}

func selectFallbacks(checkoutStrategy CheckoutMethod, fetchOpts fetchOptions) fallbackRetry {
	if fetchOpts.IsFullDepth() {
		return nil
	}

	unshallowFetchOpts := unshallowFetchOptions{
		tags:            fetchOpts.tags,
		fetchSubmodules: fetchOpts.fetchSubmodules,
	}

	switch checkoutStrategy {
	case CheckoutBranchMethod:
		// the given branch's tip will be checked out, no need to unshallow
		return nil
	case CheckoutCommitMethod, CheckoutTagMethod, CheckoutHeadBranchCommitMethod, CheckoutForkCommitMethod:
		return simpleUnshallow{
			traits: unshallowFetchOpts,
		}
	case CheckoutPRMergeBranchMethod, CheckoutPRManualMergeMethod, CheckoutPRDiffFileMethod:
		return resetUnshallow{
			traits: unshallowFetchOpts,
		}
	default:
		return nil
	}
}

package gitcloneinternal

import "testing"

func Test_selectCheckoutMethod(t *testing.T) {
	tests := []struct {
		name        string
		cfg         CheckoutConfig
		patchSource patchSource
		want        CheckoutMethod
	}{
		{
			name: "none",
			cfg:  CheckoutConfig{},
			want: CheckoutNoneMethod,
		},
		{
			name: "commit",
			cfg: CheckoutConfig{
				Commit: "76a934a",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "commit + branch",
			cfg: CheckoutConfig{
				Commit: "76a934ae",
				Branch: "hcnarb",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "branch",
			cfg: CheckoutConfig{
				Branch: "hcnarb",
			},
			want: CheckoutBranchMethod,
		},
		{
			name: "tag",
			cfg: CheckoutConfig{
				Tag: "gat",
			},
			want: CheckoutTagMethod,
		},
		{
			name: "Checkout tag, branch specifed",
			cfg: CheckoutConfig{
				Tag:    "gat",
				Branch: "hcnarb",
			},
			want: CheckoutTagMethod,
		},
		{
			name: "UNSUPPORTED Checkout commit, tag, branch specifed",
			cfg: CheckoutConfig{
				Commit: "76a934ae",
				Tag:    "gat",
				Branch: "hcnarb",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "UNSUPPORTED Checkout commit, tag specifed",
			cfg: CheckoutConfig{
				Commit: "76a934ae",
				Tag:    "gat",
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "PR - no fork - manual merge: branch and commit",
			cfg: CheckoutConfig{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRMergeBranch: "pull/7/merge",
				PRDestBranch:  "master",
				PRID:          7,
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - manual merge: branch and commit, no PRRepoURL or PRID",
			cfg: CheckoutConfig{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - fork - manual merge",
			cfg: CheckoutConfig{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - manual merge: repo is the same with different scheme",
			cfg: CheckoutConfig{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/git-clone-test.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRMergeBranch:         "pull/7/merge",
				PRID:                  7,
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			want: CheckoutPRManualMergeMethod,
		},
		{
			name: "PR - no fork - auto merge - merge branch (GitHub format)",
			cfg: CheckoutConfig{
				PRDestBranch:  "master",
				PRMergeBranch: "pull/5/merge",
				ShouldMergePR: true,
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - no fork - auto merge - diff file",
			cfg: CheckoutConfig{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				PRDestBranch:  "master",
				PRID:          7,
				Commit:        "76a934ae",
				ShouldMergePR: true,
				BuildURL:      "dummy_url",
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
		{
			name: "PR - fork - auto merge - merge branch: private fork overrides manual merge flag",
			cfg: CheckoutConfig{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRMergeBranch:         "pull/7/merge",
				PRID:                  7,
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			want: CheckoutPRMergeBranchMethod,
		},
		{
			name: "PR - fork - auto merge: private fork overrides manual merge flag",
			cfg: CheckoutConfig{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				BuildURL:              "dummy_url",
				ManualMerge:           true,
				ShouldMergePR:         true,
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
		{
			name: "PR - no merge - no fork - auto merge - head branch",
			cfg: CheckoutConfig{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRMergeBranch: "pull/7/merge",
				PRHeadBranch:  "pull/7/head",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: false,
			},
			want: CheckoutHeadBranchCommitMethod,
		},
		{
			name: "PR - no merge - no fork - manual merge",
			cfg: CheckoutConfig{
				Commit:        "76a934ae",
				Branch:        "test/commit-messages",
				PRDestBranch:  "master",
				CloneDepth:    1,
				ManualMerge:   true,
				ShouldMergePR: false,
			},
			want: CheckoutCommitMethod,
		},
		{
			name: "PR - no merge - no fork - diff file exists",
			cfg: CheckoutConfig{
				RepositoryURL: "https://github.com/bitrise-io/git-clone-test.git",
				Commit:        "76a934ae",
				PRDestBranch:  "master",
				PRID:          7,
				ShouldMergePR: false,
				BuildURL:      "dummy_url",
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutCommitMethod,
		},
		{
			name: "PR - no merge - fork - public fork",
			cfg: CheckoutConfig{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "https://github.com/bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         false,
			},
			want: CheckoutForkCommitMethod,
		},
		{
			name: "PR - no merge - fork - auto merge - diff file: private fork",
			cfg: CheckoutConfig{
				RepositoryURL:         "https://github.com/bitrise-io/git-clone-test.git",
				PRSourceRepositoryURL: "git@github.com:bitrise-io/other-repo.git",
				Branch:                "test/commit-messages",
				PRDestBranch:          "master",
				PRID:                  7,
				Commit:                "76a934ae",
				ManualMerge:           true,
				ShouldMergePR:         false,
				BuildURL:              "dummy_url",
			},
			patchSource: MockPatchSource{diffFilePath: "dummy_path"},
			want:        CheckoutPRDiffFileMethod,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := selectCheckoutMethod(tt.cfg, tt.patchSource); got != tt.want {
				t.Errorf("selectCheckoutMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

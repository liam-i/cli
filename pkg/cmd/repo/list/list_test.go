package list

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/test"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ListOptions
		wantsErr string
	}{
		{
			name: "no arguments",
			cli:  "",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "with owner",
			cli:  "monalisa",
			wants: ListOptions{
				Limit:       30,
				Owner:       "monalisa",
				Visibility:  "",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "with limit",
			cli:  "-L 101",
			wants: ListOptions{
				Limit:       101,
				Owner:       "",
				Visibility:  "",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "only public",
			cli:  "--public",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "public",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "only private",
			cli:  "--private",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "private",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "only forks",
			cli:  "--fork",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "",
				Fork:        true,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "only sources",
			cli:  "--source",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "",
				Fork:        false,
				Source:      true,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "with language",
			cli:  "-l go",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "",
				Fork:        false,
				Source:      false,
				Language:    "go",
				Topic:       "",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name: "only archived",
			cli:  "--archived",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    true,
				NonArchived: false,
			},
		},
		{
			name: "only non-archived",
			cli:  "--no-archived",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "",
				Archived:    false,
				NonArchived: true,
			},
		},
		{
			name: "with topic",
			cli:  "--topic cli",
			wants: ListOptions{
				Limit:       30,
				Owner:       "",
				Visibility:  "",
				Fork:        false,
				Source:      false,
				Language:    "",
				Topic:       "cli",
				Archived:    false,
				NonArchived: false,
			},
		},
		{
			name:     "no public and private",
			cli:      "--public --private",
			wantsErr: "specify only one of `--public` or `--private`",
		},
		{
			name:     "no forks with sources",
			cli:      "--fork --source",
			wantsErr: "specify only one of `--source` or `--fork`",
		},
		{
			name:     "conflicting archived",
			cli:      "--archived --no-archived",
			wantsErr: "specify only one of `--archived` or `--no-archived`",
		},
		{
			name:     "too many arguments",
			cli:      "monalisa hubot",
			wantsErr: "accepts at most 1 arg(s), received 2",
		},
		{
			name:     "invalid limit",
			cli:      "-L 0",
			wantsErr: "invalid limit: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &cmdutil.Factory{}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr != "" {
				assert.EqualError(t, err, tt.wantsErr)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
			assert.Equal(t, tt.wants.Owner, gotOpts.Owner)
			assert.Equal(t, tt.wants.Visibility, gotOpts.Visibility)
			assert.Equal(t, tt.wants.Fork, gotOpts.Fork)
			assert.Equal(t, tt.wants.Topic, gotOpts.Topic)
			assert.Equal(t, tt.wants.Source, gotOpts.Source)
			assert.Equal(t, tt.wants.Archived, gotOpts.Archived)
			assert.Equal(t, tt.wants.NonArchived, gotOpts.NonArchived)
		})
	}
}

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()
	ios.SetStdoutTTY(isTTY)
	ios.SetStdinTTY(isTTY)
	ios.SetStderrTTY(isTTY)

	factory := &cmdutil.Factory{
		IOStreams: ios,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: rt}, nil
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
	}

	cmd := NewCmdList(factory, nil)

	argv, err := shlex.Split(cli)
	if err != nil {
		return nil, err
	}
	cmd.SetArgs(argv)

	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err = cmd.ExecuteC()
	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func TestRepoList_nontty(t *testing.T) {
	ios, _, stdout, stderr := iostreams.Test()
	ios.SetStdoutTTY(false)
	ios.SetStdinTTY(false)
	ios.SetStderrTTY(false)

	httpReg := &httpmock.Registry{}
	defer httpReg.Verify(t)

	httpReg.Register(
		httpmock.GraphQL(`query RepositoryList\b`),
		httpmock.FileResponse("./fixtures/repoList.json"),
	)

	opts := ListOptions{
		IO: ios,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: httpReg}, nil
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		Now: func() time.Time {
			t, _ := time.Parse(time.RFC822, "19 Feb 21 15:00 UTC")
			return t
		},
		Limit: 30,
	}

	err := listRun(&opts)
	assert.NoError(t, err)

	assert.Equal(t, "", stderr.String())

	assert.Equal(t, heredoc.Doc(`
		octocat/hello-world	My first repository	public	2021-02-19T06:34:58Z
		octocat/cli	GitHub CLI	public, fork	2021-02-19T06:06:06Z
		octocat/testing		private	2021-02-11T22:32:05Z
	`), stdout.String())
}

func TestRepoList_tty(t *testing.T) {
	ios, _, stdout, stderr := iostreams.Test()
	ios.SetStdoutTTY(true)
	ios.SetStdinTTY(true)
	ios.SetStderrTTY(true)

	httpReg := &httpmock.Registry{}
	defer httpReg.Verify(t)

	httpReg.Register(
		httpmock.GraphQL(`query RepositoryList\b`),
		httpmock.FileResponse("./fixtures/repoList.json"),
	)

	opts := ListOptions{
		IO: ios,
		HttpClient: func() (*http.Client, error) {
			return &http.Client{Transport: httpReg}, nil
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		Now: func() time.Time {
			t, _ := time.Parse(time.RFC822, "19 Feb 21 15:00 UTC")
			return t
		},
		Limit: 30,
	}

	err := listRun(&opts)
	assert.NoError(t, err)

	assert.Equal(t, "", stderr.String())

	assert.Equal(t, heredoc.Doc(`

		Showing 3 of 3 repositories in @octocat

		octocat/hello-world  My first repository  public        8h
		octocat/cli          GitHub CLI           public, fork  8h
		octocat/testing                           private       7d
	`), stdout.String())
}

func TestRepoList_filtering(t *testing.T) {
	http := &httpmock.Registry{}
	defer http.Verify(t)

	http.Register(
		httpmock.GraphQL(`query RepositoryList\b`),
		httpmock.GraphQLQuery(`{}`, func(_ string, params map[string]interface{}) {
			assert.Equal(t, "PRIVATE", params["privacy"])
			assert.Equal(t, float64(2), params["perPage"])
		}),
	)

	output, err := runCommand(http, true, `--private --limit 2 `)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", output.Stderr())
	assert.Equal(t, "\nNo results match your search\n\n", output.String())
}

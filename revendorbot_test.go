package revendorbot

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/google/go-github/v21/github"
)

func TestFilterComment(t *testing.T) {
	assert.False(t, filterComment(&github.IssueCommentEvent{
		Action: github.String("deleted"),
	}))

	assert.False(t, filterComment(&github.IssueCommentEvent{
		Action: github.String("created"),
		Comment: &github.IssueComment{
			Body: github.String("hello world!"),
		},
	}))

	assert.False(t, filterComment(&github.IssueCommentEvent{
		Action: github.String("created"),
		Comment: &github.IssueComment{
			Body: github.String("/revendor"),
		},
		Issue: &github.Issue{PullRequestLinks: nil},
	}))

	assert.True(t, filterComment(&github.IssueCommentEvent{
		Action: github.String("created"),
		Comment: &github.IssueComment{
			Body: github.String("/revendor"),
		},
		Issue: &github.Issue{PullRequestLinks: &github.PullRequestLinks{}},
	}))

	assert.True(t, filterComment(&github.IssueCommentEvent{
		Action: github.String("created"),
		Comment: &github.IssueComment{
			Body: github.String(`
   /revendor

`),
		},
		Issue: &github.Issue{PullRequestLinks: &github.PullRequestLinks{}},
	}))
}

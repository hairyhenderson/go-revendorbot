package revendorbot

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v21/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	ghtokName = "GITHUB_TOKEN"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

// Repo - an interface to match various kinds of Repository structs
type Repo interface {
	GetName() string
	GetOwner() *github.User
	GetCloneURL() string
}

// Bot -
type Bot struct {
	ghclient *github.Client
	ctx      context.Context
}

// New bot
func New(ctx context.Context) (*Bot, error) {
	token := os.Getenv(ghtokName)
	if token == "" {
		return nil, errors.Errorf("GitHub API token missing - must set %s", ghtokName)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	hc := &http.Client{Transport: &oauth2.Transport{Source: ts}}
	client := github.NewClient(hc)
	return &Bot{
		ghclient: client,
		ctx:      ctx,
	}, nil
}

// Handle -
func (b *Bot) Handle(eventType, deliveryID string, payload []byte) error {
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		return errors.Wrap(err, "failed to parse webhook event")
	}

	switch e := event.(type) {
	case *github.PushEvent:
		err = b.handlePush(e)
	case *github.IssueCommentEvent:
		if !filterComment(e) {
			return nil
		}
		repo := e.GetRepo()
		prNum := e.GetIssue().GetNumber()
		err = b.handleComment(e)
		dur := time.Now().Sub(startTime)
		if err != nil {
			aerr := b.AddComment(b.ctx, repo, prNum, ":robot: :warning: RevendorBot got an error while trying to revendor:\n```\n"+err.Error()+"\n```\n\nTook "+dur.String())
			if aerr != nil {
				log.Printf("errored trying to add a comment %v", aerr)
			}
			return err
		}
		err = b.AddComment(b.ctx, repo, prNum, ":robot: RevendorBot done :hourglass: "+dur.String())
		if err != nil {
			return errors.Wrap(err, "errored trying to add a comment")
		}
	default:
		log.Printf("ignoring %T event", e)
		return nil
	}

	return err
}

func filterComment(event *github.IssueCommentEvent) bool {
	if event.GetAction() != "created" {
		return false
	}
	if event.GetComment() == nil || strings.TrimSpace(event.GetComment().GetBody()) != "/revendor" {
		return false
	}
	if !event.GetIssue().IsPullRequest() {
		return false
	}
	return true
}

func (b *Bot) handleComment(event *github.IssueCommentEvent) error {
	repo := event.GetRepo()

	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()

	commentID := event.GetComment().GetID()

	ctx := context.Background()
	// first delete the existing comment to show that we're handling it
	err := b.deleteComment(ctx, owner, repoName, commentID)
	if err != nil {
		return err
	}

	number := event.GetIssue().GetNumber()
	err = b.AddComment(ctx, repo, number, ":robot: RevendorBot says `go.mod` or `go.sum` modified, may need to revendor.")
	if err != nil {
		return err
	}
	log.Print("Added comment")

	pr, resp, err := b.ghclient.PullRequests.Get(ctx, owner, repoName, number)
	if err != nil {
		return errors.Wrap(err, "failed to get PR")
	}
	if resp.StatusCode > 299 {
		return errors.Errorf("got status %d when getting PR (%s)", resp.StatusCode, resp.Status)
	}

	ref := pr.GetHead().GetRef()
	log.Printf("HEAD is %s", ref)

	if !b.revendorRequired(ctx, repo, ref) {
		log.Print("no need to revendor")
		return b.AddComment(ctx, repo, number, ":robot: RevendorBot doesn't need to revendor! :beach_umbrella:")
	}

	return b.revendor(repo, ref)
}

func (b *Bot) deleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	resp, err := b.ghclient.Issues.DeleteComment(ctx, owner, repo, commentID)
	if err != nil {
		return errors.Wrap(err, "failed to delete comment")
	}
	if resp.StatusCode > 299 {
		return errors.Errorf("got status %d when deleting comment (%s)", resp.StatusCode, resp.Status)
	}
	return nil
}

// AddComment - comments on the issue
func (b *Bot) AddComment(ctx context.Context, repo Repo, number int, body string) error {
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()

	// even though it's a PR, we need an IssueComment. PullRequestComments are for reviews explicitly.
	comment := &github.IssueComment{
		Body: github.String(body),
	}
	_, resp, err := b.ghclient.Issues.CreateComment(ctx, owner, repoName, number, comment)
	if err != nil {
		return errors.Wrap(err, "failed to create comment")
	}
	if resp.StatusCode > 299 {
		return errors.Errorf("got status %d when creating comment (%s)", resp.StatusCode, resp.Status)
	}
	return nil
}

func (b *Bot) handlePush(event *github.PushEvent) error {
	log.Printf("handling push (%d commits)", len(event.Commits))

	ref := event.GetRef()
	if !b.revendorRequired(b.ctx, event.GetRepo(), ref) {
		log.Print("no need to revendor")
		return nil
	}

	return b.revendor(event.GetRepo(), event.GetRef())
}

func (b *Bot) revendor(repo Repo, ref string) error {
	log.Print("revendoring...")
	// we need a new context here since the main context will last longer (maybe?)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	dir, cleanup, err := b.clone(ctx, repo, ref)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}

	cmd := newCmd(ctx, dir, "go", "mod", "tidy", "-v")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=vendor", "GO111MODULE=on")
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to run command")
	}

	cmd = newCmd(ctx, dir, "go", "mod", "vendor")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=vendor", "GO111MODULE=on")
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to run command")
	}

	cmd = newCmd(ctx, dir, "git", "status", "--porcelain=v2")
	statusBuf := &bytes.Buffer{}
	cmd.Stdout = statusBuf
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "git status failed")
	}
	if statusBuf.String() == "" {
		log.Print("no changes necessary")
		return nil
	}
	log.Print("repo is dirty - will need to commit")

	cmd = newCmd(ctx, dir, "git", "add", ".")
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "git add failed")
	}

	cmd = newCmd(ctx, dir, "git", "commit", "-S", "-sm", "updating results of `go mod tidy` and `go mod vendor`")
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "git commit failed")
	}

	cmd = newCmd(ctx, dir, "git", "push")
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "git push failed")
	}
	return nil
}

func newCmd(ctx context.Context, dir, command string, arg ...string) *exec.Cmd {
	// nolint: gosec
	cmd := exec.CommandContext(ctx, command, arg...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}

func (b *Bot) revendorRequired(ctx context.Context, repo Repo, ref string) bool {
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()

	commit, resp, err := b.ghclient.Repositories.GetCommit(ctx, owner, repoName, ref)
	if err != nil {
		log.Printf("failed to get commit, ignoring: %v", err)
		return false
	}
	if resp.StatusCode > 299 {
		log.Printf("failed to get commit - got status %s", resp.Status)
		return false
	}
	for _, f := range commit.Files {
		if f.GetFilename() == "go.mod" || f.GetFilename() == "go.sum" {
			return true
		}
	}
	return false
}

func (b *Bot) clone(ctx context.Context, repo Repo, ref string) (string, func(), error) {
	tmpDir, err := ioutil.TempDir("", "revendor-")
	if err != nil {
		return tmpDir, nil, errors.Wrap(err, "failed to create temp dir")
	}
	cleanup := func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			log.Printf("failed to delete temp dir - %v", err)
		}
	}

	repoName := repo.GetName()
	owner := repo.GetOwner().GetLogin()
	dir := filepath.Join(tmpDir, owner, repoName)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return dir, cleanup, errors.Wrap(err, "failed to create repo dir")
	}

	log.Print("Running commands...")
	cmd := newCmd(ctx, dir, "git", "clone", repo.GetCloneURL(), ".")
	err = cmd.Run()
	if err != nil {
		return dir, cleanup, errors.Wrap(err, "failed to clone")
	}

	ref = strings.Replace(ref, "refs/heads/", "", 1)
	cmd = newCmd(ctx, dir, "git", "checkout", ref)
	err = cmd.Run()
	if err != nil {
		return dir, cleanup, errors.Wrap(err, "failed to checkout")
	}
	return dir, cleanup, nil
}

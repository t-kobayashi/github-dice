package main

import (
	"errors"
	"github.com/google/go-github/github"
	//"github.com/k0kubun/pp"
	"golang.org/x/oauth2"
	"os"
)

type PullRequestManager struct {
	client       *(github.Client)
	organization string
	team         string
	repository   string
	dryRun       bool
}

type Users []*github.User

func NewPullRequestManager(dryRun bool) *PullRequestManager {
	p := &PullRequestManager{}
	p.organization = os.Getenv("GITHUB_ORGANIZATION")
	p.repository = os.Getenv("GITHUB_REPO")
	p.team = os.Getenv("GITHUB_TEAM")
	p.dryRun = dryRun
	token := os.Getenv("GITHUB_ACCESS_TOKEN")

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	p.client = github.NewClient(tc)

	return p
}

func (p *PullRequestManager) FindPullRequests(state string) ([]*github.PullRequest, error) {
	options := &github.PullRequestListOptions{State: state}
	pullRequests, _, err := p.client.PullRequests.List(p.organization, p.repository, options)

	members, err := p.findUsersByTeamName(p.team)
	if err != nil {
		return nil, err
	}

	var targets []*github.PullRequest
	for _, pr := range pullRequests {
		for _, member := range members {
			if *pr.User.Login == *member.Login {
				targets = append(targets, pr)
			}
		}
	}

	return targets, nil
}

func (p *PullRequestManager) FindCandidatesOfReviewer(pr *github.PullRequest) ([]*github.User, error) {
	var users Users
	users, err := p.findUsersByTeamName(p.team)
	if err != nil {
		return nil, err
	}
	candidates := users.removeUser(pr.User)

	return candidates, nil
}

func (p *PullRequestManager) AssignUser(pr *github.PullRequest, user *github.User) bool {
	if p.dryRun {
		return true
	}
	asignees := []string{*user.Login}
	_, _, err := p.client.Issues.AddAssignees(p.organization, p.repository, *pr.Number, asignees)
	if err != nil {
		return false
	}
	return true
}

func (p *PullRequestManager) AssignAuthor(pr *github.PullRequest) bool {
	return p.AssignUser(pr, pr.User)
}

func (p *PullRequestManager) IsAlreadyAssignedExpectAuthor(pr *github.PullRequest) bool {
	var assignees Users
	issue, _, _ := p.client.Issues.Get(p.organization, p.repository, *pr.Number)
	assignees = issue.Assignees
	assineesExpectAuthor := assignees.removeUser(pr.User)

	return len(assineesExpectAuthor) > 0
}

func (p *PullRequestManager) Comment(pr *github.PullRequest, comment string) bool {
	ic := &github.IssueComment{Body: &comment}
	if p.dryRun {
		return true
	}
	_, _, err := p.client.Issues.CreateComment(p.organization, p.repository, *pr.Number, ic)

	return err != nil
}

func (p *PullRequestManager) findUsersByTeamName(name string) ([]*github.User, error) {
	teams, _, err := p.client.Repositories.ListTeams(p.organization, p.repository, nil)
	if err != nil {
		return nil, err
	}

	for _, t := range teams {
		if *t.Name == name {
			users, _, err := p.client.Organizations.ListTeamMembers(*t.ID, &github.OrganizationListTeamMembersOptions{})
			return users, err
		}
	}

	return nil, errors.New("team not found")
}

func (users Users) removeUser(user *github.User) []*github.User {
	var candidates []*github.User
	for _, u := range users {
		if *u.Login != *user.Login {
			candidates = append(candidates, u)
		}
	}
	return candidates
}

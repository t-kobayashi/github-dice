package main

import (
	"errors"
	"github.com/google/go-github/github"
	"regexp"
	"strings"
)

type IssueManager struct {
	Client       *github.Client
	Organization string
	Team         string
	Repository   string
	DryRun       bool
}

type Users []*github.User

func NewIssueManager(client *github.Client, organization string, repository string, team string, dryRun bool) *IssueManager {
	im := &IssueManager{}
	im.Client = client
	im.Organization = organization
	im.Repository = repository
	im.DryRun = dryRun
	im.Team = team

	return im
}

func (im *IssueManager) FindIssues(spec string) ([]github.Issue, error) {
	members, err := im.findUsersByTeamName(im.Team)
	if err != nil {
		return nil, err
	}
	queryString := im.buildQuery(spec)
	searchResult, _, err := im.Client.Search.Issues(queryString, &github.SearchOptions{})
	if err != nil {
		return nil, err
	}

	var targets []github.Issue
Loop:
	for _, issue := range searchResult.Issues {
		for _, member := range members {
			if *issue.User.Login == *member.Login {
				targets = append(targets, issue)
				continue Loop
			}
		}
	}

	return targets, nil
}

func (im *IssueManager) FindCandidatesOfReviewer(issue *github.Issue) ([]*github.User, error) {
	var users Users
	users, err := im.findUsersByTeamName(im.Team)
	if err != nil {
		return nil, err
	}
	candidates := users.removeUser(issue.User)

	return candidates, nil
}

func (im *IssueManager) AssignUser(issue *github.Issue, user *github.User) bool {
	if im.DryRun {
		return true
	}
	asignees := []string{*user.Login}
	_, _, err := im.Client.Issues.AddAssignees(im.Organization, im.Repository, *issue.Number, asignees)
	if err != nil {
		return false
	}
	return true
}

func (im *IssueManager) AssignAuthor(issue *github.Issue) bool {
	return im.AssignUser(issue, issue.User)
}

func (im *IssueManager) IsAlreadyAssignedExpectAuthor(issue *github.Issue) bool {
	var assignees Users
	assignees = issue.Assignees
	assineesExpectAuthor := assignees.removeUser(issue.User)

	return len(assineesExpectAuthor) > 0
}

func (im *IssueManager) Comment(issue *github.Issue, comment string) bool {
	ic := &github.IssueComment{Body: &comment}
	if im.DryRun {
		return true
	}
	_, _, err := im.Client.Issues.CreateComment(im.Organization, im.Repository, *issue.Number, ic)

	return err != nil
}

func (im *IssueManager) findUsersByTeamName(name string) ([]*github.User, error) {
	teams, _, err := im.Client.Repositories.ListTeams(im.Organization, im.Repository, nil)
	if err != nil {
		return nil, err
	}

	for _, t := range teams {
		if *t.Name == name {
			users, _, err := im.Client.Organizations.ListTeamMembers(*t.ID, &github.OrganizationListTeamMembersOptions{})
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

func (im *IssueManager) buildQuery(spec string) string {
	queries := regexp.MustCompile(" +").Split(spec, -1)
	queries = append(queries, "repo:"+im.Organization+"/"+im.Repository)
	return strings.Join(queries, " ")
}

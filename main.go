package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

type Dice struct {
	Opts    Options
	Setting map[string]string
}

type Options struct {
	Query   string `short:"q" long:"query" default:"type:pr is:open" description:"query strings for search issue/pull-request."`
	DryRun  bool   `short:"n" long:"dry-run" description:"show candidates and list issues, without assign."`
	Force   bool   `short:"f" long:"force" description:"if true, reassign even if already assigned."`
	RunOnce bool   `short:"o" long:"run-once" description:"if true, assign just once issue."`
	Debug   bool   `short:"d" long:"debug"`
	Limit   int    `short:"l" long:"limit" default:"0" description:"maximum number of issues per running command."`
}

func (d *Dice) initialize(args []string) {
	var err error
	d.Setting, err = godotenv.Read()
	if err != nil {
		d.log(err.Error())
		os.Exit(1)
	}
	p := flags.NewParser(&d.Opts, flags.Default)

	_, err = p.ParseArgs(args)
	if err != nil {
		d.log(err.Error())
		os.Exit(1)
	}
}

func (d *Dice) run() {
	client := d.createClient()
	im := NewIssueManager(client, d.Setting["GITHUB_ORGANIZATION"], d.Setting["GITHUB_REPO"], d.Setting["GITHUB_TEAM"], d.Opts.DryRun)
	issues, err := im.FindIssues(d.Opts.Query)
	if err != nil {
		d.log(err.Error())
		os.Exit(1)
	}

	candidates := []*github.User{}
	assinedNumber := 0
	for _, issue := range issues {
		im.AssignAuthor(&issue)
		if im.IsAlreadyAssignedExpectAuthor(&issue) {
			continue
		}
		candidates, err = im.FindCandidatesOfReviewer(&issue)
		if err != nil {
			d.log(err.Error())
			os.Exit(1)
		}
		assignee := d.throw(candidates)
		im.AssignUser(&issue, assignee)
		im.Comment(&issue, ":game_die: @"+*assignee.Login)
		d.log(fmt.Sprintf("#%d %s %s => author:%s assigned:%s", *issue.Number, *issue.HTMLURL, *issue.Title, *issue.User.Login, *assignee.Login))
		assinedNumber++
		if d.Opts.Limit > 0 && d.Opts.Limit <= assinedNumber {
			break
		}
	}
}

func (d *Dice) createClient() *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: d.Setting["GITHUB_ACCESS_TOKEN"]},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return github.NewClient(tc)
}

func (d *Dice) throw(candidates []*github.User) *github.User {
	exemptions := d.Setting["DICE_EXEMPTIONS"]

	var act []*github.User
Loop:
	for _, c := range candidates {
		for _, ex := range strings.Split(exemptions, ",") {
			if ex == *c.Login {
				continue Loop
			}
		}
		act = append(act, c)
	}
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(act))

	return act[i]
}

func (d *Dice) log(str string) {
	log.Println(str)
}

func main() {
	d := &Dice{}
	d.initialize(os.Args[1:])
	d.run()
}

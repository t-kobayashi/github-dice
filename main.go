package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	//"github.com/k0kubun/pp"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

type Dice struct {
	Opts Options
}

type Options struct {
	DryRun  bool `short:"n" long:"dry-run" description:"show candidates and list issues, without assign."`
	Force   bool `short:"f" long:"force" description:"if true, reassign even if already assigned."`
	RunOnce bool `short:"o" long:"run-once" description:"if true, assign just once issue."`
	Debug   bool `short:"d" long:"debug"`
	Limit   int  `short:"l" long:"limit" default:"0" description:"maximum number of issues per running command."`
}

func (d *Dice) initialize(args []string) {
	godotenv.Load()

	p := flags.NewParser(&d.Opts, flags.Default)

	_, err := p.ParseArgs(args)
	if err != nil {
		os.Exit(1)
	}
}

func (d *Dice) run() {
	p := NewPullRequestManager(d.Opts.DryRun)

	pullRequests, err := p.FindPullRequests("open")
	if err != nil {
		d.log(err.Error())
		os.Exit(1)
	}

	var candidates []*github.User
	assinedNumber := 0
	for _, pr := range pullRequests {
		p.AssignAuthor(pr)
		if p.IsAlreadyAssignedExpectAuthor(pr) {
			continue
		}
		candidates, err = p.FindCandidatesOfReviewer(pr)
		if err != nil {
			d.log(err.Error())
			os.Exit(1)
		}
		assignee := d.throw(candidates)
		p.AssignUser(pr, assignee)
		p.Comment(pr, ":game_die: @"+*assignee.Login)
		d.log(fmt.Sprintf("#%d %s => author:%s assigned:%s", *pr.Number, *pr.Title, *pr.User.Login, *assignee.Login))
		assinedNumber++
		if d.Opts.Limit > 0 && d.Opts.Limit <= assinedNumber {
			break
		}
	}
}

func (d *Dice) throw(candidates []*github.User) *github.User {
	exemptions := os.Getenv("DICE_EXEMPTIONS")

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

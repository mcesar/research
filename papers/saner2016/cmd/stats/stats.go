package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"../../structs"
)

type stats struct {
	Commits            int
	Features           int
	Features3          int
	Issues             int
	Issues3            int
	Files              map[string]int
	LayersPerCommit    map[int]int
	UsersPerIssue      map[int]int
	CommitsPerIssue    map[int]int
	LayersPerIssue     map[int]int
	UsersPerIssue3     map[int]int
	CommitsPerIssue3   map[int]int
	LayersPerIssue3    map[int]int
	UsersPerFeature    map[int]int
	CommitsPerFeature  map[int]int
	LayersPerFeature   map[int]int
	IssuesPerFeature   map[int]int
	UsersPerFeature3   map[int]int
	CommitsPerFeature3 map[int]int
	LayersPerFeature3  map[int]int
	IssuesPerFeature3  map[int]int
}

type issue struct {
	commits int
	files   int
	layers  map[string]int
	users   map[string]int
}

type feature struct {
	commits int
	files   int
	issues  map[string]int
	layers  map[string]int
	users   map[string]int
}

func main() {
	issueKind := flag.String("k", "", "issue kind")
	flag.Parse()
	file, err := os.Open(os.Args[len(os.Args)-1])
	if err != nil {
		log.Fatal("Error opening file: ", os.Args[len(os.Args)-1], err)
	}
	commits := []*structs.Commit{}
	json.NewDecoder(file).Decode(&commits)
	stats := stats{
		Commits:            0,
		Files:              map[string]int{},
		LayersPerCommit:    map[int]int{},
		UsersPerIssue:      map[int]int{},
		CommitsPerIssue:    map[int]int{},
		LayersPerIssue:     map[int]int{},
		UsersPerIssue3:     map[int]int{},
		CommitsPerIssue3:   map[int]int{},
		LayersPerIssue3:    map[int]int{},
		UsersPerFeature:    map[int]int{},
		CommitsPerFeature:  map[int]int{},
		LayersPerFeature:   map[int]int{},
		IssuesPerFeature:   map[int]int{},
		UsersPerFeature3:   map[int]int{},
		CommitsPerFeature3: map[int]int{},
		LayersPerFeature3:  map[int]int{},
		IssuesPerFeature3:  map[int]int{}}
	features := map[string]*feature{}
	issues := map[string]*issue{}
	for _, commit := range commits {
		if *issueKind != "" && commit.Issue.Kind != *issueKind {
			continue
		}
		stats.Commits++
		if f, ok := features[commit.Feature]; ok {
			f.commits += 1
		} else {
			f = &feature{commits: 1, issues: map[string]int{},
				layers: map[string]int{}, users: map[string]int{}}
			features[commit.Feature] = f
		}
		if i, ok := issues[commit.Issue.Id]; ok {
			i.commits += 1
		} else {
			i = &issue{commits: 1, layers: map[string]int{}, users: map[string]int{}}
			issues[commit.Issue.Id] = i
		}
		features[commit.Feature].issues[commit.Issue.Id] = 0
		features[commit.Feature].users[commit.Change.Author] = 0
		issues[commit.Issue.Id].users[commit.Change.Author] = 0
		layers := map[string]int{}
		count := 0
		for _, file := range commit.Files {
			layer := strings.Split(file, "/")[1]
			if layer == "siop-jpa" || layer == "siop-ejb" || layer == "siop-war" {
				count++
				layers[layer] = 0
				features[commit.Feature].layers[layer] = 0
				features[commit.Feature].files += 1
				issues[commit.Issue.Id].layers[layer] = 0
				issues[commit.Issue.Id].files += 1
				if n, ok := stats.Files[layer]; ok {
					stats.Files[layer] = n + 1
				} else {
					stats.Files[layer] = 1
				}
			}
		}
		increment(stats.LayersPerCommit, len(layers))
	}
	features3 := 0
	issues3 := 0
	for _, f := range features {
		increment(stats.CommitsPerFeature, f.commits)
		increment(stats.UsersPerFeature, len(f.users))
		increment(stats.LayersPerFeature, len(f.layers))
		increment(stats.IssuesPerFeature, len(f.issues))
		if f.files >= 3 {
			features3++
			increment(stats.CommitsPerFeature3, f.commits)
			increment(stats.UsersPerFeature3, len(f.users))
			increment(stats.LayersPerFeature3, len(f.layers))
			increment(stats.IssuesPerFeature3, len(f.issues))
		}
	}
	for _, i := range issues {
		increment(stats.CommitsPerIssue, i.commits)
		increment(stats.UsersPerIssue, len(i.users))
		increment(stats.LayersPerIssue, len(i.layers))
		if i.files > 3 {
			issues3++
			increment(stats.CommitsPerIssue3, i.commits)
			increment(stats.UsersPerIssue3, len(i.users))
			increment(stats.LayersPerIssue3, len(i.layers))
		}
	}
	stats.Features = len(features)
	stats.Features3 = features3
	stats.Issues = len(issues)
	stats.Issues3 = issues3
	fmt.Printf("%#v\n", stats)
}

func increment(m map[int]int, key int) {
	if n, ok := m[key]; ok {
		m[key] = n + 1
	} else {
		m[key] = 1
	}
}

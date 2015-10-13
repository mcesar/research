package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"../../structs"
)

type stats struct {
	Commits                    int
	Features                   int
	Issues                     int
	Files                      map[string]int
	CommitsPerLayerCombination map[string]int
	LayersPerCommit            map[int]int
	UsersPerIssue              map[int]int
	CommitsPerIssue            map[int]int
	LayersPerIssue             map[int]int
	UsersPerFeature            map[int]int
	CommitsPerFeature          map[int]int
	LayersPerFeature           map[int]int
	IssuesPerFeature           map[int]int
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
	minimumFileCount := flag.Int("n", 0, "minimum file count")
	flag.Parse()
	file, err := os.Open(os.Args[len(os.Args)-1])
	if err != nil {
		log.Fatal("Error opening file: ", os.Args[len(os.Args)-1], err)
	}
	commits := []*structs.Commit{}
	json.NewDecoder(file).Decode(&commits)
	stats := stats{
		Commits: 0,
		Files:   map[string]int{},
		CommitsPerLayerCombination: map[string]int{},
		LayersPerCommit:            map[int]int{},
		UsersPerIssue:              map[int]int{},
		CommitsPerIssue:            map[int]int{},
		LayersPerIssue:             map[int]int{},
		UsersPerFeature:            map[int]int{},
		CommitsPerFeature:          map[int]int{},
		LayersPerFeature:           map[int]int{},
		IssuesPerFeature:           map[int]int{}}
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
				incrementS(stats.Files, layer)
			}
		}
		increment(stats.LayersPerCommit, len(layers))
		incrementS(stats.CommitsPerLayerCombination, combination(layers))
	}
	featuresCount := 0
	issuesCount := 0
	for _, f := range features {
		if *minimumFileCount == 0 || f.files >= *minimumFileCount {
			featuresCount++
			increment(stats.CommitsPerFeature, f.commits)
			increment(stats.UsersPerFeature, len(f.users))
			increment(stats.LayersPerFeature, len(f.layers))
			increment(stats.IssuesPerFeature, len(f.issues))
		}
	}
	for _, i := range issues {
		if *minimumFileCount == 0 || i.files >= *minimumFileCount {
			issuesCount++
			increment(stats.CommitsPerIssue, i.commits)
			increment(stats.UsersPerIssue, len(i.users))
			increment(stats.LayersPerIssue, len(i.layers))
		}
	}
	stats.Features = featuresCount
	stats.Issues = issuesCount
	out := fmt.Sprintf("%+v", stats)
	re := regexp.MustCompile(" ([a-zA-Z]{4,}\\:)")
	fmt.Println(re.ReplaceAllString(out, "\n$1 "))
}

func increment(m map[int]int, key int) {
	if n, ok := m[key]; ok {
		m[key] = n + 1
	} else {
		m[key] = 1
	}
}

func incrementS(m map[string]int, key string) {
	if n, ok := m[key]; ok {
		m[key] = n + 1
	} else {
		m[key] = 1
	}
}

func combination(layers map[string]int) string {
	_, m := layers["siop-jpa"]
	_, v := layers["siop-war"]
	_, c := layers["siop-ejb"]
	if m && v && c {
		return "mvc"
	}
	if m && v {
		return "mv"
	}
	if m && c {
		return "mc"
	}
	if v && c {
		return "vc"
	}
	if m {
		return "m"
	}
	if v {
		return "v"
	}
	if c {
		return "c"
	}
	return ""
}

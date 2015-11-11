package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"../../lib"
)

type stats struct {
	Commits                     int
	CommitsWithIssues           int
	Features                    int
	Issues                      int
	Files                       map[string]int
	CommitsPerLayerCombination  map[string]int
	LayersPerCommit             map[int]int
	UsersPerIssue               map[int]int
	CommitsPerIssue             map[int]int
	LayersPerIssue              map[int]int
	IssuesPerLayerCombination   map[string]int
	UsersPerFeature             map[int]int
	CommitsPerFeature           map[int]int
	LayersPerFeature            map[int]int
	IssuesPerFeature            map[int]int
	FeaturesPerLayerCombination map[string]int
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
	repository := flag.String("r", "siop", "repository")
	issueKind := flag.String("k", "", "issue kind")
	minimumFileCount := flag.Int("n", 0, "minimum file count")
	commitsWithIssuesOnly := flag.Bool("i", false, "commits with issues only")
	flag.Parse()
	f := lib.CommitsFunctions(*repository)
	commits, err := f.Commits(os.Args, f.IssueExtractor)
	if err != nil {
		log.Fatal(err)
	}
	stats := stats{
		Commits:           0,
		CommitsWithIssues: 0,
		Files:             map[string]int{},
		CommitsPerLayerCombination:  map[string]int{},
		LayersPerCommit:             map[int]int{},
		UsersPerIssue:               map[int]int{},
		CommitsPerIssue:             map[int]int{},
		LayersPerIssue:              map[int]int{},
		IssuesPerLayerCombination:   map[string]int{},
		UsersPerFeature:             map[int]int{},
		CommitsPerFeature:           map[int]int{},
		LayersPerFeature:            map[int]int{},
		IssuesPerFeature:            map[int]int{},
		FeaturesPerLayerCombination: map[string]int{}}
	features := map[string]*feature{}
	issues := map[string]*issue{}
	kinds := map[string]int{}
	commitsFromKayiwa := map[string]string{}
	for _, commit := range commits {
		if *issueKind != "" && commit.Issue.Kind != *issueKind {
			continue
		}
		if *commitsWithIssuesOnly && commit.Issue.Id == "" {
			continue
		}
		kinds[commit.Issue.Kind]++
		stats.Commits++
		if commit.Issue.Id != "" {
			stats.CommitsWithIssues++
		}
		if f, ok := features[commit.Feature]; ok {
			f.commits++
		} else {
			f = &feature{commits: 1, issues: map[string]int{},
				layers: map[string]int{}, users: map[string]int{}}
			features[commit.Feature] = f
		}
		if i, ok := issues[commit.Issue.Id]; ok {
			i.commits++
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
			layer := f.LayerExtractor(file)
			if layer != "" {
				count++
				layers[layer] = 0
				features[commit.Feature].layers[layer] = 0
				features[commit.Feature].files += 1
				issues[commit.Issue.Id].layers[layer] = 0
				issues[commit.Issue.Id].files += 1
				incrementS(stats.Files, layer)
			}
			if strings.Contains(strings.ToLower(commit.Change.Author), "kayiwa") {
				commitsFromKayiwa[commit.Issue.Id] += file + " "
			}
		}
		increment(stats.LayersPerCommit, len(layers))
		incrementS(stats.CommitsPerLayerCombination, combination(layers))
	}
	featuresCount := 0
	issuesCount := 0
	for _ /*fk*/, f := range features {
		if *minimumFileCount == 0 || f.files >= *minimumFileCount {
			featuresCount++
			increment(stats.CommitsPerFeature, f.commits)
			increment(stats.UsersPerFeature, len(f.users))
			increment(stats.LayersPerFeature, len(f.layers))
			increment(stats.IssuesPerFeature, len(f.issues))
			incrementS(stats.FeaturesPerLayerCombination, combination(f.layers))
		}
	}
	for k, i := range issues {
		if *minimumFileCount == 0 || i.files >= *minimumFileCount {
			issuesCount++
			increment(stats.CommitsPerIssue, i.commits)
			increment(stats.UsersPerIssue, len(i.users))
			increment(stats.LayersPerIssue, len(i.layers))
			incrementS(stats.IssuesPerLayerCombination, combination(i.layers))
			if i.commits > 1000 {
				fmt.Println(k, i)
			}
			if len(i.users) == 2 {
				if _ /*v*/, ok := commitsFromKayiwa[k]; ok {
					//fmt.Println(k, "==>", v)
				}
			}
		}
	}
	stats.Features = featuresCount
	stats.Issues = issuesCount
	out := fmt.Sprintf("%+v", stats)
	re := regexp.MustCompile(" ([a-zA-Z]{4,}\\:)")
	fmt.Println(re.ReplaceAllString(out, "\n$1 "))
	fmt.Println(kinds)
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
	_, m := layers["m"]
	_, v := layers["v"]
	_, c := layers["c"]
	switch {
	case m && v && c:
		return "mvc"
	case m && v:
		return "mv"
	case m && c:
		return "mc"
	case v && c:
		return "vc"
	case m:
		return "m"
	case v:
		return "v"
	case c:
		return "c"
	default:
		return ""
	}
}

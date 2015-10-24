package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"../../structs"
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

var mapCommitsFunctions = map[string]struct {
	commits        func([]string, func(string) string) ([]*structs.Commit, error)
	issueExtractor func(string) string
	layerExtractor func(string) string
}{
	"siop": {
		commits:        commitsFromSiop,
		layerExtractor: siopLayerExtractor},
	"ofbiz": {
		commits:        commitsFromGitAndJira,
		issueExtractor: ofbizIssueExtractor,
		layerExtractor: ofbizLayerExtractor},
	"openmrs": {
		commits:        commitsFromGitAndJira,
		issueExtractor: openmrsIssueExtractor,
		layerExtractor: openmrsLayerExtractor},
}

func main() {
	repository := flag.String("r", "siop", "repository")
	issueKind := flag.String("k", "", "issue kind")
	minimumFileCount := flag.Int("n", 0, "minimum file count")
	commitsWithIssuesOnly := flag.Bool("i", false, "commits with issues only")
	flag.Parse()
	f := mapCommitsFunctions[*repository]
	commits, err := f.commits(os.Args, f.issueExtractor)
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
			layer := f.layerExtractor(file)
			if layer != "" {
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
			incrementS(stats.FeaturesPerLayerCombination, combination(f.layers))
		}
	}
	for _, i := range issues {
		if *minimumFileCount == 0 || i.files >= *minimumFileCount {
			issuesCount++
			increment(stats.CommitsPerIssue, i.commits)
			increment(stats.UsersPerIssue, len(i.users))
			increment(stats.LayersPerIssue, len(i.layers))
			incrementS(stats.IssuesPerLayerCombination, combination(i.layers))
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

func commitsFromSiop(args []string, _ func(string) string) ([]*structs.Commit, error) {
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("usage: stats <commits file>")
	}
	file, err := os.Open(args[len(args)-1])
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v %v", args[len(args)-1], err)
	}
	commits := []*structs.Commit{}
	json.NewDecoder(file).Decode(&commits)
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func commitsFromGitAndJira(args []string,
	issueExtractor func(string) string) ([]*structs.Commit, error) {
	if len(os.Args) < 5 {
		return nil, fmt.Errorf("usage: stats <git repo> <issues file>")
	}
	issues, err := os.Open(args[len(args)-1])
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(issues)
	issuesMap := map[string]string{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil && err != io.EOF {
			log.Fatal("Error reading issues file ", err)
		}
		issuesMap[record[0]] = record[1]
	}
	cmd := exec.Command("git", "--no-pager", "log", "--date=iso", "--reverse",
		"--pretty=format:%H%x09%an%x09%ad%x09%s")
	cmd.Dir = args[len(args)-2]
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	commits := []*structs.Commit{}
	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		arr := strings.Split(scan.Text(), "\t")
		issue := issueExtractor(arr[3])
		kind := issuesMap[issue]
		if kind != "Bug" {
			kind = "Improvement"
		}
		cmdTree := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", arr[0])
		cmdTree.Dir = args[len(args)-2]
		outTree, err := cmdTree.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, string(outTree))
			return nil, err
		}
		files := strings.Split(string(outTree), "\n")
		if len(files) > 0 && files[len(files)-1] == "" {
			files = files[:len(files)-1]
		}
		commit := &structs.Commit{
			Change: &structs.Change{
				Uuid:     arr[0],
				Author:   arr[1],
				Comment:  arr[3],
				Modified: arr[2],
			},
			Issue: structs.Issue{Id: issue, Kind: kind},
			Files: files,
		}
		commits = append(commits, commit)
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("wait %v", err)
	}
	return commits, nil
}

var ofbizRegex *regexp.Regexp

func ofbizIssueExtractor(description string) string {
	if ofbizRegex == nil {
		var err error
		ofbizRegex, err = regexp.Compile("OFBIZ-\\d+")
		if err != nil {
			log.Fatal(err)
		}
	}
	return ofbizRegex.FindString(description)
}

var openmrsRegex *regexp.Regexp

func openmrsIssueExtractor(description string) string {
	if openmrsRegex == nil {
		var err error
		openmrsRegex, err = regexp.Compile("TRUNK-\\d+")
		if err != nil {
			log.Fatal(err)
		}
	}
	return openmrsRegex.FindString(description)
}

func siopLayerExtractor(file string) string {
	layer := strings.Split(file, "/")[1]
	switch layer {
	case "siop-jpa":
		return "m"
	case "siop-ejb":
		return "c"
	case "siop-war":
		return "v"
	default:
		return ""
	}
}

func ofbizLayerExtractor(file string) string {
	arr := strings.Split(file, "/")
	if len(arr) < 3 {
		return "c"
	}
	if (arr[0] == "applications" || arr[0] == "specialpurpose" || arr[0] == "framework") &&
		(arr[2] == "data" || arr[2] == "entitydef" || arr[2] == "entityext" ||
			arr[2] == "datafile") {
		return "m"
	}
	if (arr[0] == "applications" || arr[0] == "specialpurpose" || arr[0] == "framework") &&
		(arr[2] == "config" || arr[2] == "webapp" || arr[2] == "widget" || arr[2] == "webtools") {
		return "v"
	}
	return "c"
}

func openmrsLayerExtractor(file string) string {
	arr := strings.Split(file, "/")
	//if len(arr) < 2 {
	//	return "c"
	//}
	if strings.HasPrefix(file, "api/src/main/java/org/openmrs") && arr[len(arr)-2] == "openmrs" ||
		strings.HasPrefix(file, "api/src/main/java/org/openmrs/api/db") ||
		strings.HasPrefix(file, "api/src/main/java/org/openmrs/api/handler") ||
		strings.HasPrefix(file, "api/src/main/resources") {
		return "m"
	}
	if strings.HasPrefix(file, "api/src/main/java/org/openmrs") {
		return "m"
	}
	if arr[0] == "web" || arr[0] == "webapp" {
		return "v"
	}
	return "c"
}

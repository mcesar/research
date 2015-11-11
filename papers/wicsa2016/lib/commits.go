package lib

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Functions struct {
	Commits        func([]string, func(string) string) ([]*Commit, error)
	IssueExtractor func(string) string
	LayerExtractor func(string) string
}

var mapCommitsFunctions = map[string]Functions{
	"siop": {
		Commits:        commitsFromSiop,
		LayerExtractor: siopLayerExtractor},
	"ofbiz": {
		Commits:        commitsFromGitAndJira,
		IssueExtractor: ofbizIssueExtractor,
		LayerExtractor: ofbizLayerExtractor},
	"openmrs": {
		Commits:        commitsFromGitAndJira,
		IssueExtractor: openmrsIssueExtractor,
		LayerExtractor: openmrsLayerExtractor},
}

func CommitsFunctions(key string) Functions {
	return mapCommitsFunctions[key]
}

func commitsFromSiop(args []string, _ func(string) string) ([]*Commit, error) {
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("usage: stats <commits file>")
	}
	file, err := os.Open(args[len(args)-1])
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v %v", args[len(args)-1], err)
	}
	commits := []*Commit{}
	json.NewDecoder(file).Decode(&commits)
	err = file.Close()
	if err != nil {
		return nil, err
	}
	for _, c := range commits {
		modified, err := time.Parse("02/01/2006 15:04", c.Change.Modified)
		if err != nil {
			return nil, err
		}
		c.Change.ModifiedTime = modified
	}
	return commits, nil
}

func commitsFromGitAndJira(args []string,
	issueExtractor func(string) string) ([]*Commit, error) {
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
	commits := []*Commit{}
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
		modified, err := time.Parse("2006-01-02 15:04:05 -0700", arr[2])
		if err != nil {
			return nil, err
		}
		commit := &Commit{
			Change: &Change{
				Uuid:         arr[0],
				Author:       arr[1],
				Comment:      arr[3],
				Modified:     arr[2],
				ModifiedTime: modified,
			},
			Issue: Issue{Id: issue, Kind: kind},
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

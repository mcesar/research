package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"../../structs"
)

func main() {
	folder := open(os.Args[1])
	defer folder.Close()
	fileNames, err := folder.Readdirnames(0)
	if err != nil {
		log.Fatal("Error reading file names from ", os.Args[1], err)
	}
	changesets := map[string]*structs.Change{}
	changesetsByUuid := map[string]*structs.Change{}
	months := map[string]string{"jan": "01", "fev": "02", "mar": "03", "abr": "04", "mai": "05",
		"jun": "06", "jul": "07", "ago": "08", "set": "09", "out": "10", "nov": "11", "dez": "12"}
	monthsInEnglish := map[string]string{"jan": "January", "fev": "February", "mar": "March",
		"abr": "April", "mai": "May", "jun": "June", "jul": "July", "ago": "August",
		"set": "September", "out": "October", "nov": "November", "dez": "December"}
	hours := map[string]string{"01": "13", "02": "14", "03": "15", "04": "16", "05": "17",
		"06": "18", "07": "19", "08": "20", "09": "21", "10": "22", "11": "23", "12": "12"}
	for _, fileName := range fileNames {
		if !strings.HasSuffix(fileName, ".json") {
			continue
		}
		j := open(filepath.Join(os.Args[1], fileName))
		cc := &structs.Changeset{}
		err = json.NewDecoder(j).Decode(&cc)
		if err != nil {
			log.Fatal("Error decoding file ", fileName, " ", err)
		}
		for _, c := range cc.Changes {
			arr0 := strings.Split(c.Modified, " ")
			arr1 := strings.Split(arr0[0], "-")
			arr2 := strings.Split(arr0[1], ":")
			if !strings.HasSuffix(fileName, arr1[1]+".json") &&
				!strings.HasSuffix(fileName, monthsInEnglish[arr1[1]]+".json") {
				continue
			}
			if arr0[2] == "PM" {
				arr2[0] = hours[arr2[0]]
			}
			if arr0[2] == "AM" && arr2[0] == "12" {
				arr2[0] = "00"
			}
			t := fmt.Sprintf("%v:%v", arr2[0], arr2[1])
			modified := fmt.Sprintf("%v/%v/%v %v", arr1[0], months[arr1[1]], arr1[2], t)
			comm := c.Comment
			if len(comm) > 56 {
				comm = comm[:56]
			}
			if strings.ToLower(comm) == "<nenhum comentário>" {
				comm = ""
			}
			key := strings.ToLower(fmt.Sprintf("%v - %v - %v", comm, c.Author, modified))
			if _, ok := changesets[key]; ok {
				change := changesets[key]
				change.Uuids = append(change.Uuids, c.Uuid)
				changesetsByUuid[change.Uuid] = change
			} else {
				change := &structs.Change{
					Author:   c.Author,
					Comment:  comm,
					Modified: modified,
					Uuids:    []string{c.Uuid}}
				changesets[key] = change
				changesetsByUuid[c.Uuid] = change
			}
		}
	}
	defects := open(filepath.Join(os.Args[1], "defects.csv"))
	stories := open(filepath.Join(os.Args[1], "stories.csv"))
	features := open(filepath.Join(os.Args[1], "features.csv"))
	issues := open(filepath.Join(os.Args[1], "siop-issues.csv"))
	defer func() {
		defects.Close()
		stories.Close()
		features.Close()
		issues.Close()
	}()
	lookupChangeset := func(dc string) (*structs.Change, string) {
		comm := dc[strings.Index(dc, " - ")+3:]
		comm = comm[:strings.LastIndex(comm, " - ")]
		comm = comm[:strings.LastIndex(comm, " - ")]
		if len(comm) > 56 {
			comm = comm[:56]
		}
		if strings.ToLower(comm) == "<nenhum comentário>" {
			comm = ""
		}
		arr := strings.Split(dc, " - ")
		author := arr[len(arr)-2]
		time := arr[len(arr)-1]
		key := strings.ToLower(fmt.Sprintf("%v - %v - %v", comm, author, time))
		c := changesets[key]
		if c == nil {
			//fmt.Println(dc)
			fmt.Fprintf(os.Stderr, "Key not found: ", key)
		}
		return c, key
	}
	storiesMap := map[string]string{}
	commits := map[string]*structs.Commit{}
	r := csv.NewReader(defects)
	read(r, func(record []string) {
		defectChangesets := strings.Split(record[4], "\n")
		for _, dc := range defectChangesets {
			cs, key := lookupChangeset(dc)
			commits[key] = &structs.Commit{
				Change:  cs,
				Issue:   structs.Issue{record[1], "bug"},
				Feature: strings.Split(record[3], ":")[0]}
			for _, uuid := range cs.Uuids {
				delete(changesetsByUuid, uuid)
			}
		}
	})
	r = csv.NewReader(features)
	read(r, func(record []string) {
		storiesMap[record[1]] = strings.Split(record[10], ":")[0]
	})
	r = csv.NewReader(stories)
	read(r, func(record []string) {
		defectChangesets := strings.Split(record[9], "\n")
		for _, dc := range defectChangesets {
			cs, key := lookupChangeset(dc)
			commits[key] = &structs.Commit{
				Change:  cs,
				Issue:   structs.Issue{record[8][1:], "story"},
				Feature: storiesMap[record[8][1:]]}
			for _, uuid := range cs.Uuids {
				delete(changesetsByUuid, uuid)
			}
		}
	})
	r = csv.NewReader(issues)
	issuesMap := map[string]string{}
	read(r, func(record []string) {
		if record[1] == "1" {
			issuesMap[record[0]] = "bug"
		} else {
			issuesMap[record[0]] = "story"
		}
	})
	re, err := regexp.Compile("#\\d+")
	if err != nil {
		log.Fatal(err)
	}
	for key, cs := range changesetsByUuid {
		change := *cs
		change.Uuids = []string{key}
		commits[key] = &structs.Commit{Change: &change}
		issueId := re.FindString(cs.Comment)
		if issueId == "" {
			commits[key].Issue = structs.Issue{issueId, issuesMap[issueId[1:]]}
		}
	}
	result := make([]*structs.Commit, 0, len(commits))
	for _, commit := range commits {
		commit.Files = []string{}
		for _, uuid := range commit.Change.Uuids {
			cmd := exec.Command("lscm", "list", "changes", "-r", "siop", uuid, "-j")
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Fatal(err, string(out))
			}
			change := &structs.Changeset{}
			err = json.Unmarshal(out, change)
			if err != nil {
				log.Fatal(err)
			}
			for _, c := range change.Changes {
				for _, f := range c.Changes {
					commit.Files = append(commit.Files, f.Path)
				}
			}
		}
		result = append(result, commit)
	}
	json.NewEncoder(os.Stdout).Encode(result)
}

func open(file string) *os.File {
	result, err := os.Open(file)
	if err != nil {
		log.Fatal("Error opening file ", file, err)
	}
	return result
}

func read(r *csv.Reader, f func(record []string)) {
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil && err != io.EOF {
			log.Fatal("Error reading defects file ", err)
		}
		if record[1] != "Id" {
			f(record)
		}
	}
}

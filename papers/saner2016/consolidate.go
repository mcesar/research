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
	"strings"
)

type issue struct {
	id, kind string
}

type commit struct {
	uuids   []string
	comment string
	user    string
	time    string
	feature string
	issue   issue
	files   []string
}

type changeset struct {
	Changes []change `json:changes`
}

type change struct {
	Author   string `json:author`
	Comment  string `json:comment`
	Modified string `json:Modified`
	Uuid     string `json:uuid`
	Changes  []file `json:changes`
	uuids    []string
}

type file struct {
	Path string `json:path`
}

func main() {
	folder := open(os.Args[1])
	defer folder.Close()
	fileNames, err := folder.Readdirnames(0)
	if err != nil {
		log.Fatal("Error reading file names from ", os.Args[1], err)
	}
	changesets := map[string]*change{}
	months := map[string]string{"jan": "01", "fev": "02", "mar": "03", "abr": "04", "mai": "05",
		"jun": "06", "jul": "07", "ago": "08", "set": "09", "out": "10", "nov": "11", "dez": "12"}
	hours := map[string]string{"01": "13", "02": "14", "03": "15", "04": "16", "05": "17",
		"06": "18", "07": "19", "08": "20", "09": "21", "10": "22", "11": "23", "12": "12"}
	for _, fileName := range fileNames {
		if !strings.HasSuffix(fileName, ".json") {
			continue
		}
		j := open(filepath.Join(os.Args[1], fileName))
		cc := &changeset{}
		err = json.NewDecoder(j).Decode(&cc)
		if err != nil {
			log.Fatal("Error decoding file ", fileName, " ", err)
		}
		for _, c := range cc.Changes {
			arr0 := strings.Split(c.Modified, " ")
			arr1 := strings.Split(arr0[0], "-")
			arr2 := strings.Split(arr0[1], ":")
			if !strings.HasSuffix(fileName, arr1[1]+".json") {
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
				changesets[key].uuids = append(changesets[key].uuids, c.Uuid)
			} else {
				changesets[key] = &change{uuids: []string{c.Uuid}}
			}
		}
	}
	defects := open(filepath.Join(os.Args[1], "defects.csv"))
	stories := open(filepath.Join(os.Args[1], "stories.csv"))
	features := open(filepath.Join(os.Args[1], "features.csv"))
	defer func() {
		defects.Close()
		stories.Close()
		features.Close()
	}()
	unpackChangeset := func(dc string) ([]string, string, string, string, string) {
		arr := strings.Split(dc, " - ")
		comm := ""
		for i := 1; i < len(arr)-2; i++ {
			if len(comm) > 0 {
				comm += " - "
			}
			comm += arr[i]
		}
		if len(comm) > 56 {
			comm = comm[:56]
		}
		if strings.ToLower(comm) == "<nenhum comentário>" {
			comm = ""
		}
		author := arr[len(arr)-2]
		time := arr[len(arr)-1]
		key := strings.ToLower(fmt.Sprintf("%v - %v - %v", comm, author, time))
		c := changesets[key]
		if c == nil {
			log.Fatal("Key not found: ", key)
		}
		return c.uuids, comm, author, time, key
	}
	storiesMap := map[string]string{}
	commits := map[string]*commit{}
	r := csv.NewReader(defects)
	read(r, func(record []string) {
		defectChangesets := strings.Split(record[4], "\n")
		for _, dc := range defectChangesets {
			uuids, comm, author, time, key := unpackChangeset(dc)
			commits[key] = &commit{
				uuids:   uuids,
				comment: comm,
				user:    author,
				time:    time,
				issue:   issue{record[1], "bug"},
				feature: strings.Split(record[3], ":")[0]}
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
			uuids, comm, author, time, key := unpackChangeset(dc)
			commits[key] = &commit{
				uuids:   uuids,
				comment: comm,
				user:    author,
				time:    time,
				issue:   issue{record[8][1:], "story"},
				feature: storiesMap[record[8][1:]]}
		}
	})
	for _, commit := range commits {
		commit.files = []string{}
		for _, uuid := range commit.uuids {
			command := fmt.Sprintf("lscm list changes -r siop %v -j", uuid)
			out, err := exec.Command(command).Output()
			if err != nil {
				log.Fatal(err)
			}
			change := &change{}
			err = json.Unmarshal(out, change)
			if err != nil {
				log.Fatal(err)
			}
			for _, f := range change.Changes {
				commit.files = append(commit.files, f.Path)
			}
		}
	}
	json.NewEncoder(os.Stdout).Encode(commits)
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

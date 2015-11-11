package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"../../lib"
)

type nameCount struct {
	name  string
	count int
}

type byCount []nameCount

func (arr byCount) Len() int { return len(arr) }
func (arr byCount) Less(i, j int) bool {
	return arr[i].count < arr[j].count
}
func (arr byCount) Swap(i, j int) { arr[i], arr[j] = arr[j], arr[i] }

type byModifiedTime []*lib.Commit

func (arr byModifiedTime) Len() int { return len(arr) }
func (arr byModifiedTime) Less(i, j int) bool {
	return arr[i].Change.ModifiedTime.Before(arr[j].Change.ModifiedTime)
}
func (arr byModifiedTime) Swap(i, j int) { arr[i], arr[j] = arr[j], arr[i] }

func main() {
	system := flag.String("s", "siop", "system")
	flag.Parse()
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage [-s system] repository")
	}
	f := lib.CommitsFunctions(*system)
	commits, err := f.Commits(os.Args, f.IssueExtractor)
	if err != nil {
		log.Fatal(err)
	}
	sort.Sort(byModifiedTime(commits))
	sum := 0
	totalIntervals := 0.0
	commiters := map[string]int{}
	var lastTime *time.Time
	for _, c := range commits {
		sum += len(c.Files)
		if lastTime != nil {
			totalIntervals += c.Change.ModifiedTime.Sub(*lastTime).Hours()
		}
		lastTime = &c.Change.ModifiedTime
		commiters[c.Change.Author]++
	}
	authors := make([]nameCount, 0, len(commiters))
	for k, v := range commiters {
		authors = append(authors, nameCount{k, v})
	}
	sort.Sort(byCount(authors))
	fmt.Println(float64(sum)/float64(len(commits)), float64(totalIntervals)/float64(len(commits)-1))
	for _, k := range authors {
		fmt.Println(k.count, k.name)
	}
}

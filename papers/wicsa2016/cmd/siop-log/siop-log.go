package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"time"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "usage: siop-log <output-dir> <start-YYYY/MM> <end-YYYY/MM>\n")
		os.Exit(1)
	}
	date, err := time.Parse("2006/01", os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	end, err := time.Parse("2006/01", os.Args[3])
	if err != nil {
		log.Fatal(err)
	}
	end.AddDate(0, 1, 0)
	//date := time.Date(2008, time.January, 1, 0, 0, 0, 0, time.Local)
	for !date.After(end) {
		cmd := exec.Command("lscm", "list", "changesets", "-r", "siop",
			"--created-after", date.AddDate(0, 0, -1).Format("2006/01/02"),
			"--created-before", date.AddDate(0, 1, 0).Format("2006/01/02"), "-m", "10000", "-j")
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatal(err, string(out))
		}
		f, err := os.Create(path.Join(os.Args[1],
			fmt.Sprintf("siop-changesets-%v-%v.json", date.Year(), date.Month())))
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(out)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		date = date.AddDate(0, 1, 0)
		fmt.Println(cmd.Args)
	}
}

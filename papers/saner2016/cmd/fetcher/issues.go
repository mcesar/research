package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Rss struct {
	Items []Item `xml:"channel>item"`
}

type Item struct {
	Title   string `xml:"title"`
	Key     string `xml:"key"`
	Type    string `xml:"type"`
	Parent  string `xml:"parent"`
	Created string `xml:"created"`
}

var urls = map[string]string{
	"ofbiz": "https://issues.apache.org/jira/sr/jira.issueviews:searchrequest-xml/temp/" +
		"SearchRequest.xml?jqlQuery=project+%3D+OFBIZ&tempMax=100&" +
		"field=key&field=title&field=type&field=created&field=parent&pager/start=",
	"openmrs": "https://issues.openmrs.org/sr/jira.issueviews:searchrequest-xml/temp/" +
		"SearchRequest.xml?jqlQuery=project+%3D+TRUNK&tempMax=100&" +
		"field=key&field=title&field=type&field=created&field=parent&pager/start="}

func main() {

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: issues <repository>")
		os.Exit(1)
	}
	url := urls[os.Args[1]]
	if url == "" {
		fmt.Fprint(os.Stderr, "please choose one repository as follow: ")
		for k, _ := range urls {
			fmt.Fprint(os.Stderr, k, " ")
		}
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}
	start := 0
	issues := map[string]*Item{}
	for {
		resp, err := http.Get(url + fmt.Sprint(start))
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetcher: %v\n", err)
			os.Exit(1)
		}
		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetcher: reading %s: %v\n", url, err)
			os.Exit(1)
		}
		rss := Rss{}
		err = xml.Unmarshal(b, &rss)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetcher: parsing: %v\n", err)
			os.Exit(1)
		}
		for _, item := range rss.Items {
			i := item
			issues[item.Key] = &i
		}
		if len(rss.Items) < 100 {
			break
		}
		start += 100
	}
	for _, item := range issues {
		if item.Type == "Sub-task" {
			if item.Parent == "" {
				fmt.Fprintf(os.Stderr, "fetcher: empty parent %v\n", item.Key)
			} else {
				if parent, ok := issues[item.Parent]; ok {
					item.Type = parent.Type
				} else {
					fmt.Fprintf(os.Stderr, "fetcher: parent not found %v\n", item.Parent)
				}
			}
		}
		fmt.Printf("%v,%v\n", item.Key, item.Type)
	}
}

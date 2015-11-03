package lib

import "time"

type Issue struct {
	Id, Kind string
}

type Commit struct {
	Feature string
	Issue   Issue
	Change  *Change
	Files   []string
}

type Changeset struct {
	Changes []Change `json:changes`
}

type Change struct {
	Author       string `json:author`
	Comment      string `json:comment`
	Modified     string `json:Modified`
	ModifiedTime time.Time
	Uuid         string `json:uuid`
	Changes      []File `json:changes`
	Uuids        []string
}

type File struct {
	Path string `json:path`
}

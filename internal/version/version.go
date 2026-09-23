package version

import "fmt"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func Current() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}

func (info Info) String() string {
	return fmt.Sprintf("espercli %s (commit %s, built %s)", info.Version, info.Commit, info.Date)
}

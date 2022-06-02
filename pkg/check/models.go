package check

import (
	. "github.com/DrummyFloyd/gitlab-merge-request-resource/pkg"
)

// cas ou pas de number donc list merge request

type Request struct {
	Source  Source  `json:"source"`
	Version Version `json:"version,omitempty"`
}

type Response []Version

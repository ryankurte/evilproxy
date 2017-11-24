/**
 * SRI plugin re-writes or recomputes SubResource Integrity Tags for injected files
 *
 * Copyright 2017 Ryan Kurte
 */

package plugins

import (
	"regexp"
)

const (
	SHA256 = "sha256"
	SHA384 = "sha384"
	SHA512 = "sha512"
)

var sriExp = regexp.MustCompile(`(integrity\=\"[a-z0-9A-Z\-\+\/]+"[\n\r\s]*)`)

type SRIPlugin struct {
}

func NewSRIPlugin() *SRIPlugin {
	return &SRIPlugin{}
}

func (s *SRIPlugin) HandleResponseBody(body []byte) []byte {
	return sriExp.ReplaceAll(body, []byte{})
}

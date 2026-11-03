package blob

import (
	"fmt"
	"regexp"
	"strings"
)

var hexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

func Path(root, sha string) (string, error) {
	if !hexRe.MatchString(sha) {
		return "", fmt.Errorf("bad sha")
	}
	return fmt.Sprintf("%s/blobs/%s/%s/%s", root, sha[:2], sha[2:4], sha), nil
}

func IsValid(sha string) bool { return hexRe.MatchString(strings.ToLower(sha)) }

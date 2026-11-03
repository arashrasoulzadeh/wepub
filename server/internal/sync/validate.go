package sync

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var ulidRe = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

type Op struct {
	ID       string `json:"id"`
	DeviceID string `json:"device_id"`
	HLC      string `json:"hlc"`
	Entity   string `json:"entity"`
	EntityID string `json:"entity_id"`
	Field    string `json:"field"`
	Value    []byte `json:"value"`
}

func ValidateOp(op Op, now time.Time) error {
	if op.ID == "" || !ulidRe.MatchString(op.ID) {
		return errors.New("bad id")
	}
	if op.Entity == "" {
		return errors.New("empty entity")
	}
	if len(op.Value) > 64*1024 {
		return errors.New("oversize value")
	}
	// hlc format: {wall_ms:013d}-{counter:04x}-{device_id}
	parts := splitHLC(op.HLC)
	if len(parts) != 3 {
		return errors.New("malformed hlc")
	}
	if parts[2] != op.DeviceID {
		return errors.New("hlc device mismatch")
	}
	wall, err := parseWall(parts[0])
	if err != nil {
		return err
	}
	if wall.After(now.Add(24 * time.Hour)) {
		return fmt.Errorf("hlc too far future")
	}
	return nil
}

func splitHLC(s string) []string {
	// 0000000000000-0000-dev
	if len(s) < 20 {
		return nil
	}
	// find dashes
	var out []string
	cur := ""
	for _, c := range s {
		if c == '-' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(c)
		}
	}
	out = append(out, cur)
	if len(out) != 3 {
		return nil
	}
	return out
}

func parseWall(s string) (time.Time, error) {
	if len(s) != 13 {
		return time.Time{}, errors.New("bad wall")
	}
	var ms int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return time.Time{}, errors.New("bad wall")
		}
		ms = ms*10 + int64(c-'0')
	}
	return time.UnixMilli(ms), nil
}

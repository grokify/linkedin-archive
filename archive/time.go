package archive

import (
	"regexp"
	"strings"
	"time"
)

var timeLayouts = []string{
	"2006-01-02 15:04:05 MST",
	"2006/01/02 15:04:05 MST",
	"2006-01-02 15:04 MST",
	"2006-01-02 15:04:05",
	"02 Jan 2006",
	"2 Jan 2006",
	"1/2/06, 3:04 PM",
	"1/2/2006, 3:04 PM",
	"1/2/06 3:04 PM",
	"Mon Jan 02 15:04:05 MST 2006",
	"Mon Jan 2 15:04:05 MST 2006",
	"January 2, 2006 at 3:04 PM (MST)",
	"Jan 2006",
	"2006-01-02",
	time.RFC3339,
	time.RFC3339Nano,
}

// embeddedDate extracts dates embedded in prose, such as
// "You uploaded a feed photo on June 24, 2026 at 10:33 PM (GMT)".
var embeddedDate = regexp.MustCompile(`[A-Z][a-z]+ \d{1,2}, \d{4} at \d{1,2}:\d{2} [AP]M \([A-Z]+\)`)

// ParseTime parses the date formats found in LinkedIn exports. Values without
// a zone are interpreted as UTC. Empty and "N/A" values return ok == false.
func ParseTime(s string) (t time.Time, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "N/A") {
		return time.Time{}, false
	}
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t.UTC(), true
		}
	}
	if m := embeddedDate.FindString(s); m != "" {
		return ParseTime(m)
	}
	return time.Time{}, false
}

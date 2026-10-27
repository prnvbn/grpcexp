package call

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "now") {
		return time.Now(), nil
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05Z0700",
		"2006-01-02 15:04:05 Z07:00",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02T15:04:05",
		time.DateTime,
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		time.DateOnly,
		time.RFC1123Z,
		"Mon, 02 Jan 2006 15:04:05 GMT",
		time.RFC822Z,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return checkedTimestamp(t)
		}
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		// Infer Unix timestamp precision from its number of digits.
		digits := len(strings.TrimLeft(s, "+-"))
		switch digits {
		case 13:
			return checkedTimestamp(time.UnixMilli(n))
		case 16:
			return checkedTimestamp(time.UnixMicro(n))
		case 19:
			return checkedTimestamp(time.Unix(0, n))
		default:
			return checkedTimestamp(time.Unix(n, 0))
		}
	}

	return time.Time{}, fmt.Errorf("must be now, a valid date/time, or a Unix timestamp")
}

func checkedTimestamp(t time.Time) (time.Time, error) {
	if err := timestamppb.New(t).CheckValid(); err != nil {
		return time.Time{}, fmt.Errorf("timestamp out of range: %w", err)
	}
	return t, nil
}

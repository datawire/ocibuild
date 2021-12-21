package reproducible

import (
	"os"
	"strconv"
	"time"
)

var now time.Time

func Now() time.Time {
	if now.IsZero() {
		secs, err := strconv.ParseInt(os.Getenv("SOURCE_DATE_EPOCH"), 10, 64)
		if err == nil {
			now = time.Unix(secs, 0)
		} else {
			now = time.Now()
		}
	}
	return now
}

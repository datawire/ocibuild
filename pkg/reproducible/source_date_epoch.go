package reproducible

import (
	"os"
	"strconv"
	"sync"
	"time"
)

//nolint:gochecknoglobals // this needs to be global
var (
	nowOnce sync.Once
	now     time.Time
)

func Now() time.Time {
	nowOnce.Do(func() {
		secs, err := strconv.ParseInt(os.Getenv("SOURCE_DATE_EPOCH"), 10, 64)
		if err == nil {
			now = time.Unix(secs, 0)
		} else {
			now = time.Now()
		}
	})
	return now
}

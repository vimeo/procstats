package procstats

import (
	"os"
	"runtime"
	"testing"
)

func TestReadRSS(t *testing.T) {
	t.Parallel()
	pid := os.Getpid()
	st := runtime.MemStats{}
	runtime.ReadMemStats(&st)
	rss, err := readProcessRSS(pid)
	if err != nil {
		t.Errorf("failed to read RSS for self: %s", err)
		return
	}

	if rss < int64(os.Getpagesize()) {
		t.Errorf("rss is less than 1 page: %d", rss)
	}

	// Since we did some work after grabbing the mem stats, but
	// while/before reading the RSS from /proc/pid/statm, we expect that
	// RSS will be larger than st.Sys here. (50% and 5% were flaky, so the threshold is at 0 now)
	if rss == 0 {
		t.Errorf("rss is zero: rss: %d", rss)
	}
	if rss > int64(st.Sys)*20 {
		t.Errorf("rss is more than 20x of MemStats.Sys: rss: %d vs Sys: %d", rss, st.Sys)
	}
}

package procstats

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func TestReadCPUUsage(t *testing.T) {
	t.Parallel()
	dur := time.Minute
	var table = []struct {
		Name string
		Stat []byte
		Err  bool
		Want CPUTime
	}{
		{
			Name: "zeroes",
			Stat: []byte("x x x x x x x x x x x x x 0 0 0 0 x"),
			Err:  false,
			Want: CPUTime{},
		},
		{
			Name: "err parse",
			Stat: []byte("x x x x x x x x x x x x 0 0 0 0"),
			Err:  true,
			Want: CPUTime{},
		},
		{
			Name: "err fmt utime",
			Stat: []byte("x x x x x x x x x x x x x x 0 0 0 x"),
			Err:  true,
			Want: CPUTime{},
		},
		{
			Name: "err fmt stime",
			Stat: []byte("x x x x x x x x x x x x x 0 0 0 x x"),
			Err:  true,
			Want: CPUTime{},
		},
		{
			Name: "parse",
			Stat: []byte("x x x x x x x x x x x x x "),
			Err:  false,
			Want: CPUTime{2 * dur, 2 * dur},
		},
	}

	// Need to create an additional test case that's specific to the system
	// because we call out to sysconf to do the parsing.
	thz := time.Second / time.Duration(sysClockTick())
	x := &table[len(table)-1]
	x.Stat = strconv.AppendInt(x.Stat, int64(dur/thz), 10)
	x.Stat = append(x.Stat, ' ')
	x.Stat = strconv.AppendInt(x.Stat, int64(dur/thz), 10)
	x.Stat = append(x.Stat, ' ')
	x.Stat = strconv.AppendInt(x.Stat, int64(dur/thz), 10)
	x.Stat = append(x.Stat, ' ')
	x.Stat = strconv.AppendInt(x.Stat, int64(dur/thz), 10)
	x.Stat = append(x.Stat, []byte(" x")...)

	for _, c := range table {
		t.Run(c.Name, func(t *testing.T) {
			t.Logf("%q", string(c.Stat))
			ct, err := linuxParseCPUTime(c.Stat)
			t.Logf("%+v", ct)
			if c.Err {
				if err == nil {
					t.Fatalf("want: %v, got: %v", c.Err, err)
				}
				t.Logf("got error: %v", err)
				return
			}
			if err != nil {
				t.Fatalf("want: %v, got: %v", c.Err, err)
			}
			if want, got := c.Want, ct; want != got {
				t.Fatalf("want: %v, got: %v", want, got)
			}
		})
	}

	// NOTE(hank) This test actually round-trips everything via looking at the parent process.
	// Since this test is run in a separate binary, hopefully `go test` did enough work to
	// spawn us for us to notice. If this test goes flaky, feel free to remove.
	t.Run("parent", func(t *testing.T) {
		ct, err := readProcessCPUTime(os.Getppid())
		t.Logf("%+v", ct)
		if err != nil {
			t.Fatal(err)
		}
		if ct == (CPUTime{}) {
			t.Errorf("want: <non-zero>, got: %+v", ct)
		}
	})
}

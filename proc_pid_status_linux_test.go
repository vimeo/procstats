package procstats

import "testing"

func TestProcPidStatusParse(t *testing.T) {
	procSelfStatus := `Name:	vim
Umask:	0022
State:	R (running)
Tgid:	22987
Ngid:	0
Pid:	22987
PPid:	29507
TracerPid:	0
Uid:	1000	1000	1000	1000
Gid:	1000	1000	1000	1000
FDSize:	64
Groups:	10 18 19 27 78 85 102 999 1000 1001 
NStgid:	22987
NSpid:	22987
NSpgid:	22987
NSsid:	29507
VmPeak:	 2455528 kB
VmSize:	 2455528 kB
VmLck:	       0 kB
VmPin:	       0 kB
VmHWM:	   46661 kB
VmRSS:	   46660 kB
RssAnon:	   32276 kB
RssFile:	   14384 kB
RssShmem:	       0 kB
VmData:	  288308 kB
VmStk:	     132 kB
VmExe:	    2656 kB
VmLib:	   15660 kB
VmPTE:	     620 kB
VmSwap:	       0 kB
HugetlbPages:	       0 kB
CoreDumping:	0
Threads:	32
SigQ:	0/78835
SigPnd:	0000000000000000
ShdPnd:	0000000000000000
SigBlk:	0000000000000000
SigIgn:	0000000000003000
SigCgt:	00000001ef824eff
CapInh:	0000000000000000
CapPrm:	0000000000000000
CapEff:	0000000000000000
CapBnd:	0000003fffffffff
CapAmb:	0000000000000000
NoNewPrivs:	0
Seccomp:	0
Speculation_Store_Bypass:	vulnerable
Cpus_allowed:	f
Cpus_allowed_list:	0-3
Mems_allowed:	00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000000,00000001
Mems_allowed_list:	0
voluntary_ctxt_switches:	1163
nonvoluntary_ctxt_switches:	597`

	out := ProcPidStatus{}
	if parseErr := procPidStatusParser.Parse([]byte(procSelfStatus), &out); parseErr != nil {
		t.Fatalf("failed to parse: %s", parseErr)
	}

	if out.VMHWM != 46661*1024 {
		t.Errorf("unexpected value for VmHWM: %d; expected %d", out.VMHWM,
			46661*1024)
	}
}

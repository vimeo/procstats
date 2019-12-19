package pparser

import "testing"

func TestParseSimpleValFloats(t *testing.T) {
	type testStruct struct {
		A int64
		B float64
		C string
		D uint64
	}
	testVal := `A: 1023
B: 23.25
C: abcde
D: 12345`

	p := NewLineKVFileParser(testStruct{}, ":")

	out := testStruct{}
	err := p.Parse([]byte(testVal), &out)
	if err != nil {
		t.Fatalf("failed to parse: %s", err)
	}
	if out.A != 1023 {
		t.Errorf("unexpected value for A; %d; expected 1023", out.A)
	}
	if out.B != 23.25 {
		t.Errorf("unexpected value for B; %g; expected 23.25", out.B)
	}
	if out.C != "abcde" {
		t.Errorf("unexpected value for C; %q; expected \"abcde\"", out.C)
	}
	if out.D != 12345 {
		t.Errorf("unexpected value for D; %q; expected 12345", out.D)
	}
}

func TestParseUnknown(t *testing.T) {
	type testStruct struct {
		Known   int64
		Unknown map[string]int64 `pparser:"skip,unknown"`
	}

	testVal := `Known: 1023
B: 42
C: 123`

	p := NewLineKVFileParser(testStruct{}, ":")

	out := testStruct{}
	err := p.Parse([]byte(testVal), &out)
	if err != nil {
		t.Fatalf("failed to parse: %s", err)
	}

	if out.Known != 1023 {
		t.Errorf("unexpected value for A; %d; expected 1023", out.Known)
	}

	if len(out.Unknown) != 2 {
		t.Errorf("expected unknown fields to have 2 values, got %d instead", len(out.Unknown))
	}

	if out.Unknown["B"] != 42 {
		t.Errorf("expected unknown 'B' to be 42, got %d instead", out.Unknown["B"])
	}
	if out.Unknown["C"] != 123 {
		t.Errorf("expected unknown 'C' to be 123, got %d instead", out.Unknown["C"])
	}
}

func TestParseNoUnknownFields(t *testing.T) {
	type testStruct struct {
		Known int64
	}

	testVal := `Known: 1023
B: 42
C: 123`

	p := NewLineKVFileParser(testStruct{}, ":")

	out := testStruct{}
	err := p.Parse([]byte(testVal), &out)
	if err == nil {
		t.Fatal("expected error from parsing data with unknown fields")
	}
}

func TestParseDatatypeTooSmall(t *testing.T) {
	type testStruct struct {
		Known int8
	}

	testVal := `Known: 1024`
	p := NewLineKVFileParser(testStruct{}, ":")

	out := testStruct{}
	err := p.Parse([]byte(testVal), &out)
	if err == nil {
		t.Fatal("expected data overflow error")
	}
}

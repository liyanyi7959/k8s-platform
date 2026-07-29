package application

import "testing"

func TestActionPayload(t *testing.T) {
	payload := map[string]any{"replicas": 3}
	if got := ActionPayload(map[string]any{"payload": payload}); got["replicas"] != 3 {
		t.Fatalf("payload = %#v", got)
	}
	for _, input := range []map[string]any{nil, {}, {"payload": nil}, {"payload": "invalid"}} {
		if got := ActionPayload(input); len(got) != 0 {
			t.Fatalf("empty payload = %#v", got)
		}
	}
}

func TestActionExecutionStatusLabel(t *testing.T) {
	cases := map[string]string{
		"succeeded":   "成功",
		" SUCCEEDED ": "成功",
		"failed":      "失败",
		"pending":     "完成",
	}
	for input, want := range cases {
		if got := ActionExecutionStatusLabel(input); got != want {
			t.Fatalf("ActionExecutionStatusLabel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestActionConfirmationText(t *testing.T) {
	if got := ActionConfirmationText(42); got != "confirm-proposal-42" {
		t.Fatalf("ActionConfirmationText(42) = %q", got)
	}
}

func TestIntegerInputValue(t *testing.T) {
	cases := []struct {
		input any
		want  int
		ok    bool
	}{
		{42, 42, true}, {int8(3), 3, true}, {int16(100), 100, true}, {int32(1000), 1000, true},
		{int64(9999), 9999, true}, {uint(7), 7, true}, {uint8(8), 8, true}, {uint16(9), 9, true},
		{uint32(10), 10, true}, {uint64(11), 11, true}, {float32(1.5), 1, true}, {float64(3.14), 3, true},
		{"hello", 0, false}, {nil, 0, false}, {true, 0, false},
	}
	for _, tt := range cases {
		got, ok := IntegerInputValue(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("IntegerInputValue(%#v) = (%d, %v), want (%d, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNestedIntegerInputValue(t *testing.T) {
	input := map[string]any{
		"config": map[string]any{"timeout": 30, "nested": map[string]any{"deep": 42}},
		"simple": 10,
	}
	for _, tt := range []struct {
		keys []string
		want int
	}{
		{[]string{"config", "timeout"}, 30},
		{[]string{"config", "nested", "deep"}, 42},
		{[]string{"simple"}, 10},
		{[]string{"missing", "key"}, 0},
		{[]string{"config", "missing"}, 0},
		{nil, 0},
	} {
		if got := NestedIntegerInputValue(input, tt.keys...); got != tt.want {
			t.Fatalf("NestedIntegerInputValue(%v) = %d, want %d", tt.keys, got, tt.want)
		}
	}
}

func TestStringInputValue(t *testing.T) {
	for _, tt := range []struct {
		input any
		want  string
	}{
		{"hello", "hello"}, {"  spaced  ", "spaced"}, {[]byte("bytes"), "bytes"},
		{[]byte("  bytes  "), "bytes"}, {42, "42"}, {nil, ""},
	} {
		if got := StringInputValue(tt.input); got != tt.want {
			t.Fatalf("StringInputValue(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

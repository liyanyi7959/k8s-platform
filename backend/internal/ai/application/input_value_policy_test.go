package application

import (
	"reflect"
	"testing"
)

func TestStringSliceInput(t *testing.T) {
	if got := StringSliceInput([]string{" a ", "", "b"}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("string slice = %#v", got)
	}
	if got := StringSliceInput([]any{"a", 7, " "}); !reflect.DeepEqual(got, []string{"a", "7"}) {
		t.Fatalf("any slice = %#v", got)
	}
	if got := StringSliceInput("unsupported"); got != nil {
		t.Fatalf("unsupported input = %#v", got)
	}
}

package application

import (
	"reflect"
	"testing"
)

func TestToolPermissionPolicy(t *testing.T) {
	if !HasAllToolPermissions([]string{" a ", "b"}, []string{"a", "b"}) {
		t.Fatal("trimmed permissions should be accepted")
	}
	if HasAllToolPermissions([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("missing required permission should be rejected")
	}
	if got := MissingToolPermissions([]string{"a"}, []string{"a", "b", "", "c"}); !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatalf("missing permissions = %#v", got)
	}
}

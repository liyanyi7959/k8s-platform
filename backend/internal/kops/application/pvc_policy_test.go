package application

import "testing"

func TestNormalizePVCAccessModes(t *testing.T) {
	modes, err := NormalizePVCAccessModes([]string{" ReadWriteOnce ", "ReadWriteOnce", "ReadOnlyMany"})
	if err != nil || len(modes) != 2 || modes[0] != "ReadWriteOnce" || modes[1] != "ReadOnlyMany" {
		t.Fatalf("modes=%#v err=%v", modes, err)
	}
	if _, err := NormalizePVCAccessModes([]string{"invalid"}); err == nil {
		t.Fatal("invalid mode must fail")
	}
}

func TestValidatePVCCapacity(t *testing.T) {
	if err := ValidatePVCCapacity("1Gi"); err != nil {
		t.Fatalf("valid capacity error=%v", err)
	}
	if err := ValidatePVCCapacity("0"); err == nil {
		t.Fatal("zero capacity must fail")
	}
}

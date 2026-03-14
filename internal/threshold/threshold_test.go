package threshold

import "testing"

func TestCheckPass(t *testing.T) {
	if err := Check(85.0, 80.0); err != nil {
		t.Errorf("expected pass, got: %v", err)
	}
}

func TestCheckExact(t *testing.T) {
	if err := Check(80.0, 80.0); err != nil {
		t.Errorf("expected pass at exact threshold, got: %v", err)
	}
}

func TestCheckFail(t *testing.T) {
	err := Check(79.0, 80.0)
	if err == nil {
		t.Fatal("expected failure below threshold")
	}
}

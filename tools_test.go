package toolkit

import "testing"

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Errorf("Expected random string of length 10, but got length: %d", len(s))
	}
}

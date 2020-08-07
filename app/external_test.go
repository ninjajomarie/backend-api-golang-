package external_test

import "testing"

func TestFixture(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()
}

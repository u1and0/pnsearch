package main

import "testing"

func TestToRagex(t *testing.T) {
	s := `ab cd`
	expect := `.*ab.*cd.*`
	actual := ToRegex(s)
	if expect != actual {
		t.Fatalf("got: %v want: %v", actual, expect)
	}
}

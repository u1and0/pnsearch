package main

import (
	"reflect"
	"testing"

	"github.com/vishalkuo/bimap"
)

/*
	init() にflag.Parse()を書くと起きる不具合 Go version > 1.13

[go test flag: flag provided but not defined](https://stackoverflow.com/questions/29699982/go-test-flag-flag-provided-but-not-defined)
If you've migrated to golang 13, it changed the order of the test initializer, so it could lead to something like

flag provided but not defined: -test.timeout
as a possible workaround, you can use

	var _ = func() bool {
	    testing.Init()
	    return true
	}()

that would call test initialization before the application one. More info can be found on the original thread:
*/
var _ = func() bool {
	testing.Init()
	return true
}()

func TestToRagex(t *testing.T) {
	s := `ab cd`
	expect := `(?i).*ab.*cd.*`
	actual := ToRegex(s)
	if expect != actual {
		t.Fatalf("got: %v want: %v", actual, expect)
	}

	// 複数スペース除去
	s = `    ab             cd    `
	expect = `(?i).*ab.*cd.*`
	actual = ToRegex(s)
	if expect != actual {
		t.Fatalf("got: %v want: %v", actual, expect)
	}

	// 全角スペース除去
	s = `　ab　cd　`
	expect = `(?i).*ab.*cd.*`
	actual = ToRegex(s)
	if expect != actual {
		t.Fatalf("got: %v want: %v", actual, expect)
	}

	// タブ文字除去
	s = "\tab\tcd\t"
	expect = `(?i).*ab.*cd.*`
	actual = ToRegex(s)
	if expect != actual {
		t.Fatalf("got: %v want: %v\n", actual, expect)
	}
}

func TestConvertHeader(t *testing.T) {
	mp := map[string]string{"a": "1", "b": "2", "c": "3"}
	convertMap := bimap.NewBiMapFromMap(mp)

	testSlice := []string{"a", "b", "c", "d"}
	expect := []string{"1", "2", "3", "d"}
	actual := ConvertHeader(convertMap, testSlice, true)
	if reflect.DeepEqual(expect, actual) {
		t.Fatalf("got: %v want: %v\n", actual, expect)
	}

	testSlice = []string{"1", "2", "3", "4"}
	expect = []string{"a", "b", "c", "4"}
	actual = ConvertHeader(convertMap, testSlice, false)
	if reflect.DeepEqual(expect, actual) {
		t.Fatalf("got: %v want: %v\n", actual, expect)
	}
}

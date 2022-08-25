package main

import (
	"reflect"
	"testing"
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

func TestFieldNameToAlias(t *testing.T) {
	testSlice := []string{"品名", "ユニットNo"}
	expect := []string{"品名", "要求番号"}
	actual := FieldNameToAlias(testSlice)
	for i, e := range expect {
		if actual[i] != e {
			t.Fatalf("got: %v want: %v\n", actual, expect)
		}
	}
}

func TestAliasToFieldName(t *testing.T) {
	testSlice := []string{"品名", "要求番号"}
	expect := []string{"品名", "ユニットNo"}
	actual := AliasToFieldName(testSlice)
	for i, e := range expect {
		if actual[i] != e {
			t.Fatalf("got: %v want: %v\n", actual, expect)
		}
	}
}

func TestLabelMaker(t *testing.T) {
	aliases := []string{"labela", "labelb", "ユニットNo"}
	actual := LabelMaker(aliases)
	expect := Labels{
		//      Alias   Name
		Label{"labela", "labela"},
		Label{"labelb", "labelb"},
		Label{"要求番号", "ユニットNo"},
	}
	for i, e := range expect {
		if actual[i] != e {
			t.Fatalf("got: %v want: %v\n", actual, expect)
		}
	}
}

func Test_TableToObject(t *testing.T) {
	//            0   1    2    3    4    5
	c1 := Column{"", "AA", "", "", "bb", "cC", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", ""}
	c2 := Column{"", "1", "", "", "2", "3", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", ""}
	c3 := Column{"", "zz", "", "", "yy", "x", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", ""}
	test := Table{c1, c2, c3}
	o1 := Object{
		ProductNo: "AA", // 1
		Pid:       "1",  // 4
		Name:      "zz", //5
	}
	o2 := Object{
		ProductNo: "bb", // 1
		Pid:       "2",  // 4
		Name:      "yy", //5
	}
	o3 := Object{
		ProductNo: "cC", // 1
		Pid:       "3",  // 4
		Name:      "x",  //5
	}
	expect := []Object{o1, o2, o3}
	actual := test.ToObject()
	for i, e := range expect {
		if reflect.DeepEqual(actual[i], e) {
			t.Fatalf("got: %v want: %v\n", actual, expect)
		}
	}
}

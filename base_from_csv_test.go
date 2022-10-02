package setlogs

import (
	"fmt"
	"testing"
)

func TestReadCSV(t *testing.T) {
	f := OpenFile("testbase.csv")
	defer f.Close()
	logs := ReadCSV(f, TypeBase)

	fmt.Println(logs.Filter(func(r Record) bool {
		fmt.Println(r, r["name"] == "mofon")
		fmt.Printf("%q\n", r["name"])
		return r["name"] == "mofon"
	}))
}

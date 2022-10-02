package setlogs

import (
	"fmt"
	"os"
)

func OpenFile(filename string) *os.File {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return f
}

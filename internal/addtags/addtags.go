// Command addtags adds the specified build tags to the specified Go source file.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
)

var tags = flag.String("tags", "!js", "build tags to add to the file")
var file = flag.String("file", "main.go", "name of the file to add build tags to")

func main() {
	flag.Parse()
	f, err := os.Open(*file)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	re := regexp.MustCompile("package (.*)\n\nimport")
	o := re.ReplaceAll(b, []byte(fmt.Sprintf("//+build %s\n\npackage ${1}\n\nimport", *tags)))

	of, err := os.Create(*file)
	if err != nil {
		panic(err)
	}

	if _, err := of.Write(o); err != nil {
		panic(err)
	}
	of.Close()
}

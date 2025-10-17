package main

import (
	"os"
	"path/filepath"
	"strings"
)

func main() {
	filepath.Walk("proto", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".pb.go") {
			os.Remove(path)
		}
		return nil
	})
}


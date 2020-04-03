package main

import (
	"fmt"
	"path"
	"strings"
)

func getUniverseAndWorld(dir string) (universe, world string, err error) {
	if strings.HasSuffix(dir, "/") {
		dir = dir[:len(dir)-1]
	}
	world = path.Base(dir)
	universe = path.Dir(dir)

	if world == "." {
		err = fmt.Errorf("path empty")
	}

	return
}

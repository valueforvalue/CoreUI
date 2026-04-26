package main

import (
	"fmt"
	"os"

	"github.com/valueforvalue/coreui/pkg/docs"
)

func main() {
	content, err := docs.RenderComponentsReference()
	if err != nil {
		fail(err)
	}

	if err := os.WriteFile("COMPONENTS.md", []byte(content), 0o644); err != nil {
		fail(err)
	}

	fmt.Println("COMPONENTS.md")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"coreui/pkg/compiler"
)

const version = "dev"

func main() {
	var outputPath string
	var showVersion bool

	flag.StringVar(&outputPath, "o", "", "output JSON path")
	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: corec [-o output.json] input.cui")
		os.Exit(1)
	}

	inputPath := flag.Arg(0)
	if outputPath == "" {
		outputPath = defaultOutputPath(inputPath)
	}

	data, err := compiler.CompileFile(inputPath, compiler.Options{Version: version})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Println(outputPath)
}

func defaultOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	if ext == "" {
		return inputPath + ".json"
	}
	return strings.TrimSuffix(inputPath, ext) + ".json"
}

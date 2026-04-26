package compiler

import (
	"encoding/json"
	"os"
	"time"

	"coreui/pkg/generator"
	"coreui/pkg/parser"
)

type Options struct {
	Timestamp time.Time
	Version   string
}

func CompileSource(name, source string, options Options) ([]byte, error) {
	_ = name

	if options.Timestamp.IsZero() {
		options.Timestamp = time.Now().UTC()
	}
	if options.Version == "" {
		options.Version = "dev"
	}

	document, err := parser.New(source).ParseDocument()
	if err != nil {
		return nil, err
	}

	output := generator.Build(document, options.Timestamp, options.Version)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(data, '\n'), nil
}

func CompileFile(path string, options Options) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return CompileSource(path, string(data), options)
}

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nime/logparser/pkg/parser"
)

func main() {
	templatePath := flag.String("template", "templates/template.yaml", "Path to template YAML file")
	outputFormat := flag.String("format", "json", "Output format: json, table")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: logparser [options] <logfile> [logfile2 ...]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	t, err := parser.LoadTemplate(*templatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load template: %v\n", err)
		os.Exit(1)
	}

	for _, path := range flag.Args() {
		if err := processFile(path, t, *outputFormat); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", path, err)
		}
	}
}

func processFile(path string, t *parser.Template, outputFormat string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	p, err := parser.NewParser(t)
	if err != nil {
		return err
	}

	metrics, err := p.ParseStream(bufio.NewScanner(f))
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		result := map[string]interface{}{
			"file":     path,
			"template": t.Name,
			"metrics":  metrics,
		}
		if err := enc.Encode(result); err != nil {
			return err
		}
	case "table":
		fmt.Printf("\n=== %s (template: %s) ===\n", path, t.Name)
		fmt.Printf("%-40s %12s %10s\n", "METRIC", "VALUE_MS", "SOURCE")
		fmt.Println(strings.Repeat("-", 65))
		for _, m := range metrics {
			fmt.Printf("%-40s %12.2f %10s\n", m.Name, m.ValueMs, m.Source)
		}
	default:
		return fmt.Errorf("unknown format: %s", outputFormat)
	}
	return nil
}

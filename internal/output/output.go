// Package output provides formatting helpers for CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Table prints rows in aligned columns.
func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println("(no results)")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Repeat("─\t", len(headers)))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// JSON prints data as indented JSON.
func JSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// Success prints a success message.
func Success(msg string) { fmt.Printf("✓ %s\n", msg) }

// Error prints an error message to stderr.
func Error(msg string) { fmt.Fprintf(os.Stderr, "✗ %s\n", msg) }

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/u7k4rs6/Umbra/umbra/internal/report"
)

// emit writes every requested output. The table always goes to stdout unless
// another format was asked for on stdout; --out writes the files.
func emit(o *Options, sd report.Sealed) error {
	// Stdout.
	switch o.Format {
	case "table":
		if err := writeTable(sd, o.All); err != nil {
			return err
		}
	case "json":
		if err := report.WriteJSON(os.Stdout, sd); err != nil {
			return err
		}
	case "packet":
		if err := report.Packet(os.Stdout, sd); err != nil {
			return err
		}
	case "html":
		// The HTML always goes to a file, so the table keeps stdout useful.
		if err := writeTable(sd, o.All); err != nil {
			return err
		}
	}

	if o.Out == "" {
		return nil
	}
	if err := os.MkdirAll(o.Out, 0o755); err != nil {
		return err
	}

	jsonPath := filepath.Join(o.Out, "umbra.json")
	f, err := os.Create(jsonPath)
	if err != nil {
		return err
	}
	if err := report.WriteJSON(f, sd); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	packetPath := filepath.Join(o.Out, "umbra.packet.md")
	pf, err := os.Create(packetPath)
	if err != nil {
		return err
	}
	if err := report.Packet(pf, sd); err != nil {
		pf.Close()
		return err
	}
	if err := pf.Close(); err != nil {
		return err
	}

	htmlPath := filepath.Join(o.Out, "umbra.html")
	hf, err := os.Create(htmlPath)
	if err != nil {
		return err
	}
	if err := report.HTML(hf, sd); err != nil {
		hf.Close()
		return err
	}
	if err := hf.Close(); err != nil {
		return err
	}

	if o.Format == "table" || o.Format == "html" {
		fmt.Printf("packet  %s   report %s\n", packetPath, htmlPath)
	}
	return nil
}

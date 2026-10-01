package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sf966gpm/mssh/internal/sshconfig"
)

func main() {
	path, err := sshconfig.ConfigPath()
	if err != nil {
		fail(err)
	}

	config, err := sshconfig.ParseFileExpanded(path)
	if err != nil {
		fail(err)
	}

	printConfig(config)
	if len(config.Diagnostics) > 0 {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "MSSH:", err)
	os.Exit(1)
}

func printConfig(config *sshconfig.Config) {
	fmt.Println("Read:", config.Path)
	for _, host := range config.Hosts {
		fmt.Printf("Host: %s (%s:%d)\n", strings.Join(host.Patterns, " "), host.Path, host.Line)
		for _, field := range host.Fields {
			fmt.Printf("  %s: %s (%s:%d)\n", field.Name, field.DisplayValue(), field.Path, field.Line)
		}
	}
	for _, include := range config.Includes {
		fmt.Printf("Include (%s:%d): %s\n", include.Path, include.Line, include.Pattern)
	}
	for _, diagnostic := range config.Diagnostics {
		fmt.Printf("Diagnostic: %s:%d: %s\n", diagnostic.File, diagnostic.Line, diagnostic.Message)
	}
}

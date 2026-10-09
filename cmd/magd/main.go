package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"

	"magd/internal/ast"
	"magd/internal/lexer"
	"magd/internal/parser"
)

// Exit codes follow sysexits.h, so a shell can tell a usage mistake apart
// from a bad script.
const (
	exitOK    = 0
	exitUsage = 64 // EX_USAGE an Error caused by misuse of the user
	exitData  = 65 // EX_DATAERR an Error caused by invalid data
)

// usage prints help text. The caller decides the exit code, so -h and a bad
// flag can report differently.
func usage(out io.Writer) {
	fmt.Fprintf(out, `MAGD - an interpreter for the MAGD programming language.

Usage:
  %[1]s [flags] [file]

With a file, runs it and exits. With no file, starts a REPL.

Examples:
  %[1]s                  start the REPL
  %[1]s script.magd      run a script
`, os.Args[0])
}

func run(source string) bool {
	l := lexer.NewLexer(source)
	tokens := l.ScanTokens()

	if errs := l.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		return false
	}

	p := parser.NewParser(tokens)
	expression, err := p.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, "internal error:", err)
		return false
	}

	// Parse recovers from syntax errors and records them instead of
	// returning one, so the real diagnostics live in Errors().
	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		return false
	}

	printed, err := (&ast.Printer{}).Print(expression)
	if err != nil {
		fmt.Fprintln(os.Stderr, "internal error:", err)
		return false
	}

	fmt.Println(printed)
	return true
}

func runFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		os.Exit(exitData)
	}

	if !run(string(content)) {
		os.Exit(exitData)
	}
}

// TODO: migrate the REPL into it's own REPL package

func runPrompt() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "> ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		run(line)
	}

	err := scanner.Err()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read stdin failed: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	flag.Usage = func() { usage(os.Stderr) }
	// ContinueOnError so an unrecognised flag exits with exitUsage rather
	// than flag's default of 2, keeping every usage error identical.
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		// Parse already printed the problem and the usage text.
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(exitOK)
		}
		os.Exit(exitUsage)
	}

	args := flag.Args()
	if len(args) > 1 {
		usage(os.Stderr)
		os.Exit(exitUsage)
	}

	if len(args) == 1 {
		runFile(args[0])
		return
	}

	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Hello %s! This is the MAGD programming language!\n",
		user.Username)
	runPrompt()
}

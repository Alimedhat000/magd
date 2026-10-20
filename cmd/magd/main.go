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
	"magd/internal/eval"
	"magd/internal/interp"
	"magd/internal/repl"
)

// Exit codes follow sysexits.h, so a shell can tell a usage mistake apart
// from a bad script.
const (
	exitOK    = 0
	exitUsage = 64 // EX_USAGE an Error caused by misuse of the user
	exitData  = 65 // EX_DATAERR an Error caused by invalid data
)

const DEBUG = false

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
	statements, errs := interp.Compile(source)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		return false
	}

	// The evaluator resolves the statements before running them, so a name
	// that cannot be bound is reported before anything executes.
	if err := eval.NewEvaluator().Eval(statements); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}

	if DEBUG {
		printed, err := (&ast.Printer{}).Print(statements)
		if err != nil {
			fmt.Fprintln(os.Stderr, "internal error:", err)
			return false
		}

		fmt.Println(printed)
	}

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

func runPrompt() {
	s := repl.NewSession(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "> ")
		if !scanner.Scan() {
			break
		}
		s.Eval(scanner.Text())
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

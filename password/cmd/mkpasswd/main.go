package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"

	"go.rtnl.ai/x/password"
)

var (
	n         int
	policy    string
	list      bool
	show      bool
	config    bool
	noNewline bool
)

const usageText = `Usage: mkpasswd [options]

Generate random passwords from a password policy. Policies are read from the
configuration file (see -config) and fall back to the built-in "default" and
"basic" policies when no configuration file is found.

Options:
  -n int            number of passwords to generate (default 1)
  -p, -policy name  policy to use for password generation (default "default")
  -N, -no-lf        do not output a newline after each password
  -l, -list         list the names of the available policies
  -s, -show         print the selected policy as JSON instead of a password
  -c, -config       print the path to the policies configuration file
  -h, -help         print this help message
`

const (
	policyDefault  = "default"
	policyUsage    = "policy to use for password generation"
	listUsage      = "list the names of the available policies"
	showUsage      = "print the selected policy as JSON"
	configUsage    = "print the path to the policies configuration file"
	noNewlineUsage = "do not output a newline after each password"
)

func init() {
	flag.Usage = usage
	flag.IntVar(&n, "n", 1, "number of passwords to generate")
	flag.StringVar(&policy, "p", policyDefault, policyUsage)
	flag.StringVar(&policy, "policy", policyDefault, policyUsage)
	flag.BoolVar(&noNewline, "N", false, noNewlineUsage)
	flag.BoolVar(&noNewline, "no-lf", false, noNewlineUsage)
	flag.BoolVar(&list, "l", false, listUsage)
	flag.BoolVar(&list, "list", false, listUsage)
	flag.BoolVar(&show, "s", false, showUsage)
	flag.BoolVar(&show, "show", false, showUsage)
	flag.BoolVar(&config, "c", false, configUsage)
	flag.BoolVar(&config, "config", false, configUsage)

}

func main() {
	flag.Parse()

	var err error
	switch {
	case list:
		err = listPolicies()
	case show:
		err = showPolicy()
	case config:
		err = showConfigPath()
	default:
		err = generatePasswords()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// Prints the names of all available policies in alphabetical order.
func listPolicies() (err error) {
	var policies map[string]*password.Policy
	if policies, err = password.LoadAll(); err != nil {
		return err
	}

	for _, name := range slices.Sorted(maps.Keys(policies)) {
		fmt.Fprintf(os.Stdout, "- %s\n", name)
	}
	return nil
}

// Prints the selected policy as indented JSON.
func showPolicy() (err error) {
	var selected *password.Policy
	if selected, err = password.Load(policy); err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(selected)
}

// Prints the path to the policies configuration file.
func showConfigPath() error {
	fmt.Fprintln(os.Stdout, password.ConfigPath())
	return nil
}

// Generates n passwords with the selected policy, writing each one to stdout.
func generatePasswords() (err error) {
	var selected *password.Policy
	if selected, err = password.Load(policy); err != nil {
		return err
	}

	for range n {
		var pw string
		if pw, err = selected.Generate(); err != nil {
			return err
		}

		if noNewline {
			fmt.Fprint(os.Stdout, pw)
		} else {
			fmt.Fprintln(os.Stdout, pw)
		}
	}
	return nil
}

// Prints the help text and a compact description of the available options.
func usage() {
	fmt.Fprint(flag.CommandLine.Output(), usageText)
}

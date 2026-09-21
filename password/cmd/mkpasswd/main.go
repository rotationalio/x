package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

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

const (
	policyDefault = "default"
	policyUsage   = "policy to use for password generation"
)

func init() {
	flag.IntVar(&n, "n", 1, "number of passwords to generate")
	flag.StringVar(&policy, "p", policyDefault, policyUsage)
	flag.StringVar(&policy, "policy", policyDefault, policyUsage)
	flag.BoolVar(&list, "list", false, "list available policies")
	flag.BoolVar(&show, "show", false, "show a policy")
	flag.BoolVar(&config, "config", false, "show the path to the configuration file")
	flag.BoolVar(&noNewline, "N", false, "do not output a newline after each password")
}

func main() {
	flag.Parse()
	switch {
	case list:
		policies, err := password.LoadAll()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		for name := range policies {
			fmt.Fprintf(os.Stdout, "- %s\n", name)
		}
	case show:
		policy, err := password.Load(policy)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(policy); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	case config:
		fmt.Println(password.ConfigPath())
	default:
		for range n {
			password, err := password.Generate(policy)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			if noNewline {
				fmt.Fprint(os.Stdout, password)
			} else {
				fmt.Println(password)
			}
		}
	}
}

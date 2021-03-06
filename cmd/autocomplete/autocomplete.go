// Package autocomplete provides helpers for shell completion
package autocomplete

import (
	"fmt"
	"os"
	"strings"

	"github.com/blinkhealth/go-config-yourself/internal/yaml"

	pvd "github.com/blinkhealth/go-config-yourself/pkg/provider"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

// CommandAutocomplete is the function that autocompletes the main command
func CommandAutocomplete(ctx *cli.Context) {
	firstArg := ctx.Args().Get(0)
	if ctx.NArg() == 0 || ctx.NArg() == 1 && firstArg[0] == '-' {
		ListAllFlags(ctx)
	}

	for _, cmd := range ctx.App.VisibleCommands() {
		if firstArg == "" || strings.HasPrefix(cmd.Name, firstArg) {
			if os.Getenv("_CLI_ZSH_AUTOCOMPLETE_HACK") == "1" {
				fmt.Printf("%s:%s\n", cmd.Name, cmd.Usage)
			} else {
				fmt.Println(cmd.Name)
			}
		}
	}
}

//ListKeys lists keys at a given keypath
func ListKeys(ctx *cli.Context) int {
	configPath := strings.Replace(ctx.Args().Get(0), "~", os.Getenv("HOME"), 1)
	configFile, err := yaml.FromPathname(configPath)
	if err != nil {
		log.Debug(err)
		return 99
	}

	if ctx.NArg() > 0 {
		keyPath := os.Getenv("CUR")
		keys, err := possibleSubKeys(keyPath, configFile)
		if err != nil {
			return 99
		}

		format := "%s"
		if kp := strings.Split(keyPath, "."); len(kp) > 1 {
			root := strings.Join(kp[0:len(kp)-1], ".")
			format = fmt.Sprintf("%s.%s", root, format)
		}

		log.Debugf("keys: %s", keys)
		for _, key := range keys {
			fmt.Println(fmt.Sprintf(format, key))
		}
	}
	return 0
}

// ListProviderFlags lists available provider flags for an autocomplete context
func ListProviderFlags(ctx *cli.Context) (keepGoing bool) {
	query := os.Getenv("CUR")

	if query, ok := LastFlagIs("provider"); !ctx.IsSet("provider") && ok {
		found := false
		for _, provider := range pvd.ProviderList {
			if query == provider {
				found = true
				break
			}

			if query != "" && !strings.HasPrefix(provider, query) {
				continue
			}

			fmt.Println(provider)
		}

		if !found {
			return false
		}
	}

	if ctx.NArg() == 0 || len(query) > 0 && query[0] == '-' {
		ListAllFlags(ctx)
	}

	return true
}

// ListAllFlags all possible flags
func ListAllFlags(ctx *cli.Context) {
	var flags []cli.Flag
	if ctx.Command != nil && ctx.Command.Name != "" {
		flags = ctx.Command.VisibleFlags()
	} else {
		flags = ctx.App.VisibleFlags()
	}

	query := ""
	if strings.HasPrefix(os.Getenv("CUR"), "-") {
		query = strings.TrimLeft(os.Getenv("CUR"), "-")
	}

	isZSH := os.Getenv("_CLI_ZSH_AUTOCOMPLETE_HACK")
	for _, f := range flags {
		name := f.Names()[0]
		if query != "" && !strings.HasPrefix(name, query) {
			continue
		}
		// if name == "init-completion" {
		// 	continue
		// }

		_, isRepeatable := f.(*cli.StringSliceFlag)

		if isRepeatable || !ctx.IsSet(name) {
			if isZSH == "1" {
				fmt.Printf("--%s:%s\n", name, flagDescription(f))
			} else {
				fmt.Printf("--%s\n", name)
			}
		}
	}
}

// LastFlagIs tells wether we're completing a flag or not
func LastFlagIs(flagName string) (query string, ok bool) {
	args := validArgs()
	argLen := len(args)
	if argLen < 1 {
		return
	}

	flag := fmt.Sprintf("--%s", flagName)
	if arg := args[argLen-1]; arg == flag {
		ok = true
		return
	}

	if argLen > 1 {
		if arg := args[argLen-2]; arg == flag {
			query = args[argLen-1]
			ok = true
			return
		}
	}

	return
}

func validArgs() (validArgs []string) {
	for _, arg := range os.Args {
		if arg == "--" || arg == "--generate-bash-completion" {
			break
		}

		validArgs = append(validArgs, arg)
	}
	return
}

func flagDescription(f cli.Flag) (description string) {
	switch typedFlag := f.(type) {
	case *cli.StringFlag:
		description = typedFlag.Usage
	case *cli.BoolFlag:
		description = typedFlag.Usage
	case *cli.GenericFlag:
		description = typedFlag.Usage
	case *cli.StringSliceFlag:
		description = typedFlag.Usage
	default:
		log.Warningf("%s: %T", f.Names(), typedFlag)
	}
	if description != "" {
		description = strings.SplitN(description, ".", 2)[0]
		description = strings.ReplaceAll(description, ":", "—")
	}
	return
}

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blinkhealth/go-config-yourself/cmd/autocomplete"
	"github.com/blinkhealth/go-config-yourself/cmd/util"
	"github.com/blinkhealth/go-config-yourself/internal/input"
	"github.com/blinkhealth/go-config-yourself/pkg/file"
	log "github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v2"
)

func init() {
	App.Commands = append(App.Commands, &cli.Command{
		Name:        "set",
		Before:      beforeCommand,
		Aliases:     []string{"edit"},
		Usage:       "Set a config value in CONFIG_FILE",
		ArgsUsage:   "CONFIG_FILE KEYPATH",
		Description: "Prompts for a value from stdin, encrypts it and saves it at KEYPATH of CONFIG_FILE",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:   "keypath",
				Value:  "",
				Usage:  "Used internally by the app",
				Hidden: true,
			},
			&cli.BoolFlag{
				Name:    "plain-text",
				Value:   false,
				Usage:   "save the value as plain text instead of encrypting",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "input-file",
				Value:   "",
				Usage:   "Read this file instead of prompting for input",
				Aliases: []string{"i"},
			},
		},
		ShellComplete: func(ctx *cli.Context) {
			if ctx.NArg() == 0 {
				autocomplete.ListAllFlags(ctx)
				if !ctx.IsSet("input-file") {
					if _, ok := autocomplete.LastFlagIs("input-file"); ok {
						os.Exit(1)
					}
				}
			}

			if ctx.NArg() == 1 {
				autocomplete.ListKeys(ctx)
			}

			os.Exit(1)
		},
		Action: set,
	})
}

const noCryptoError = "Unable to store an encrypted value for '%s', use --plain-text to store as a non-encrypted value"

//Set saves an encrypted or plaintext value on the file
func set(ctx *cli.Context) error {
	keyPath := ctx.String("keypath")
	isPlainText := ctx.Bool("plain-text")
	if !configFile.HasCrypto() && !isPlainText {
		// Won't store a plaintext unless we very explicitly ask for it
		message := fmt.Sprintf(noCryptoError, keyPath)
		return Exit(errors.New(message), ExitCodeInputError)
	}

	var plainText []byte
	var err error
	if file := ctx.String("input-file"); file != "" {
		plainText, err = input.ReadFile(file)
	} else {
		prompt := fmt.Sprintf("Enter value for “%s”", keyPath)
		plainText, err = input.ReadSecret(prompt, !isPlainText)
	}

	if err != nil {
		return Exit(err, ExitCodeInputError)
	}

	if isPlainText {
		err = configFile.VeryInsecurelySetPlaintext(keyPath, plainText)
	} else {
		err = configFile.Set(keyPath, plainText)
	}

	if err != nil {
		return Exit(fmt.Sprintf("Could not set %s: %s", keyPath, err), ExitCodeToolError)
	}

	target := ctx.Args().Get(0)
	if err := util.SerializeAndWrite(target, configFile); err != nil {
		return Exit(err, ExitCodeToolError)
	}

	log.Infof("Value set at %s", keyPath)
	// update defaults file if write was successful
	updateDefaultsFile(target, keyPath)

	return nil
}

func updateDefaultsFile(target string, keyPath string) {
	if strings.HasPrefix(filepath.Base(target), "default") {
		return
	}
	configFolder := filepath.Dir(target)
	extension := filepath.Ext(target)
	if configFolder != "" {
		configFolder = fmt.Sprintf("%s/", configFolder)
	}

	for _, name := range []string{"default", "defaults"} {
		candidate := fmt.Sprintf("%s%s%s", configFolder, name, extension)
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			log.Debugf("Found defaults file: %s", candidate)
			defaultsFile, err := file.Load(candidate)
			if err == nil {
				_, err := defaultsFile.Get(keyPath)
				if err != nil && strings.Contains(err.Error(), "Could not find a value") {
					if err := defaultsFile.VeryInsecurelySetPlaintext(keyPath, nil); err == nil {
						// Don't panic if it doesn't get updated
						if util.SerializeAndWrite(candidate, defaultsFile) != nil {
							log.Infof("Updated value in defaults file %s", candidate)
						}
						return
					}
				}
			}
		}
	}
}
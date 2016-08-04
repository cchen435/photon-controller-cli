// Copyright (c) 2016 VMware, Inc. All Rights Reserved.
//
// This product is licensed to you under the Apache License, Version 2.0 (the "License").
// You may not use this product except in compliance with the License.
//
// This product may include a number of subcomponents with separate copyright notices and
// license terms. Your use of these subcomponents is subject to the terms and conditions
// of the subcomponent's license, as noted in the LICENSE file.

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/vmware/photon-controller-go-sdk/photon"
	"github.com/vmware/photon-controller-cli/Godeps/_workspace/src/github.com/vmware/photon-controller-go-sdk/photon/lightwave"

	"os"
	"text/tabwriter"

	"github.com/vmware/photon-controller-cli/photon/client"
	"github.com/vmware/photon-controller-cli/photon/configuration"
)

// Create a cli.command object for command "auth"
func GetAuthCommand() cli.Command {
	command := cli.Command{
		Name:  "auth",
		Usage: "options for auth",
		Subcommands: []cli.Command{
			{
				Name:  "show",
				Usage: "Display auth info",
				Action: func(c *cli.Context) {
					err := show(c)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "show-login-token",
				Usage: "Show login token",
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "raw, r",
						Usage: "raw, prints the full JSON text of the token",
					},
				},
				Action: func(c *cli.Context) {
					err := showLoginToken(c)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
			{
				Name:  "get-api-tokens",
				Usage: "Retrieve access and refresh tokens",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "username, u",
						Usage: "username, if this is provided a password needs to be provided as well",
					},
					cli.StringFlag{
						Name:  "password, p",
						Usage: "password, if this is provided a username needs to be provided as well",
					},
					cli.BoolFlag{
						Name:  "raw, r",
						Usage: "raw, prints the full JSON text of the token",
					},
				},
				Action: func(c *cli.Context) {
					err := getApiTokens(c)
					if err != nil {
						log.Fatal(err)
					}
				},
			},
		},
	}
	return command
}

// Get auth info
func show(c *cli.Context) error {
	err := checkArgNum(c.Args(), 0, "auth show")
	if err != nil {
		return err
	}
	client.Esxclient, err = client.GetClient(c.GlobalIsSet("non-interactive"))
	if err != nil {
		return err
	}

	auth, err := client.Esxclient.Auth.Get()
	if err != nil {
		return err
	}

	err = printAuthInfo(auth, c.GlobalIsSet("non-interactive"))
	if err != nil {
		return err
	}

	return nil
}

func showLoginToken(c *cli.Context) error {
	return showLoginTokenWriter(c, os.Stdout, nil)
}

// Handles show-login-token, which shows the current login token, if any
func showLoginTokenWriter(c *cli.Context, w io.Writer, config *configuration.Configuration) error {
	err := checkArgNum(c.Args(), 0, "auth show-login-token")
	if err != nil {
		return err
	}

	if config == nil {
		config, err = configuration.LoadConfig()
		if err != nil {
			return err
		}
	}

	if config.Token == "" {
		err = fmt.Errorf("No login token available")
		return err
	}
	if !c.GlobalIsSet("non-interactive") {
		raw := c.Bool("raw")
		if raw {
			dumpTokenDetailsRaw(w, "Login Access Token", config.Token)
		} else {
			dumpTokenDetails(w, "Login Access Token", config.Token)
		}
	} else {
		fmt.Fprintf(w, "%s\n", config.Token)
	}
	return nil
}

func getApiTokens(c *cli.Context) error {
	err := checkArgNum(c.Args(), 0, "auth get-tokens")
	if err != nil {
		return err
	}

	username := c.String("username")
	password := c.String("password")

	if !c.GlobalIsSet("non-interactive") {
		username, err = askForInput("User name (username@tenant): ", username)
		if err != nil {
			return err
		}
		password, err = askForInput("Password: ", password)
		if err != nil {
			return err
		}
	}

	if len(username) == 0 || len(password) == 0 {
		return fmt.Errorf("Please provide username/password")
	}

	client.Esxclient, err = client.GetClient(c.GlobalIsSet("non-interactive"))
	if err != nil {
		return err
	}

	tokens, err := client.Esxclient.Auth.GetTokensByPassword(username, password)
	if err != nil {
		return err
	}

	if !c.GlobalIsSet("non-interactive") {
		raw := c.Bool("raw")
		if raw {
			dumpTokenDetailsRaw(os.Stdout, "API Access Token", tokens.AccessToken)
			dumpTokenDetailsRaw(os.Stdout, "API Refresh Token", tokens.RefreshToken)
		} else {
			dumpTokenDetails(os.Stdout, "API Access Token", tokens.AccessToken)
			dumpTokenDetails(os.Stdout, "API Refresh Token", tokens.RefreshToken)
		}
	} else {
		fmt.Printf("%s\t%s", tokens.AccessToken, tokens.RefreshToken)
	}

	return nil
}

// Print out auth info
func printAuthInfo(auth *photon.AuthInfo, isScripting bool) error {
	if isScripting {
		fmt.Printf("%t\t%s\t%d\n", auth.Enabled, auth.Endpoint, auth.Port)
	} else {
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 4, 4, 2, ' ', 0)
		fmt.Fprintf(w, "Enabled\tEndpoint\tPort\n")
		fmt.Fprintf(w, "%t\t%s\t%d\n", auth.Enabled, auth.Endpoint, auth.Port)
		err := w.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

// A JSON web token is a set of Base64 encoded strings separated by a period (.)
// When decoded, it will either be JSON text or a signature
// Here we decode the strings into a single token structure and print the most
// useful fields. We do not print the signature.
func dumpTokenDetails(w io.Writer, name string, encodedToken string) {
	jwtToken := lightwave.ParseTokenDetails(encodedToken)

	fmt.Fprintf(w, "%s:\n", name)
	fmt.Fprintf(w, "\tSubject: %s\n", jwtToken.Subject)
	fmt.Fprintf(w, "\tGroups: ")
	if jwtToken.Groups == nil {
		fmt.Fprintf(w, "<none>\n")
	} else {
		fmt.Fprintf(w, "%s\n", strings.Join(jwtToken.Groups, ", "))
	}
	fmt.Fprintf(w, "\tIssued: %s\n", timestampToString(jwtToken.IssuedAt*1000))
	fmt.Fprintf(w, "\tExpires: %s\n", timestampToString(jwtToken.Expires*1000))
	fmt.Fprintf(w, "\tToken: %s\n", encodedToken)
}

// A JSON web token is a set of Base64 encoded strings separated by a period (.)
// When decoded, it will either be JSON text or a signature
// Here we print the full JSON text. We do not print the signature.
func dumpTokenDetailsRaw(w io.Writer, name string, encodedToken string) {
	jsonStrings, err := lightwave.ParseRawTokenDetails(encodedToken)
	if err != nil {
		fmt.Fprintf(w, "<unparseable>\n")
	}

	fmt.Fprintf(w, "%s:\n", name)
	for _, jsonString := range jsonStrings {
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, []byte(jsonString), "", "  ")
		if err == nil {
			fmt.Fprintf(w, "%s\n", string(prettyJSON.Bytes()))
		}
	}
	fmt.Fprintf(w, "Token: %s\n", encodedToken)
}

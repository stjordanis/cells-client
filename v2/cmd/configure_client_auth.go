package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

const authTypeClientAuth = "client-auth"

var (
	configHost       string
	configUser       string
	configPwd        string
	configLabel      string
	configSkipVerify bool
)

var configureClientAuthCmd = &cobra.Command{
	Use:   authTypeClientAuth,
	Short: "Connect to the server directly using the Client Credentials",
	Long: `
Launch an interactive process to gather necessary client information to configure a connection to a running Pydio Cells server instance.

You must use a valid userName/password with enough permissions to achieve what you want on the server.

Please beware that this sensitive information will be stored in clear text if you do not have a **correctly configured and running** keyring on your client machine.

You can also go through the whole process in a non-interactive manner by using the provided flags.
`,
	Run: func(cm *cobra.Command, args []string) {

		var err error
		newConf := &rest.CecConfig{}

		list, err := rest.GetConfigList()
		if err != nil {
			list = &rest.ConfigList{Configs: map[string]*rest.CecConfig{}}
		}

		if notEmpty(configHost) == nil && notEmpty(configUser) == nil && notEmpty(configPwd) == nil {
			newConf, label, err = nonInteractive(list)
		} else {
			newConf, label, err = interactive(list)
		}
		if err != nil {
			log.Fatal(err)
		}

		// Now save config!
		if !skipKeyring {
			if err := rest.ConfigToKeyring(newConf); err != nil {
				fmt.Println(promptui.IconWarn + " Cannot save token in keyring! " + err.Error())
			}
		}

		if err := list.Add(label, newConf); err != nil {
			log.Fatal(err)
		}

		if err := list.SaveConfigFile(); err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved under label %s\n", promptui.IconGood, color.New(color.Bold).Sprint(label))
			fmt.Printf("%s Configuration saved, you can now use the client to interact with %s.\n", promptui.IconGood, newConf.Url)
		}
	},
}

func interactive(currentList *rest.ConfigList) (newConf *rest.CecConfig, label string, err error) {

	var e error
	newConf = &rest.CecConfig{}
	// PROMPT URL
	p := promptui.Prompt{Label: "Server Address (provide a valid URL)", Validate: validUrl}
	if newConf.Url, e = p.Run(); e != nil {
		return newConf, label, e
	}
	newConf.Url = strings.TrimSpace(newConf.Url)

	u, e := url.Parse(newConf.Url)
	if e != nil {
		return nil, label, e
	}

	if u.Scheme == "https" {
		// PROMPT SKIP VERIFY
		p2 := promptui.Select{Label: "Skip SSL Verification? (not recommended)", Items: []string{"No", "Yes"}}
		if _, y, e := p2.Run(); y == "Yes" && e == nil {
			newConf.SkipVerify = true
		}
	}

	// PROMPT LOGIN
	p = promptui.Prompt{
		Label:    "User Login",
		Validate: notEmpty,
	}
	if newConf.User, e = p.Run(); e != nil {
		return newConf, label, e
	}

	// PROMPT PASSWORD
	p = promptui.Prompt{Label: "User Password", Mask: '*', Validate: notEmpty}
	if newConf.Password, e = p.Run(); e != nil {
		return newConf, label, e
	}

	// Test a simple PING with this config before saving
	fmt.Println(promptui.IconWarn + " Testing this configuration before saving")
	rest.DefaultConfig = newConf
	if _, _, e := rest.GetApiClient(); e != nil {
		fmt.Println("\r" + promptui.IconBad + " Could not connect to server, please recheck your configuration")
		fmt.Println("Cause: " + e.Error())
		return newConf, label, fmt.Errorf("test connection failed")
	}
	fmt.Println("\r" + promptui.IconGood + " Successfully logged to server")
	bold := color.New(color.Bold)

	label = "default"

	var found bool
	if currentList != nil && currentList.Configs != nil {
		for k, v := range currentList.Configs {
			if v.Url == newConf.Url && v.TokenUser == newConf.TokenUser {
				label = k
				found = true
				break
			}
		}
		if !found {
			var i int
			for {
				if i > 0 {
					label = fmt.Sprintf("default-%d", i)
				}
				if _, ok := currentList.Configs[label]; !ok {
					break
				}
				i++
			}
		}
	}

	// if a config with the same parameters (url and username) is found we return the same config for an update
	if found {
		return newConf, label, nil
	}
	pLabel := promptui.Select{Label: fmt.Sprintf("Would you like to use this default label - %s", bold.Sprint(label)), Items: []string{"Yes", "No"}, Size: 2}
	if _, y, err := pLabel.Run(); y == "No" && err == nil {
		p5 := promptui.Prompt{Label: "Enter the new label for the config", Validate: notEmpty}
		label, err = p5.Run()
		if err != nil {
			log.Fatal(err)
		}
	}

	return newConf, label, nil
}

func nonInteractive(currentList *rest.ConfigList) (newConf *rest.CecConfig, label string, err error) {

	newConf = &rest.CecConfig{}
	// if label is empty (in case the flag is used without a value)
	if label == "" {
		label = "default"
	}
	label = configLabel
	newConf.Url = configHost
	newConf.User = configUser
	newConf.Password = configPwd
	newConf.SkipVerify = configSkipVerify

	// if configuration with the same label exists put a default label
	var found bool
	if currentList != nil && currentList.Configs != nil {
		for k, v := range currentList.Configs {
			if v.Url == newConf.Url && v.User == newConf.User {
				label = k
				found = true
				break
			}
		}
		if !found {
			var i int
			for {
				if i > 0 {
					label = fmt.Sprintf("default-%d", i)
				}
				if _, ok := currentList.Configs[label]; !ok {
					break
				}
				i++
			}
		}
	}

	if found {
		return newConf, label, fmt.Errorf("A configuration with the same parameters already exists under the label [%s]", label)
	}

	// Insure values are legal
	if err := validUrl(newConf.Url); err != nil {
		return nil, "", fmt.Errorf("URL %s is not valid: %s", newConf.Url, err.Error())
	}

	// Test a simple ping with this config before saving
	rest.DefaultConfig = newConf
	if _, _, e := rest.GetApiClient(); e != nil {
		return nil, "", fmt.Errorf("could not connect to newly configured server failed, cause: %s", e.Error())
	}

	return newConf, label, nil
}

func validUrl(input string) error {
	// Warning: trim must also be performed when retrieving the final value.
	// Here we only validate that the trimed input is valid, but do not modify it.
	input = strings.Trim(input, " ")
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty")
	}
	u, e := url.Parse(input)
	if e != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("Please, provide a valid URL")
	}
	return nil
}

func notEmpty(input string) error {
	if len(input) == 0 {
		return fmt.Errorf("field cannot be empty")
	}
	return nil
}

func init() {

	flags := configureClientAuthCmd.PersistentFlags()

	flags.StringVarP(&configHost, "url", "u", "", "HTTP URL to server")
	flags.StringVarP(&configUser, "login", "l", "", "User login")
	flags.StringVarP(&configPwd, "password", "p", "", "User password")
	flags.StringVarP(&configLabel, "label", "", "default", "Configuration label")
	flags.BoolVar(&configSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	configureCmd.AddCommand(configureClientAuthCmd)
}

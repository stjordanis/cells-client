package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

const authTypeOAuth = "oauth"

var (
	oAuthUrl        string
	oAuthIdToken    string
	oAuthSkipVerify bool
	callbackPort    = 3000
)

type oAuthHandler struct {
	// Input
	done   chan bool
	closed bool
	state  string
	// Output
	code string
	err  error
}

var configureOAuthCmd = &cobra.Command{
	Use:   authTypeOAuth,
	Short: "User OAuth2 to login to server",
	Long:  `Configure Authentication using OAuth2`,
	Run: func(cm *cobra.Command, args []string) {

		var err error
		var label string
		newConf := &rest.CecConfig{}

		list, err := rest.GetConfigList()
		if err != nil {
			list = &rest.ConfigList{Configs: map[string]*rest.CecConfig{}}
		}

		if oAuthUrl != "" && oAuthIdToken != "" {
			if newConf, label, err = oAuthNonInteractive(list); err != nil {
				log.Fatal(err)
			}
		} else {
			if newConf, label, err = oAuthInteractive(list); err != nil {
				log.Fatal(err)
			}
		}

		// Now save config!
		if !skipKeyring {
			if err := rest.ConfigToKeyring(newConf); err != nil {
				fmt.Println(promptui.IconWarn + " Cannot save token in keyring! " + err.Error())
			}
		}

		// add config to the list
		if err = list.Add(label, newConf); err != nil {
			log.Fatal(err)
		}

		if err := list.SaveConfigFile(); err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved, you can now use the client to interact with %s.\n", promptui.IconGood, newConf.Url)
		}
	},
}

func (o *oAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if !o.closed {
			o.closed = true
			close(o.done)
		}
	}()
	values := r.URL.Query()
	if values.Get("state") != o.state {
		o.err = fmt.Errorf("wrong state received")
		return
	}
	if values.Get("code") == "" {
		o.err = fmt.Errorf("empty code received")
		return
	}
	o.code = values.Get("code")
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<p style="display: flex;height: 100%;width: 100%;align-items: center;justify-content: center;font-family: sans-serif;color: #607D8B;font-size: 20px;">
			You can now close this window and go back to your shell!
		</p>
		<script type="text/javascript">window.close();</script>
	`))
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandString(n int) string {
	b := make([]byte, n)
	rand.Seed(time.Now().Unix())
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func oAuthInteractive(currentList *rest.ConfigList) (newConf *rest.CecConfig, label string, err error) {
	var e error
	newConf = &rest.CecConfig{}
	// PROMPT URL
	p := promptui.Prompt{
		Label:    "Server Address (provide a valid URL)",
		Validate: validUrl,
		Default:  "",
	}

	if newConf.Url, e = p.Run(); e != nil {
		return nil, "", e
	} else {
		newConf.Url = strings.TrimSpace(newConf.Url)
	}
	u, e := url.Parse(newConf.Url)
	if e != nil {
		return nil, "", e
	}
	if u.Scheme == "https" {
		// PROMPT SKIP VERIFY
		p2 := promptui.Select{Label: "Skip SSL Verification? (not recommended)", Items: []string{"No", "Yes"}}
		if _, y, e := p2.Run(); y == "Yes" && e == nil {
			newConf.SkipVerify = true
		}
	}

	newConf.ClientKey = "cells-client"
	pE := promptui.Select{Label: "Do you want to edit OAuth client data (defaults generally work)?", Items: []string{"Use defaults", "Edit OAuth client"}}
	if _, v, e := pE.Run(); e == nil && v != "Use defaults" {
		// PROMPT CLIENT ID
		p = promptui.Prompt{
			Label:     "OAuth APP ID (found in your server pydio.json)",
			Validate:  notEmpty,
			Default:   "cells-client",
			AllowEdit: true,
		}
		if newConf.ClientKey, e = p.Run(); e != nil {
			return nil, "", e
		}
		p = promptui.Prompt{Label: "OAuth APP Secret (leave empty for a public client)", Default: "", Mask: '*'}
		newConf.ClientSecret, _ = p.Run()
	}

	openBrowser := true
	p3 := promptui.Select{Label: "Can you open a browser on this computer? If not, you will make the authentication process by copy/pasting", Items: []string{"Yes", "No"}}
	if _, v, e := p3.Run(); e == nil && v == "No" {
		openBrowser = false
	}

	// Check default port availability: Note that we do not offer option to change the port,
	// because it would also require impacting the registered client in the pydio.json
	// of the server that is not an acceptable option.
	avail := isPortAvailable(callbackPort, 10)
	if !avail {
		fmt.Printf("Warning: default port %d is not available on this machine, "+
			"you thus won't be able to automatically complete the auth code flow with the implicit callback URL."+
			"Please free this port or choose the copy/paste solution.\n", callbackPort)
		openBrowser = false
	}

	// Starting authentication process
	var returnCode string
	state := RandString(16)
	directUrl, callbackUrl, err := rest.OAuthPrepareUrl(newConf.Url, newConf.ClientKey, newConf.ClientSecret, state, openBrowser)
	if err != nil {
		log.Fatal(err)
	}
	if openBrowser {
		fmt.Println("Opening URL", directUrl)
		go open.Run(directUrl)
		h := &oAuthHandler{
			done:  make(chan bool),
			state: state,
		}
		srv := &http.Server{Addr: fmt.Sprintf(":%d", callbackPort)}
		srv.Handler = h
		go func() {
			<-h.done
			srv.Shutdown(context.Background())
		}()
		srv.ListenAndServe()
		if h.err != nil {
			log.Fatal("Could not correctly connect", h.err)
		}
		returnCode = h.code
	} else {
		col := color.FgLightRed.Render
		fmt.Println("Please copy and paste this URL in a browser", col(directUrl))
		var err error
		pr := promptui.Prompt{
			Label:    "Please Paste the code returned to you in the browser",
			Validate: notEmpty,
		}
		returnCode, err = pr.Run()
		if err != nil {
			log.Fatal("Could not read code!")
		}
	}

	fmt.Println(promptui.IconGood + " Now exchanging the code for a valid IdToken")
	if err := rest.OAuthExchangeCode(newConf, returnCode, callbackUrl); err != nil {
		log.Fatal(err)
	}
	fmt.Println(promptui.IconGood + " Successfully Received Token!")

	// Test a simple PING with this config before saving!
	fmt.Println(promptui.IconWarn + " Testing this configuration before saving")
	rest.DefaultConfig = &rest.CecConfig{
		SdkConfig: newConf.SdkConfig,
	}
	if _, _, e := rest.GetApiClient(); e != nil {
		fmt.Println("\r" + promptui.IconBad + " Could not connect to server, please recheck your configuration")
		fmt.Printf("Id_token: [%s]\n", newConf.IdToken)

		fmt.Println("Cause: " + e.Error())
		return nil, "", fmt.Errorf("test connection failed")
	}
	fmt.Println("\r" + promptui.IconGood + fmt.Sprintf(" Successfully logged to server, token will be refreshed at %v", time.Unix(int64(newConf.TokenExpiresAt), 0)))
	bold := color.New(color.Bold)

	fmt.Println("\r"+promptui.IconGood+" "+"You are logged-in as user:", bold.Sprintf("%s", rest.CurrentUser))
	newConf.TokenUser = rest.CurrentUser

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

	if found {
		return newConf, label, nil
	}
	p4 := promptui.Select{Label: fmt.Sprintf("Would you like to use this default label - %s", bold.Sprint(label)), Items: []string{"Yes", "No"}, Size: 2}
	if _, y, err := p4.Run(); y == "No" && err == nil {
		p5 := promptui.Prompt{Label: "Enter the new label for the config", Validate: notEmpty}
		label, err = p5.Run()
		if err != nil {
			log.Fatal(err)
		}
	}

	return newConf, label, nil
}

// TODO finish integration of this mode
func oAuthNonInteractive(currentList *rest.ConfigList) (conf *rest.CecConfig, label string, err error) {
	conf = &rest.CecConfig{}
	conf.Url = oAuthUrl
	conf.IdToken = oAuthIdToken
	conf.SkipVerify = configSkipVerify

	// Insure values are legal
	if err = validUrl(conf.Url); err != nil {
		err = fmt.Errorf("URL %s is not valid: %s", conf.Url, err.Error())
		return
	}

	// Test a simple PING with this config before saving!
	rest.DefaultConfig.SdkConfig = conf.SdkConfig
	if _, _, e := rest.GetApiClient(); e != nil {
		err = fmt.Errorf("test connection to newly configured server failed")
		return
	}

	label = "default"
	var found bool
	if currentList != nil && currentList.Configs != nil {
		for k, v := range currentList.Configs {
			if v.Url == conf.Url && v.User == conf.TokenUser {
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

	conf.User = rest.CurrentUser
	return
}

func isPortAvailable(port int, timeout int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func init() {

	flags := configureOAuthCmd.PersistentFlags()

	flags.StringVarP(&oAuthUrl, "url", "u", "", "HTTP URL to server")
	flags.StringVarP(&oAuthIdToken, "idToken", "t", "", "Valid IdToken")
	flags.BoolVar(&oAuthSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	configureCmd.AddCommand(configureOAuthCmd)
}

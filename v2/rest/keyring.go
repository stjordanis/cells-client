package rest

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService              = "com.pydio.cells-client"
	keyringIdTokenKey           = "IdToken"
	keyringClientCredentialsKey = "ClientCredentials"
)

func BuildKeyName(conf *CecConfig, clientType string) string {
	parsedURL, _ := url.Parse(conf.Url)
	// expecting name to look like this https://username@cells-test.your-files-your-rules.eu/::IdToken
	return fmt.Sprintf("%s://%s@%s::%s", parsedURL.Scheme, conf.TokenUser, parsedURL.Host, clientType)
}

// ConfigToKeyring tries to store tokens in local keychain and Remove them from the conf
func ConfigToKeyring(conf *CecConfig) error {

	// We use OAuth2 grant flow
	if conf.IdToken != "" && conf.RefreshToken != "" {
		// key := conf.Url + "::" + keyringIdTokenKey
		key := BuildKeyName(conf, keyringIdTokenKey)
		value := conf.IdToken + "__//__" + conf.RefreshToken
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.IdToken = ""
		conf.RefreshToken = ""
	}

	// We use client credentials
	if conf.ClientSecret != "" && conf.Password != "" {
		// key := conf.Url + "::" + keyringClientCredentialsKey
		key := BuildKeyName(conf, keyringClientCredentialsKey)
		value := conf.ClientSecret + "__//__" + conf.Password
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.ClientSecret = ""
		conf.Password = ""
	}

	return nil
}

// ConfigFromKeyring tries to find sensitive info inside local keychain and feed the conf.
func ConfigFromKeyring(conf *CecConfig) error {

	// If only client key and user name, consider Client Secret and password are in the keyring
	if conf.ClientKey != "" && conf.ClientSecret == "" && conf.User != "" && conf.Password == "" {
		if value, e := keyring.Get(keyringService, BuildKeyName(conf, keyringClientCredentialsKey)); e == nil {
			parts := strings.Split(value, "__//__")
			conf.ClientSecret = parts[0]
			conf.Password = parts[1]
		} else {
			return e
		}
	}

	// If no token, no user and no client key, consider tokens are stored in keyring
	if conf.IdToken == "" && conf.RefreshToken == "" && conf.User == "" && conf.Password == "" {
		if value, e := keyring.Get(keyringService, BuildKeyName(conf, keyringIdTokenKey)); e == nil {
			parts := strings.Split(value, "__//__")
			conf.IdToken = parts[0]
			conf.RefreshToken = parts[1]
		} else {
			return e
		}
	}
	return nil
}

// ClearKeyring removes sensitive info from local keychain, if they are present.
func ClearKeyring(conf *CecConfig) error {
	// Best effort to Remove known keys from keyring
	// TODO maybe check if at least one of the two has been found and deleted and otherwise print at least a warning
	// keyringClientCredentialsKey case
	if err := keyring.Delete(keyringService, BuildKeyName(conf, keyringClientCredentialsKey)); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}
	// keyringIdTokenKey case
	if err := keyring.Delete(keyringService, BuildKeyName(conf, keyringIdTokenKey)); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}
	return nil
}

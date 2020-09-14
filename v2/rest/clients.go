package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/go-openapi/strfmt"
	"github.com/shibukawa/configdir"

	cells_sdk "github.com/pydio/cells-sdk-go"
	"github.com/pydio/cells-sdk-go/client"
	"github.com/pydio/cells-sdk-go/transport"

	"github.com/pydio/cells-client/v2/common"
)

var (
	DefaultConfig  *CecConfig
	configFilePath string
)

type CecConfig struct {
	cells_sdk.SdkConfig
	TokenUser string
}

func GetConfigFilePath() string {
	if configFilePath != "" {
		return configFilePath
	}
	return DefaultConfigFilePath()
}

func SetConfigFilePath(confPath string) {
	configFilePath = confPath
}

func DefaultConfigFilePath() string {

	vendor := "Pydio"
	if runtime.GOOS == "linux" {
		vendor = "pydio"
	}
	appName := "cells-client"
	configDirs := configdir.New(vendor, appName)
	folders := configDirs.QueryFolders(configdir.Global)
	if len(folders) == 0 {
		folders = configDirs.QueryFolders(configdir.Local)
	}
	f := folders[0].Path
	if err := os.MkdirAll(f, 0777); err != nil {
		log.Fatal("Could not create local data dir - please check that you have the correct permissions for the folder -", f)
	}

	f = filepath.Join(f, "config.json")

	return f
}

// GetApiClient connects to the Pydio Cells server defined by this config, by sending an authentication
// request to the OIDC service to get a valid JWT (or taking the JWT from cache).
// It also returns a context to be used in subsequent requests.
func GetApiClient(anonymous ...bool) (context.Context, *client.PydioCellsRest, error) {

	anon := false
	if len(anonymous) > 0 && anonymous[0] {
		anon = true
	}
	DefaultConfig.CustomHeaders = map[string]string{"User-Agent": "cells-client/" + common.Version}
	c, t, e := transport.GetRestClientTransport(&DefaultConfig.SdkConfig, anon)
	if e != nil {
		return nil, nil, e
	}
	cl := client.New(t, strfmt.Default)
	return c, cl, nil

}

// SetUpEnvironment retrieves parameters and stores them in the DefaultConfig of the SDK.
// It also puts the sensible info in the server keyring if one is present.
// Note the precedence order (for each start of the app):
//  1) environment variables,
//  2) config files whose path is passed as argument of the start command
//  3) local config file (that are generated at first start with one of the 2 options above OR by calling the configure command.
func SetUpEnvironment(confPath string) error {

	DefaultConfig = new(CecConfig)
	// Use a config file
	if confPath != "" {
		SetConfigFilePath(confPath)
	}

	cecCfg := new(CecConfig)
	// Get config params from environment variables
	c, err := getSdkConfigFromEnv()
	if err != nil {
		return err
	}
	cecCfg.SdkConfig = c

	l, err := GetConfigList()
	if err != nil {
		log.Fatal(err)
	}

	if c.Url == "" {
		cecCfg = l.GetActiveConfig()

		// Retrieves sensible info from the keyring if one is presente
		err = ConfigFromKeyring(cecCfg)
		if err != nil {
			return err
		}

		// Refresh token if required
		if refreshed, err := RefreshIfRequired(&cecCfg.SdkConfig); refreshed {
			if err != nil {
				log.Fatal("Could not refresh authentication token:", err)
			}
			// Copy config as IdToken will be cleared and kept inside the keyring
			storeConfig := cecCfg
			err = ConfigToKeyring(storeConfig)
			if err != nil {
				return err
			}

			l.updateActiveConfig(storeConfig)
			// Save config to renew TokenExpireAt
			if err := l.SaveConfigFile(); err != nil {
				return err
			}
		}
	}

	// Store the retrieved parameters in a public static singleton
	DefaultConfig = cecCfg
	return nil
}

var refreshMux = &sync.Mutex{}

func RefreshAndStoreIfRequired(c *CecConfig) bool {
	refreshMux.Lock()
	defer refreshMux.Unlock()

	refreshed, err := RefreshIfRequired(&c.SdkConfig)
	if err != nil {
		log.Fatal("Could not refresh authentication token:", err)
	}
	if refreshed {
		// Copy config as IdToken will be cleared
		storeConfig := *c
		ConfigToKeyring(&storeConfig)
		// Save config to renew TokenExpireAt
		confData, _ := json.MarshalIndent(&storeConfig, "", "\t")
		ioutil.WriteFile(GetConfigFilePath(), confData, 0600)
	}

	return refreshed
}

func getSdkConfigFromEnv() (cells_sdk.SdkConfig, error) {

	var c cells_sdk.SdkConfig

	// Check presence of environment variables
	url := os.Getenv(KeyURL)
	clientKey := os.Getenv(KeyClientKey)
	clientSecret := os.Getenv(KeyClientSecret)
	user := os.Getenv(KeyUser)
	password := os.Getenv(KeyPassword)
	skipVerifyStr := os.Getenv(KeySkipVerify)
	if skipVerifyStr == "" {
		skipVerifyStr = "false"
	}
	skipVerify, err := strconv.ParseBool(skipVerifyStr)
	if err != nil {
		return c, err
	}

	// Client Key and Client Secret are not used anymore
	// if !(len(url) > 0 && len(clientKey) > 0 && len(clientSecret) > 0 && len(user) > 0 && len(password) > 0) {
	if !(len(url) > 0 && len(user) > 0 && len(password) > 0) {
		return c, nil
	}

	c.Url = url
	c.ClientKey = clientKey
	c.ClientSecret = clientSecret
	c.User = user
	c.Password = password
	c.SkipVerify = skipVerify

	// Note: this cannot be set via env variable. Enhance?
	c.UseTokenCache = true

	return c, nil
}

func getS3ConfigFromSdkConfig(sConf cells_sdk.SdkConfig) cells_sdk.S3Config {
	var c cells_sdk.S3Config
	c.Bucket = "io"
	c.ApiKey = "gateway"
	c.ApiSecret = "gatewaysecret"
	c.UsePydioSpecificHeader = false
	c.IsDebug = false
	c.Region = "us-east-1"
	c.Endpoint = sConf.Url
	return c
}

// GetConfigList retrieves the current configurations stored in the config.json file
func GetConfigList() (*ConfigList, error) {
	// assuming they are located in the default folder
	data, err := ioutil.ReadFile(GetConfigFilePath())
	if err != nil {
		return nil, err
	}

	cfg := &ConfigList{}
	err = json.Unmarshal(data, cfg)
	if err == nil {
		return cfg, nil
	}

	var oldConf *cells_sdk.SdkConfig
	if err = json.Unmarshal(data, &oldConf); err != nil {
		return nil, fmt.Errorf("unknown config format: %s", err)
	}
	// cfg = new(ConfigList)
	defaultLabel := "default"
	cfg.ActiveConfig = defaultLabel
	cfg = &ConfigList{
		Configs: map[string]*CecConfig{"default": {
			SdkConfig: *oldConf,
		}},
		ActiveConfig: defaultLabel,
	}

	return cfg, nil
}

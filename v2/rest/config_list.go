package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	cells_sdk "github.com/pydio/cells-sdk-go"
)

type ConfigList struct {
	Configs      map[string]*CecConfig
	ActiveConfig string
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

// Add appends the new config to the list and set it as default
func (list *ConfigList) Add(label string, config *CecConfig) error {
	_, ok := list.Configs[label]
	if ok {
		return fmt.Errorf("this label is already used: [%s]", label)
	}
	list.ActiveConfig = label
	list.Configs[label] = config
	return nil
}

// Remove removes a config from the list of available configurations by its label
func (list *ConfigList) Remove(label string) error {
	if _, ok := list.Configs[label]; !ok {
		return fmt.Errorf("config not found, this label is not valid [%s]", label)
	}
	delete(list.Configs, label)
	return nil
}

func (list *ConfigList) SetActiveConfig(label string) error {
	if _, ok := list.Configs[label]; !ok {
		return fmt.Errorf("this label does not exist %s", label)
	}
	list.ActiveConfig = label
	return nil
}

func (list *ConfigList) GetActiveConfig() *CecConfig {
	return list.Configs[list.ActiveConfig]
}

func (list *ConfigList) updateActiveConfig(cf *CecConfig) {
	list.Configs[list.ActiveConfig] = cf
}

// SaveConfigFile saves inside the config file
func (list *ConfigList) SaveConfigFile() error {
	confData, _ := json.MarshalIndent(&list, "", "\t")
	if err := ioutil.WriteFile(GetConfigFilePath(), confData, 0666); err != nil {
		return fmt.Errorf("could not save the config file, cause: %s", err)
	}
	return nil
}

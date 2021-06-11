package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/oss/utils"
)

const ServiceName = "oss"

type Configs struct {
	path               string
	InstanceAddress    string `json:"instance_address"`
	MqttServerURI      string `json:"mqtt_server_uri"`
	MqttUsername       string `json:"mqtt_server_username"`
	MqttPassword       string `json:"mqtt_server_password"`
	MqttClientIdPrefix string `json:"mqtt_client_id_prefix"`
	LogFile            string `json:"log_file"`
	LogLevel           string `json:"log_level"`
	LogFormat          string `json:"log_format"`
	WorkDir            string `json:"-"`
	ConfiguredAt       string `json:"configured_at"`
	ConfiguredBy       string `json:"configured_by"`
	Email              string `json:"email"`
	EmailCode          string `json:"emailCode"`
	UserID             string `json:"userId"`
	AccessToken        string `json:"accessToken"`
	PollTimeSec        string `json:"poll_time_sec"`
	SelectedMeters     string `json:"selected_meters"`
}

func NewConfigs(workDir string) *Configs {
	conf := &Configs{WorkDir: workDir}
	conf.path = filepath.Join(workDir, "data", "config.json")
	if !utils.FileExists(conf.path) {
		log.Info("Config file doesn't exist.Loading default config")
		defaultConfigFile := filepath.Join(workDir, "defaults", "config.json")
		err := utils.CopyFile(defaultConfigFile, conf.path)
		if err != nil {
			fmt.Print(err)
			panic("Can't copy config file.")
		}
	}
	return conf
}

func (cf *Configs) LoadFromFile() error {
	configFileBody, err := ioutil.ReadFile(cf.path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configFileBody, cf)
	if err != nil {
		return err
	}
	return nil
}

func (cf *Configs) SaveToFile() error {
	cf.ConfiguredBy = "auto"
	cf.ConfiguredAt = time.Now().Format(time.RFC3339)
	bpayload, err := json.Marshal(cf)
	err = ioutil.WriteFile(cf.path, bpayload, 0664)
	if err != nil {
		return err
	}
	return err
}

func (cf *Configs) GetDataDir() string {
	return filepath.Join(cf.WorkDir, "data")
}

func (cf *Configs) GetDefaultDir() string {
	return filepath.Join(cf.WorkDir, "defaults")
}

func (cf *Configs) LoadDefaults() error {
	configFile := filepath.Join(cf.WorkDir, "data", "config.json")
	os.Remove(configFile)
	log.Info("Config file doesn't exist.Loading default config")
	defaultConfigFile := filepath.Join(cf.WorkDir, "defaults", "config.json")
	return utils.CopyFile(defaultConfigFile, configFile)
}

func (cf *Configs) IsConfigured() bool {
	if cf.SelectedMeters != "" {
		return true
	}
	return false
}

type ConfigReport struct {
	OpStatus  string    `json:"op_status"`
	ErrorText string    `json:"error_text"`
	AppState  AppStates `json:"app_state"`
}

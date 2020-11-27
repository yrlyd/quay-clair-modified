// Copyright 2018 clair authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/quay/clair/v3"
	"github.com/quay/clair/v3/api"
	"github.com/quay/clair/v3/database"
	"github.com/quay/clair/v3/ext/notification"
	"github.com/quay/clair/v3/ext/vulnsrc"
	"github.com/quay/clair/v3/pkg/pagination"
)

// ErrDatasourceNotLoaded is returned when the datasource variable in the
// configuration file is not loaded properly
var ErrDatasourceNotLoaded = errors.New("could not load configuration: no database source specified")

// File represents a YAML configuration file that namespaces all Clair
// configuration under the top-level "clair" key.
type File struct {
	Clair Config `yaml:"clair"`
}

// Config is the global configuration for an instance of Clair.
type Config struct {
	Database database.RegistrableComponentConfig
	Updater  *clair.UpdaterConfig
	Notifier *notification.Config
	API      *api.Config
}

// DefaultConfig is a configuration that can be used as a fallback value.
func DefaultConfig() Config {
	return Config{
		Database: database.RegistrableComponentConfig{
			Type: "pgsql",
		},
		Updater: &clair.UpdaterConfig{
			EnabledUpdaters: vulnsrc.ListUpdaters(),
			Interval:        1 * time.Hour,
		},
		API: &api.Config{
			HealthAddr: "0.0.0.0:6061",
			Addr:       "0.0.0.0:6060",
			Timeout:    900 * time.Second,
		},
		Notifier: &notification.Config{
			Attempts:         5,
			RenotifyInterval: 2 * time.Hour,
		},
	}
}

// LoadConfig is a shortcut to open a file, read it, and generate a Config.
//
// It supports relative and absolute paths. Given "", it returns DefaultConfig.
func LoadConfig(path string) (config *Config, err error) {
	var cfgFile File
	cfgFile.Clair = DefaultConfig()
	if path == "" {
		return &cfgFile.Clair, nil
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return
	}
	defer f.Close()

	d, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(d, &cfgFile)
	if err != nil {
		return
	}
	config = &cfgFile.Clair

	// Generate a pagination key if none is provided.
	if v, ok := config.Database.Options["paginationkey"]; !ok || v == nil || v.(string) == "" {
		log.Warn("pagination key is empty, generating...")
		config.Database.Options["paginationkey"] = pagination.Must(pagination.NewKey()).String()
	} else {
		_, err = pagination.KeyFromString(config.Database.Options["paginationkey"].(string))
		if err != nil {
			return
		}
	}

	return
}

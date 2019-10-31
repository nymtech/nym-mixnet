// Copyright 2019 The Loopix-Messaging Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"text/template"

	"github.com/BurntSushi/toml"
)

var configTemplate *template.Template

func init() {
	var err error
	if configTemplate, err = template.New("configFileTemplate").Funcs(template.FuncMap{
		"FormatFloats": func(f float64) string { return fmt.Sprintf("%.2f", f) },
	}).Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

// LoadBinary loads, parses and validates the provided buffer b (as a config)
// and returns the Config.
func LoadBinary(b []byte) (*Config, error) {
	cfg := new(Config)
	_, err := toml.Decode(string(b), cfg)
	if err != nil {
		return nil, err
	}
	if err := cfg.validateAndApplyDefaults(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFile loads, parses and validates the provided file and returns the Config.
func LoadFile(f string) (*Config, error) {
	b, err := ioutil.ReadFile(filepath.Clean(f))
	if err != nil {
		return nil, err
	}
	return LoadBinary(b)
}

// WriteConfigFile renders config using the template and writes it to specified file path.
func WriteConfigFile(path string, config *Config) error {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		return err
	}

	return ioutil.WriteFile(path, buffer.Bytes(), 0644)
}

// helper function to make config creation independent of root dir
// adapted from the tendermint code
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

// While using normal toml marshalling would have been way simpler, I think it's useful to have comments attached to
// the saved file.
// Note: any changes to the template must be reflected in the appropriate structs and tags.
const defaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

##### main base client config options #####
[client]

# Human readable ID of this particular client.
id = "{{ .Client.ID }}"

# URL to the topology endpoint of the directory server.
directory_server_topology = "{{ .Client.DirectoryServerTopologyEndpoint }}"

# Path to file containing private key.
priv_key_file = "{{ .Client.PrivateKey }}"

# Path to file containing public key.
pub_key_file = "{{ .Client.PublicKey }}"

##### additional client config options #####

# ID of the provider to which the client should send messages.
provider_id = "{{ .Client.ProviderID }}"

# directory for mixapps, such as a chat client, to store their app-specific data.
mixapps_directory = "{{ .Client.MixAppsDirectory }}"

##### advanced configuration options #####

# Absolute path to the home loopix Clients directory.
loopix_home_directory = "{{ .Client.HomeDirectory }}"

##### logging configuration options #####
[logging]

# Whether to disable disables logging entirely.
disable = {{ .Logging.Disable }}

# The log file. If omitted or set to empty value, stdout will be used.
file = "{{ .Logging.File }}"

# The logging level of the client. The available options include:
# trace, debug, info, warning, error, panic, fatal
# Warning: The 'trace' and 'debug' log levels are unsafe for production use.
level = "{{ .Logging.Level }}"

##### debug configuration options #####
[debug]

# The rate at which clients are sending loop packets in the loop cover traffic stream.
# The value is the parameter of an exponential distribution, and is the reciprocal of the
# expected value of the exponential distribution.
# If set to a negative value, the loop cover traffic stream will be disabled.
loop_cover_traffic_rate = {{FormatFloats .Debug.LoopCoverTrafficRate }}

# The rate at which clients are querying the providers for received packets.
# The value is the parameter of an exponential distribution, and is the reciprocal of the
# expected value of the exponential distribution.
# If set to a negative value, client will never try to fetch its messages.
fetch_message_rate = {{FormatFloats .Debug.FetchMessageRate }}

# The rate at which clients are sending their real traffic to providers.
# If no real packets are available and cover traffic is enabled,
# a drop cover message is sent instead in order to preserve the rate.
# The value is the parameter of an exponential distribution, and is the reciprocal of the
# expected value of the exponential distribution.
# If set to a negative value, client will never try to send real traffic data.
message_sending_rate = {{FormatFloats .Debug.MessageSendingRate }}

# Whether loop cover messages should be sent to respect message_sending_rate.
# In the case of it being disabled and not having enough real traffic
# waiting to be sent the actual sending rate is going be lower than the desired value
# thus decreasing the anonymity.
rate_compliant_cover_messages_disabled = {{ .Debug.RateCompliantCoverMessagesDisabled }}


`

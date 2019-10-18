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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	mainConfig "github.com/nymtech/nym-mixnet/config"
	"github.com/sirupsen/logrus"
)

const (
	defaultLoopixDirectory        = ".loopix"
	defaultLoopixClientsDirectory = "clients"
	defaultClientMixAppDirectory  = "mixapps"
	defaultConfigDirectory        = "config"
	defaultConfigFileName         = "config.toml"

	defaultLogLevel = "info"

	defaultPrivateKeyFileName = "private_key.pem"
	defaultPublicKeyFileName  = "public_key.pem"

	defaultLoopCoverTrafficRate = 10.0
	defaultFetchMessageRate     = 10.0
	defaultMessageSendingRate   = 10.0

	defaultDirectoryServerTopologyEndpoint      = mainConfig.DirectoryServerTopology
	DefaultLocalDirectoryServerTopologyEndpoint = mainConfig.LocalDirectoryServerTopology
)

//nolint: gochecknoglobals
var (
	// TODO: if/when we decide to create configs for other loopix entities (i.e. providers, mixnodes)
	// there should be a 'master' home of $HOME/.loopix from which we would have the subdirectories of
	// Mixnodes, Providers, Clients, etc.
	defaultHomeDirectory  = os.ExpandEnv(filepath.Join("$HOME", defaultLoopixDirectory, defaultLoopixClientsDirectory))
	defaultPrivateKeyPath = filepath.Join(defaultConfigDirectory, defaultPrivateKeyFileName)
	defaultPublicKeyPath  = filepath.Join(defaultConfigDirectory, defaultPublicKeyFileName)
)

// DefaultConfigPath returns absolute path to the default configuration file of the particular client.
// The returned path should be $HOME/.loopix/Clients/clientID/config/config.toml
func DefaultConfigPath(clientID string) (string, error) {
	if len(clientID) == 0 {
		return "", errors.New("invalid clientID provided")
	}
	return filepath.Join(
		defaultHomeDirectory,
		clientID,
		defaultConfigDirectory,
		defaultConfigFileName,
	), nil
}

// Client is the Loopix Client configuration.
type Client struct {
	// HomeDirectory specifies absolute path to the home loopix Clients directory.
	// It is expected to use default value and hence .toml file should not redefine this field.
	HomeDirectory string `toml:"loopix_home_directory"`

	// ID specifies the human readable ID of this particular client.
	// If not provided a random id will be generated instead.
	ID string `toml:"id"`

	// DirectoryServerTopologyEndpoint specifies URL to the topology endpoint of the directory server.
	DirectoryServerTopologyEndpoint string `toml:"directory_server_topology"`

	// MixAppsDirectory specifies directory for mixapps, such as a chat client,
	// to store their app-specific data.
	MixAppsDirectory string `toml:"mixapps_directory"`

	// PrivateKey specifies path to file containing private key.
	PrivateKey string `toml:"priv_key_file"`

	// PublicKey specifies path to file containing public key.
	// TODO: we could actually get rid of public key file completely, as it can be inferred from the private key alone
	// But I guess having an explicit public key file could be convenient?
	// To say, for example, share it with somebody else.
	PublicKey string `toml:"pub_key_file"`

	// ProviderID specifies ID of the provider to which the client should send messages.
	// If initially omitted, a random provider will be chosen from the available topology.
	ProviderID string `toml:"provider_id"`
}

// DefaultClientConfig returns default Client config for provided clientID.
func DefaultClientConfig(clientID string) (*Client, error) {
	if len(clientID) == 0 {
		return nil, errors.New("invalid clientID provided")
	}
	// Even though defaults could be obtained by validating empty struct, lets be explicit about it.
	return &Client{
		HomeDirectory:                   defaultHomeDirectory,
		ID:                              clientID,
		MixAppsDirectory:                defaultClientMixAppDirectory,
		DirectoryServerTopologyEndpoint: defaultDirectoryServerTopologyEndpoint,
		PrivateKey:                      defaultPrivateKeyPath,
		PublicKey:                       defaultPublicKeyPath,
	}, nil
}

func (cfg *Client) Home() string {
	return filepath.Join(cfg.HomeDirectory, cfg.ID)
}

// PrivateKeyFile returns the full path to the public key file.
func (cfg *Client) PrivateKeyFile() string {
	return rootify(cfg.PrivateKey, cfg.Home())
}

// PublicKeyFile returns the full path to the private key file.
func (cfg *Client) PublicKeyFile() string {
	return rootify(cfg.PublicKey, cfg.Home())
}

func (cfg *Client) FullMixAppsDir() string {
	return rootify(cfg.MixAppsDirectory, cfg.Home())
}

func (cfg *Client) validateAndApplyDefaults() error {
	// if custom home directory is specified it must have an absolute path
	if len(cfg.HomeDirectory) > 0 {
		if !filepath.IsAbs(cfg.HomeDirectory) {
			return errors.New("config: specified home directory is not an absolute path")
		}
	} else {
		cfg.HomeDirectory = defaultHomeDirectory
	}

	// if custom mixapps directory is specified it must have an absolute path
	if len(cfg.MixAppsDirectory) == 0 {
		cfg.MixAppsDirectory = defaultClientMixAppDirectory
	}

	// it is also required to specify ID otherwise we could not distinguish between multiple instances
	if len(cfg.ID) == 0 {
		return errors.New("config: client ID was not specified")
	}

	// for the rest, if left unspecified, use defaults
	if len(cfg.DirectoryServerTopologyEndpoint) == 0 {
		cfg.DirectoryServerTopologyEndpoint = defaultDirectoryServerTopologyEndpoint
	}

	// we're not checking for existence of the key files as if they do not exist, they're going to be generated
	if len(cfg.PrivateKey) == 0 {
		cfg.PrivateKey = defaultPrivateKeyPath
	}

	if len(cfg.PublicKey) == 0 {
		cfg.PublicKey = defaultPublicKeyPath
	}

	return nil
}

// Logging is the Loopix Client logging configuration.
type Logging struct {
	// Disable disables logging entirely.
	Disable bool `toml:"disable"`

	// File specifies the log file, if omitted stdout will be used.
	File string `toml:"file"`

	// Level specifies the log level.
	Level string `toml:"level"`
}

func (cfg *Logging) validate() error {
	_, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("config: invalid logging level: %s (%v)", cfg.Level, err)
	}
	return nil
}

// DefaultLoggingConfig returns default logging configuration.
func DefaultLoggingConfig(clientID string) *Logging {
	return &Logging{
		Disable: false,
		File:    filepath.Join("/tmp", "loopix_"+clientID+".log"),
		Level:   defaultLogLevel,
	}
}

// Debug is the Loopix Client debug configuration.
type Debug struct {
	// LoopCoverTrafficRate defines the rate at which clients are sending loop packets in the loop cover traffic stream.
	// The value is the parameter of an exponential distribution, and is the reciprocal of the
	// expected value of the exponential distribution.
	// If set to a negative value, the loop cover traffic stream will be disabled.
	LoopCoverTrafficRate float64 `toml:"loop_cover_traffic_rate"`

	// FetchMessageRate defines the rate at which clients are querying the providers for received packets.
	// The value is the parameter of an exponential distribution, and is the reciprocal of the
	// expected value of the exponential distribution.
	// If set to a negative value, client will never try to fetch its messages.
	FetchMessageRate float64 `toml:"fetch_message_rate"`

	// MessageSendingRate defines the rate at which clients are sending their real traffic to providers.
	// If no real packets are available and cover traffic is enabled,
	// a drop cover message is sent instead in order to preserve the rate.
	// The value is the parameter of an exponential distribution, and is the reciprocal of the
	// expected value of the exponential distribution.
	// If set to a negative value, client will never try to send real traffic data.
	MessageSendingRate float64 `toml:"message_sending_rate "`

	// RateCompliantCoverMessagesDisabled specifies whether loop cover messages should be sent
	// to respect MessageSendingRate. In the case of it being disabled and not having enough real traffic
	// waiting to be sent the actual sending rate is going be lower than the desired value
	// thus decreasing the anonymity.
	RateCompliantCoverMessagesDisabled bool `toml:"rate_compliant_cover_messages_disabled"`
}

func (dCfg *Debug) applyDefaults() {
	if dCfg.LoopCoverTrafficRate == 0.0 {
		dCfg.LoopCoverTrafficRate = defaultLoopCoverTrafficRate
	}
	if dCfg.FetchMessageRate == 0.0 {
		dCfg.FetchMessageRate = defaultFetchMessageRate
	}
	if dCfg.MessageSendingRate == 0.0 {
		dCfg.MessageSendingRate = defaultMessageSendingRate
	}
}

// DefaultDebugConfig returns default debug configuration.
func DefaultDebugConfig() *Debug {
	return &Debug{
		LoopCoverTrafficRate:               defaultLoopCoverTrafficRate,
		FetchMessageRate:                   defaultFetchMessageRate,
		MessageSendingRate:                 defaultMessageSendingRate,
		RateCompliantCoverMessagesDisabled: false,
	}
}

// Config is the top level Loopix Client configuration.
type Config struct {
	Client  *Client  `toml:"client"`
	Logging *Logging `toml:"logging"`
	Debug   *Debug   `toml:"debug"`
}

// DefaultConfig returns full default config for given clientID
func DefaultConfig(clientID string) (*Config, error) {
	if len(clientID) == 0 {
		return nil, errors.New("invalid clientID provided")
	}
	defaultClientConfig, _ := DefaultClientConfig(clientID)
	return &Config{
		Client:  defaultClientConfig,
		Logging: DefaultLoggingConfig(clientID),
		Debug:   DefaultDebugConfig(),
	}, nil
}

func (cfg *Config) validateAndApplyDefaults() error {
	if cfg.Client == nil {
		return errors.New("config: No Client block was present")
	}

	if err := cfg.Client.validateAndApplyDefaults(); err != nil {
		return err
	}

	if cfg.Debug == nil {
		cfg.Debug = &Debug{}
	}
	cfg.Debug.applyDefaults()

	if cfg.Logging == nil {
		cfg.Logging = DefaultLoggingConfig(cfg.Client.ID)
	}

	if err := cfg.Logging.validate(); err != nil {
		return err
	}

	return nil
}

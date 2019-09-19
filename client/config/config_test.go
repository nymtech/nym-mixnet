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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigPathNoID(t *testing.T) {
	configPath, err := DefaultConfigPath("")
	assert.Len(t, configPath, 0)
	assert.Error(t, err)
}

func TestDefaultConfigPath(t *testing.T) {

	// I think hardcoding this value is actually a good way to ensure we don't unexpectedly change something
	configPath, err := DefaultConfigPath("foo")
	homeDir := os.ExpandEnv("$HOME")
	assert.Equal(t, filepath.Join(homeDir, "/.loopix/clients/foo/config/config.toml"), configPath)
	assert.Nil(t, err)

}

func TestDefaultConfigNoID(t *testing.T) {
	fullCfg, err := DefaultConfig("")
	assert.Nil(t, fullCfg)
	assert.Error(t, err)

	clientCfg, err := DefaultClientConfig("")
	assert.Nil(t, clientCfg)
	assert.Error(t, err)
}

func TestDefaultConfig(t *testing.T) {
	someID := "foo"

	fullCfg, err := DefaultConfig(someID)
	assert.NotNil(t, fullCfg)
	assert.Nil(t, err)

	// check that replacing homedir correctly affects locations of keys
	// note that this should have never been changed anyway...
	fullCfg.Client.HomeDirectory = "/baz"

	assert.Equal(t, "/baz/foo/config/private_key.pem", fullCfg.Client.PrivateKeyFile())
	assert.Equal(t, "/baz/foo/config/public_key.pem", fullCfg.Client.PublicKeyFile())

	// However, if keys have absolute paths, homedir should be ignored
	fullCfg.Client.PrivateKey = "/some/absolute/path/priv.pem"
	fullCfg.Client.PublicKey = "/some/absolute/path/pub.pem"

	assert.Equal(t, "/some/absolute/path/priv.pem", fullCfg.Client.PrivateKeyFile())
	assert.Equal(t, "/some/absolute/path/pub.pem", fullCfg.Client.PublicKeyFile())
}

func TestValidateAndApplyDefaults(t *testing.T) {
	// if we create empty structs and apply defaults to them, we should obtain results identical
	// to just obtaining default structs
	someID := "foo"

	fullCfg, err := DefaultConfig(someID)
	assert.NotNil(t, fullCfg)
	assert.Nil(t, err)

	freshFullCfg := &Config{
		Client: &Client{},
	}
	// We need to explicitly set the ID field
	freshFullCfg.Client.ID = someID

	assert.Nil(t, freshFullCfg.validateAndApplyDefaults())
	assert.Equal(t, fullCfg, freshFullCfg)

	clientCfg, err := DefaultClientConfig(someID)
	assert.NotNil(t, clientCfg)
	assert.Nil(t, err)

	freshClientCfg := new(Client)
	freshClientCfg.ID = someID

	assert.Nil(t, freshClientCfg.validateAndApplyDefaults())
	assert.Equal(t, clientCfg, freshClientCfg)

	debugCfg := DefaultDebugConfig()
	assert.NotNil(t, debugCfg)

	freshDebugCfg := new(Debug)
	freshDebugCfg.applyDefaults()

	assert.Equal(t, debugCfg, freshDebugCfg)

	// No client block
	newCfg := &Config{}
	assert.Error(t, newCfg.validateAndApplyDefaults())
}

func TestValidateClientBlock(t *testing.T) {
	// Setting custom home directory that is not absolute
	someID := "foo"
	fullCfg, err := DefaultConfig(someID)
	assert.NotNil(t, fullCfg)
	assert.Nil(t, err)

	fullCfg.Client.HomeDirectory = "non/absolute/path"
	assert.Error(t, fullCfg.validateAndApplyDefaults())

	fullCfg, err = DefaultConfig(someID)
	assert.NotNil(t, fullCfg)
	assert.Nil(t, err)

	fullCfg.Client.ID = ""
	assert.Error(t, fullCfg.validateAndApplyDefaults())
}

func TestValidateLogging(t *testing.T) {
	validLevels := []string{
		"trace",
		"debug",
		"info",
		"warn",
		"warning",
		"error",
		"fatal",
		"TRACE",
		"DEBUG",
		"INFO",
		"WARN",
		"WARNING",
		"ERROR",
		"FATAL",
		"TrAcE",
		"dEbUg",
		"InFo",
		"WaRn",
		"WaRnInG",
		"eRrOr",
		"FaTaL",
	}
	someID := "foo"
	for _, validLevel := range validLevels {
		fullCfg, err := DefaultConfig(someID)
		assert.NotNil(t, fullCfg)
		assert.Nil(t, err)

		fullCfg.Logging.Level = validLevel
		assert.Nil(t, fullCfg.validateAndApplyDefaults())
	}

	invalidLevels := []string{
		"foo",
		"/info",
		"trac",
		"dbg",
	}

	for _, invalidLevel := range invalidLevels {
		fullCfg, err := DefaultConfig(someID)
		assert.NotNil(t, fullCfg)
		assert.Nil(t, err)

		fullCfg.Logging.Level = invalidLevel
		assert.Error(t, fullCfg.validateAndApplyDefaults())
	}
}

func TestLoadBinary(t *testing.T) {
	cfg, err := LoadBinary([]byte(""))
	assert.Nil(t, cfg)
	assert.Error(t, err)

	cfg2, err := LoadBinary([]byte("[someinvalid[toml{data]"))
	assert.Nil(t, cfg2)
	assert.Error(t, err)

	someID := "foo"
	fullCfg, err := DefaultConfig(someID)
	assert.NotNil(t, fullCfg)
	assert.Nil(t, err)

	b, err := toml.Marshal(fullCfg)
	assert.Nil(t, err)

	cfg3, err := LoadBinary(b)
	assert.NotNil(t, cfg3)
	assert.Nil(t, err)

	assert.Equal(t, fullCfg, cfg3)
}

func TestLoadFile(t *testing.T) {
	cfg, err := LoadFile("/path/that/does/not/exist")
	assert.Nil(t, cfg)
	assert.Error(t, err)

	someID := "foo"
	fullCfg, err := DefaultConfig(someID)
	assert.NotNil(t, fullCfg)
	assert.Nil(t, err)

	b, err := toml.Marshal(fullCfg)
	assert.Nil(t, err)

	outFile := "testCfg.toml"

	tmpDir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)

	outFilePath := filepath.Join(tmpDir, outFile)
	ioutil.WriteFile(outFilePath, b, 0600)

	loadedCfg, err := LoadFile(outFilePath)
	assert.Nil(t, err)
	assert.Equal(t, fullCfg, loadedCfg)
}

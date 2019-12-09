// // Copyright 2019 The Nym Mixnet Authors
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //      http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

package logger

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	validLevel := "INFO"

	tmpfile, err := ioutil.TempFile("", "tmplog")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	tests := []struct {
		f       string
		disable bool

		isValid bool
	}{
		{f: "someinvalidnotexistingpath/", disable: false, isValid: false},

		{f: "", disable: true, isValid: true},
		// if it disabled, we shouldn't need to care about the rest
		{f: "someinvalidnotexistingpath/", disable: true, isValid: true},
		{f: tmpfile.Name(), disable: false, isValid: true},
		{f: tmpfile.Name(), disable: true, isValid: true},
	}

	for _, test := range tests {
		if err := os.Truncate(tmpfile.Name(), 0); err != nil {
			log.Fatal(err)
		}

		logger, err := New(test.f, validLevel, test.disable)
		if test.isValid {
			assert.NotNil(t, logger)
			assert.Nil(t, err)
		} else {
			assert.Nil(t, logger)
			assert.Error(t, err)
		}

		if test.f == tmpfile.Name() && !test.disable {
			logger.GetLogger("test").Info("Test log")

			file, err := os.Open(tmpfile.Name())
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			scanner.Scan()
			written := scanner.Text()
			assert.True(t, strings.HasSuffix(written, "Test log"))
		}

	}
}

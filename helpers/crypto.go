// Copyright 2018-2019 The Loopix-Messaging Authors
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

/*
	Package helpers implements all useful functions which are used in the code of anonymous messaging system.
*/

package helpers

import (
	"crypto/sha256"
	"errors"
	"math/rand"
	"time"

	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/sphinx"
)

var (
	ErrPermEmptyList                = errors.New("cannot permute an empty list of mixes")
	ErrTooBigSampleSize             = errors.New("cannot take a sample larger than the given list")
	ErrExponentialDistributionParam = errors.New("the parameter of exponential distribution has to be larger than zero")
)

func init() {
	// TODO: replace math/rand with crypto/rand to get rid of needing to seed it?
	// + it will be more 'secure'
	// However, we would need to implement 'Perm' ourselves
	rand.Seed(time.Now().UTC().UnixNano())
}

// RandomMix returns a single pseudorandomly chosen mix from given slices of mixes.
func RandomMix(mixes []config.MixConfig) config.MixConfig {
	return mixes[rand.Intn(len(mixes))]
}

// a very dummy implementation of getting "random" string of given length
// could be improved in number of ways but for the test sake it's good enough
func RandomString(length int) string {
	letterRunes := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func RandomExponential(expParam float64) (float64, error) {
	if expParam <= 0.0 {
		return 0.0, ErrExponentialDistributionParam
	}
	return rand.ExpFloat64() / expParam, nil
}

// SHA256 computes the hash value of a given argument using SHA256 algorithm.
func SHA256(arg []byte) ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write(arg); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func IsZeroElement(el sphinx.CryptoElement) bool {
	bytes := el.Bytes()
	for _, b := range bytes {
		if b != 0 {
			return false
		}
	}
	return true
}

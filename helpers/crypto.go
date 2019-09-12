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
)


func init() {
	// TODO: replace math/rand with crypto/rand to get rid of needing to seed it?
	// + it will be more 'secure'
	// However, we would need to implement 'Perm' ourselves
	rand.Seed(time.Now().UTC().UnixNano())
}

func Permute(slice []config.MixConfig) ([]config.MixConfig, error) {
	if len(slice) == 0 {
		return nil, ErrPermEmptyList
	}

	permutedData := make([]config.MixConfig, len(slice))
	permutation := rand.Perm(len(slice))
	for i, v := range permutation {
		permutedData[v] = slice[i]
	}
	return permutedData, nil
}

// RandomSample takes a slice of MixConfigs, and returns a new
// slice of length `length` in a randomised order.
func RandomSample(slice []config.MixConfig, length int) ([]config.MixConfig, error) {
	if len(slice) < length {
		return nil, ErrTooBigSampleSize
	}

	permuted, err := Permute(slice)
	if err != nil {
		return nil, err
	}

	return permuted[:length], err
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

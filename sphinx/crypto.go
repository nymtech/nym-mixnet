// Copyright 2018 The Loopix-Messaging Authors
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

package sphinx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
)

// AesCtr returns AES XOR ciphertext in counter mode for the given key and plaintext
func AesCtr(key, plaintext []byte) ([]byte, error) {

	ciphertext := make([]byte, len(plaintext))

	iv := []byte("0000000000000000")
	//if _, err := io.ReadFull(crand.Reader, iv); err != nil {
	//	panic(err)
	//}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)

	return ciphertext, nil
}

func hash(arg []byte) ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write(arg); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// Hmac computes a hash-based message authentication code for a given key and message.
// Returns a byte array containing the MAC checksum.
func Hmac(key, message []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(message); err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil
}

// KDF returns the hash of K for a given key
func KDF(key []byte) ([]byte, error) {
	b, err := hash(key)
	if err != nil {
		return nil, err
	}
	return b[:K], nil
}

func computeMac(key, data []byte) ([]byte, error) {
	return Hmac(key, data)
}

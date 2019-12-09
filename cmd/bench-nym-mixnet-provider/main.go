// Copyright 2019 The Nym Mixnet Authors
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

package main

import "github.com/tav/golly/optparse"

func main() {
	var logo = `
                        
  _ __  _   _ _ __ ___  
 | '_ \| | | | '_ \ _ \
 | | | | |_| | | | | | |
 |_| |_|\__, |_| |_| |_|
        |___/  
          
(mixnet-OUTDATED-benchmark-provider)
`
	cmds := map[string]func([]string, string){
		"run": cmdRun,
	}
	info := map[string]string{
		"run": "Run a benchmark Nym mixnet provider",
	}
	optparse.Commands("bench-nym-mixnet-provider", "0.4.0", cmds, info, logo)
}

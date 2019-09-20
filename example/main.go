// Copyright 2019-present Facebook Inc. All Rights Reserved.
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
//
// Run this exporting FLOG_VERBOSITY
package main

import (
	"os"

	"github.com/facebookincubator/flog"
)

func main() {
	// Normal use
	flog.Infof("Current uid is: %d", os.Geteuid())
	flog.Warningf("Current gid is: %d", os.Getgid())

	// Assume values from env
	flog.V(0).Info("You should be able to see this line without env vars")
	flog.V(1).Info("Set FLOG_VERBOSITY > 0")
	flog.V(2).Info("Set FLOG_VERBOSITY > 1")

	// Setup a config
	logCfg := &flog.Config{Verbosity: "3"}
	logCfg.Set()
	flog.V(2).Info("Verbosity 3 under config")
}

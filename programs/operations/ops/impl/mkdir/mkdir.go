/*
 Copyright 2016 Padduck, LLC

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 	http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package mkdir

import (
	"github.com/pufferpanel/pufferd/programs/operations/ops"
	"os"

	"github.com/pufferpanel/apufferi/common"
	"github.com/pufferpanel/apufferi/logging"
	"github.com/pufferpanel/pufferd/environments"
)

type Mkdir struct {
	TargetFile  string
}

func (m *Mkdir) Run(env environments.Environment) error {
	logging.Debugf("Making directory: %s\n", m.TargetFile)
	env.DisplayToConsole("Creating directory: %s\n", m.TargetFile)
	target := common.JoinPath(env.GetRootDirectory(), m.TargetFile)
	return os.MkdirAll(target, 0755)
}

type MkdirOperationFactory struct {
}

func (of MkdirOperationFactory) Create(op ops.CreateOperation) ops.Operation {
	target := op.OperationArgs["target"].(string)
	return &Mkdir{TargetFile: target}
}

func (of MkdirOperationFactory) Key() string {
	return "mkdir"
}


var Factory MkdirOperationFactory
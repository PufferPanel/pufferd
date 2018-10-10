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

package command

import (
	"github.com/pufferpanel/pufferd/programs/operations/ops"
	"strings"

	"fmt"
	"github.com/pufferpanel/apufferi/common"
	"github.com/pufferpanel/apufferi/logging"
	"github.com/pufferpanel/pufferd/environments"
)

type Command struct {
	Commands []string
	Env      map[string]string
}

func (c Command) Run(env environments.Environment) error {
	for _, cmd := range c.Commands {
		logging.Debugf("Executing command: %s", cmd)
		env.DisplayToConsole(fmt.Sprintf("Executing: %s\n", cmd))
		parts := strings.Split(cmd, " ")
		cmd := parts[0]
		args := parts[1:]
		_, err := env.Execute(cmd, args, c.Env, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

type CommandOperationFactory struct {
}

func (of CommandOperationFactory) Create(op ops.CreateOperation) ops.Operation {
	commands := common.ToStringArray(op.OperationArgs["commands"])
	env := common.ReplaceTokensInMap(op.EnvironmentVariables, op.DataMap)
	return Command{Commands: commands, Env: env}
}

func (of CommandOperationFactory) Key() string {
	return "command"
}


var Factory CommandOperationFactory
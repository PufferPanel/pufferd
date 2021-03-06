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

package operations

import (
	"github.com/pufferpanel/apufferi/v4"
	"github.com/pufferpanel/pufferd/v2"
	"github.com/pufferpanel/pufferd/v2/environments/envs"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/command"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/download"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/forgedl"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/mkdir"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/mojangdl"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/move"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/spongeforgedl"
	"github.com/pufferpanel/pufferd/v2/programs/operations/impl/writefile"
	"github.com/pufferpanel/pufferd/v2/programs/operations/ops"
	"github.com/spf13/cast"
)

var commandMapping map[string]ops.OperationFactory

func LoadOperations() {
	commandMapping = make(map[string]ops.OperationFactory)

	loadCoreModules()

	loadOpModules()
}

func GenerateProcess(directions []interface{}, environment envs.Environment, dataMapping map[string]interface{}, env map[string]string) (OperationProcess, error) {
	dataMap := make(map[string]interface{})
	for k, v := range dataMapping {
		dataMap[k] = v
	}

	dataMap["rootDir"] = environment.GetRootDirectory()
	operationList := make([]ops.Operation, 0)
	for _, mapping := range directions {

		var typeMap apufferi.MetadataType
		err := apufferi.UnmarshalTo(mapping, &typeMap)
		if err != nil {
			return OperationProcess{}, err
		}

		factory := commandMapping[typeMap.Type]
		if factory == nil {
			return OperationProcess{}, pufferd.ErrMissingFactory
		}

		mapCopy := make(map[string]interface{}, 0)

		//replace tokens
		for k, v := range typeMap.Metadata {
			switch r := v.(type) {
			case string:
				{
					mapCopy[k] = apufferi.ReplaceTokens(r, dataMap)
				}
			case []string:
				{
					mapCopy[k] = apufferi.ReplaceTokensInArr(r, dataMap)
				}
			case map[string]string:
				{
					mapCopy[k] = apufferi.ReplaceTokensInMap(r, dataMap)
				}
			case []interface{}:
				{
					//if we can convert this to a string list, we can work with it
					temp := cast.ToStringSlice(r)
					if len(temp) == len(r) {
						mapCopy[k] = apufferi.ReplaceTokensInArr(temp, dataMap)
					} else {
						mapCopy[k] = v
					}
				}
			default:
				mapCopy[k] = v
			}
		}

		envMap := apufferi.ReplaceTokensInMap(env, dataMap)

		opCreate := ops.CreateOperation{
			OperationArgs:        mapCopy,
			EnvironmentVariables: envMap,
			DataMap:              dataMap,
		}

		op := factory.Create(opCreate)

		operationList = append(operationList, op)
	}
	return OperationProcess{processInstructions: operationList}, nil
}

type OperationProcess struct {
	processInstructions []ops.Operation
}

func (p *OperationProcess) Run(env envs.Environment) (err error) {
	for p.HasNext() {
		err = p.RunNext(env)
		if err != nil {
			break
		}
	}
	return
}

func (p *OperationProcess) RunNext(env envs.Environment) error {
	var op ops.Operation
	op, p.processInstructions = p.processInstructions[0], p.processInstructions[1:]
	err := op.Run(env)
	return err
}

func (p *OperationProcess) HasNext() bool {
	return len(p.processInstructions) != 0 && p.processInstructions[0] != nil
}

func loadCoreModules() {
	commandFactory := command.Factory
	commandMapping[commandFactory.Key()] = commandFactory

	downloadFactory := download.Factory
	commandMapping[downloadFactory.Key()] = downloadFactory

	mkdirFactory := mkdir.Factory
	commandMapping[mkdirFactory.Key()] = mkdirFactory

	moveFactory := move.Factory
	commandMapping[moveFactory.Key()] = moveFactory

	writeFileFactory := writefile.Factory
	commandMapping[writeFileFactory.Key()] = writeFileFactory

	mojangFactory := mojangdl.Factory
	commandMapping[mojangFactory.Key()] = mojangFactory

	spongeforgeDlFactory := spongeforgedl.Factory
	commandMapping[spongeforgeDlFactory.Key()] = spongeforgeDlFactory

	forgeDlFactory := forgedl.Factory
	commandMapping[forgeDlFactory.Key()] = forgeDlFactory
}

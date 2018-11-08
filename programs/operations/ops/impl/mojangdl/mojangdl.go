/*
 Copyright 2018 Padduck, LLC

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

package mojangdl

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pufferpanel/apufferi/common"
	"github.com/pufferpanel/apufferi/logging"
	"github.com/pufferpanel/pufferd/commons"
	"github.com/pufferpanel/pufferd/environments"
	"github.com/pufferpanel/pufferd/programs/operations/ops"
	"net/http"
)

const VERSION_JSON = "https://launchermeta.mojang.com/mc/game/version_manifest.json"

type MojangDl struct {
	Version string
	Target  string
}

func (op MojangDl) Run(env environments.Environment) error {
	client := &http.Client{}

	response, err := client.Get(VERSION_JSON)
	if err != nil {
		return err
	}

	var data MojangLauncherJson
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return err
	}
	err = response.Body.Close()
	if err != nil {
		return err
	}

	var targetVersion string
	switch op.Version {
		case "release":
			targetVersion = data.Latest.Release
		case "latest":
			targetVersion = data.Latest.Release
		case "snapshot":
			targetVersion = data.Latest.Snapshot
		default:
			targetVersion = op.Version
	}

	for _, version := range data.Versions {
		if version.Id == targetVersion {
			logging.Debugf("Version %s json located, downloading from %s", version.Id, version.Url)
			env.DisplayToConsole(fmt.Sprintf("Version %s json located, downloading from %s\n", version.Id, version.Url))
			//now, get the version json for this one...
			return downloadServerFromJson(version.Url, op.Target, env)
		}
	}

	env.DisplayToConsole("Could not locate version " + targetVersion + "\n")

	return errors.New("Version not located: " + op.Version)
}

func downloadServerFromJson(url, target string, env environments.Environment) error {
	client := &http.Client{}
	response, err := client.Get(url)
	if err != nil {
		return err
	}

	var data MojangVersionJson
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return err
	}
	err = response.Body.Close()
	if err != nil {
		return err
	}

	serverBlock := data.Downloads["server"]

	logging.Debugf("Version jar located, downloading from %s", serverBlock.Url)
	env.DisplayToConsole(fmt.Sprintf("Version jar located, downloading from %s\n", serverBlock.Url))

	return commons.DownloadFile(serverBlock.Url, target, env)
}

type MojangDlOperationFactory struct {
}

func (of MojangDlOperationFactory) Create(op ops.CreateOperation) ops.Operation {
	version := op.OperationArgs["version"].(string)
	target := op.OperationArgs["target"].(string)

	version = common.ReplaceTokens(version, op.DataMap)
	target = common.ReplaceTokens(target, op.DataMap)

	return MojangDl{Version: version, Target: target}
}

func (of MojangDlOperationFactory) Key() string {
	return "mojangdl"
}

type MojangLauncherJson struct {
	Versions []MojangLauncherVersion `json:"versions"`
	Latest MojangLatest `json:"latest"`
}

type MojangLatest struct {
	Release string `json:"release"`
	Snapshot string `json:"snapshot"`
}

type MojangLauncherVersion struct {
	Id   string `json:"id"`
	Url  string `json:"url"`
	Type string `json:"type"`
}

type MojangVersionJson struct {
	Downloads map[string]MojangDownloadType `json:"downloads"`
}

type MojangDownloadType struct {
	Sha1 string `json:"sha1"`
	Size uint64 `json:"size"`
	Url  string `json:"url"`
}

var Factory MojangDlOperationFactory
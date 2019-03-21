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

package spongeforgedl

import (
	"encoding/json"
	"errors"
	"github.com/pufferpanel/apufferi/common"
	"github.com/pufferpanel/pufferd/commons"
	"github.com/pufferpanel/pufferd/environments"
	"github.com/pufferpanel/pufferd/environments/envs"
	"github.com/pufferpanel/pufferd/programs/operations/ops"
	"github.com/pufferpanel/pufferd/programs/operations/ops/impl/forgedl"
	"net/http"
	"os"
	"path"
)

const DownloadApiUrl = "https://dl-api.spongepowered.org/v1/org.spongepowered/spongeforge/downloads?type=stable&limit=1"
const RecommendedApiUrl = "https://dl-api.spongepowered.org/v1/org.spongepowered/spongeforge/downloads/recommended"

var client = &http.Client{}

type SpongeForgeDl struct {
	ReleaseType string
}

type download struct {
	Dependencies dependencies        `json:"dependencies"`
	Artifacts    map[string]artifact `json:"artifacts"`
}

type dependencies struct {
	Forge     string `json:"forge"`
	Minecraft string `json:"minecraft"`
}

type artifact struct {
	Url string `json:"url"`
}

func (op SpongeForgeDl) Run(env envs.Environment) error {
	var versionData download

	if op.ReleaseType == "latest" {
		response, err := client.Get(DownloadApiUrl)
		defer commons.CloseResponse(response)
		if err != nil {
			return err
		}

		var all []download
		err = json.NewDecoder(response.Body).Decode(&all)
		if err != nil {
			return err
		}
		err = response.Body.Close()
		if err != nil {
			return err
		}

		versionData = all[0]
	} else {
		response, err := client.Get(RecommendedApiUrl)
		defer commons.CloseResponse(response)

		if err != nil {
			return err
		}

		err = json.NewDecoder(response.Body).Decode(&versionData)
		if err != nil {
			return err
		}
		err = response.Body.Close()
		if err != nil {
			return err
		}
	}

	if versionData.Artifacts == nil || len(versionData.Artifacts) == 0 {
		return errors.New("no artifacts found to download")
	}

	//convert to a forge operation and have built-in process run this
	mapping := make(map[string]interface{})
	mapping["version"] = versionData.Dependencies.Minecraft + "-" + versionData.Dependencies.Forge
	mapping["target"] = "forge-installer.jar"
	forgeDlOp := forgedl.Factory.Create(ops.CreateOperation{OperationArgs: mapping})

	err := forgeDlOp.Run(env)
	if err != nil {
		return err
	}

	err = os.Mkdir(path.Join(env.GetRootDirectory(), "mods"), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	file, err := environments.DownloadViaMaven(versionData.Artifacts[""].Url, env)
	if err != nil {
		return err
	}

	err = common.CopyFile(file, path.Join("mods", "spongeforge.jar"))
	if err != nil {
		return err
	}

	return nil
}

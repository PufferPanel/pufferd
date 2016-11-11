package migration

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pufferpanel/pufferd/logging"
	"encoding/json"
	"github.com/pufferpanel/pufferd/utils"
	"os"
)

const Scales = "/srv/scales/data"

func MigrateFromScales() {
	wd, err := os.Getwd()
	programFiles, err := ioutil.ReadDir(Scales)
	if err != nil {
		logging.Critical("Error reading from old Scales folder", err)
		return
	}
	for _, element := range programFiles {
		if element.IsDir() {
			continue
		}
		id := strings.TrimSuffix(element.Name(), filepath.Ext(element.Name()))
		logging.Infof("Attempting to migrate %s", id)
		data, err := ioutil.ReadAll(utils.JoinPath(Scales, element))
		if err != nil {
			logging.Error("Error read server config " + id, err)
			continue
		}
		scales := scalesServer{}
		err = json.Unmarshal(data, &scales)
		if err != nil {
			logging.Error("Error read server config " + id, err)
			continue
		}
		newPath := utils.JoinPath(wd, "data", "servers", scales.Name)
		err = os.Rename(utils.JoinPath("/home", scales.User), newPath)
		if err != nil {
			logging.Error("Error moving folder", err);
			continue
		}
		err = os.Chown(newPath, os.Getuid(), os.Getgid())
		if err != nil {
			logging.Error("Error changing owner of folder", err);
			continue
		}
	}
}

type scalesServer struct {
	Name     string            `json:"name,omitempty"`
	User     string            `json:"user,omitempty"`
	Build    scalesServerBuild `json:"build,omitempty"`
	Gameport int               `json:"gameport,omitempty"`
	Gamehost string            `json:"gamehost,omitempty"`
	Plugin   string            `json:"plugin,omitempty"`
}

type scalesServerBuild struct {
	Memory string `json:"memory,omitempty"`
}

type scalesServerStartup struct {
	Variables scalesServerStartupVariables `json:"variables,omitempty"`
}

type scalesServerStartupVariables struct {
	Build_Params string `json:"build_params,omitempty"`
}

package permissions

import (
	"encoding/json"
	"os"
	"slices"

	"github.com/rs/zerolog/log"
)

type Permission struct {
	Permissions []string `json:"permissions"`
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Skip        bool     `json:"skip"`
}

type PermissionData struct {
	Endpoints []Permission `json:"endpoints"`
	Skip      bool         `json:"skip"`
}

func (r *PermissionData) FindPermissions(path, method string) Permission {
	idx := slices.IndexFunc(r.Endpoints, func(rp Permission) bool {
		return rp.Path == path && rp.Method == method
	})

	if idx == -1 {
		return Permission{}
	}

	return r.Endpoints[idx]
}

func Get() *PermissionData {
	file, err := os.Open("permissions.json")
	if err != nil {
		log.Err(err).Msg("Failed to open file")

		return nil
	}
	defer file.Close()

	var permissions PermissionData

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&permissions)

	if err != nil {
		log.Err(err).Msg("Failed to decode file")

		return nil
	}

	return &permissions
}

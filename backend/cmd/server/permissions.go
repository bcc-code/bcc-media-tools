package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

	"connectrpc.com/connect"
)

type PermissionsAPI struct{}

func (_ PermissionsAPI) GetPermissions(_ context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.Permissions], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(400, fmt.Errorf("missing email header"))
	}

	permissions := PermissionsForEmail(email)
	res := connect.NewResponse(permissions)
	return res, nil
}

func (_ PermissionsAPI) UpdatePermissions(_ context.Context, req *connect.Request[apiv1.SetPermissionsRequest]) (*connect.Response[apiv1.Void], error) {
	if !IsAdmin(req) {
		return nil, connect.NewError(403, fmt.Errorf("not authorized"))
	}

	permissions, err := readPermissionsFile()
	if err != nil {
		return nil, connect.NewError(500, err)
	}

	email := strings.ToLower(req.Msg.Email)
	permissions[email] = req.Msg.Permissions

	err = writePermissionsFile(permissions)
	if err != nil {
		return nil, connect.NewError(500, err)
	}

	return connect.NewResponse(&apiv1.Void{}), err
}

func (_ PermissionsAPI) DeletePermissions(_ context.Context, req *connect.Request[apiv1.DeletePermissionsRequest]) (*connect.Response[apiv1.Void], error) {
	if !IsAdmin(req) {
		return nil, connect.NewError(403, fmt.Errorf("not authorized"))
	}

	permissions, err := readPermissionsFile()
	if err != nil {
		return nil, connect.NewError(500, err)
	}

	delete(permissions, req.Msg.Email)

	err = writePermissionsFile(permissions)
	if err != nil {
		return nil, connect.NewError(500, err)
	}

	return connect.NewResponse(&apiv1.Void{}), err
}

func (_ PermissionsAPI) ListPermissions(_ context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.PermissionsList], error) {
	if !IsAdmin(req) {
		return nil, connect.NewError(http.StatusUnauthorized, fmt.Errorf("not authorized"))
	}

	permissions, err := readPermissionsFile()
	if err != nil {
		return nil, connect.NewError(500, err)
	}

	res := connect.NewResponse(&apiv1.PermissionsList{
		Permissions: permissions,
	})
	return res, nil
}

var PermissionsFile = path.Join(os.Getenv("CONFIG_ROOT"), "permissions.json")

func readPermissionsFile() (map[string]*apiv1.Permissions, error) {
	permissionsBytes, err := os.ReadFile(PermissionsFile)
	if err != nil {
		return nil, err
	}

	permissions := make(map[string]*apiv1.Permissions)
	err = json.Unmarshal(permissionsBytes, &permissions)
	if err != nil {
		return nil, err
	}

	return permissions, nil
}

func writePermissionsFile(permissions map[string]*apiv1.Permissions) error {
	permissionsBytes, err := json.Marshal(permissions)
	if err != nil {
		return err
	}

	err = os.WriteFile(PermissionsFile, permissionsBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func PermissionsForEmail(email string) *apiv1.Permissions {
	email = strings.ToLower(email)
	permissions, _ := readPermissionsFile()
	if permissions != nil {
		if perms, ok := permissions[email]; ok {
			sort.Slice(perms.Bmm.Podcasts, func(i, j int) bool {
				return perms.Bmm.Podcasts[i] > perms.Bmm.Podcasts[j]
			})
			perms.Email = email
			return perms
		}
	}

	return &apiv1.Permissions{
		Admin: false,
		Email: email,
		Bmm: &apiv1.BMMPermission{
			Albums:    make([]string, 0),
			Languages: make([]string, 0),
			Admin:     false,
		},
	}
}

func IsAdmin[T any](req *connect.Request[T]) bool {
	email := getEmail(req)
	return PermissionsForEmail(email).Admin
}

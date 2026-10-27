package main

import (
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestMaintenance(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))

	h.do(fiber.MethodGet, "/api/admin/maintenance/status", nil).expect(t, 200, `{"active":false,"action":"end"}`)
	h.do(fiber.MethodPost, "/api/admin/maintenance/login", `{}`).expect(t, 400, `{"message":"Not in maintenance mode"}`)
	h.do(fiber.MethodPost, "/api/admin/maintenance/login", `{"token":1}`).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["token"],"message":"Invalid input: expected string, received number"}]}`)

	h.do(fiber.MethodPost, "/api/admin/maintenance", `{"action":"end"}`, owner...).expect(t, 201, "")
	h.do(fiber.MethodPost, "/api/admin/maintenance", `{"action":"start"}`, owner...).
		expect(t, 400, `{"message":"Maintenance mode is not supported by korda"}`)
	h.do(fiber.MethodPost, "/api/admin/maintenance", `{"action":"restore_database"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"custom","path":["restoreBackupFilename"],"message":"Backup filename is required when action is restore_database"}]}`)
	h.do(fiber.MethodPost, "/api/admin/maintenance", `{"action":"end"}`, member...).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodGet, "/api/admin/maintenance/detect-install", nil, owner...).
		expect(t, 200, `{"storage":[{"folder":"encoded-video","readable":false,"writable":false,"files":0},{"folder":"library","readable":false,"writable":false,"files":0},{"folder":"upload","readable":false,"writable":false,"files":0},{"folder":"profile","readable":false,"writable":false,"files":0},{"folder":"thumbs","readable":false,"writable":false,"files":0},{"folder":"backups","readable":false,"writable":false,"files":0}]}`)

	h.do(fiber.MethodGet, "/api/admin/integrity/summary", nil, owner...).
		expect(t, 200, `{"untracked_file":0,"missing_file":0,"checksum_mismatch":0}`)
	h.do(fiber.MethodGet, "/api/admin/integrity/report?type=missing_file", nil, owner...).expect(t, 200, `{"items":[]}`)
	h.do(fiber.MethodGet, "/api/admin/integrity/report?type=missing_file&cursor=x", nil, owner...).
		expect(t, 500, `{"message":"Failed to get integrity report"}`)
	h.do(fiber.MethodGet, "/api/admin/integrity/report", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["untracked_file","missing_file","checksum_mismatch"],"path":["type"],"message":"Invalid option: expected one of \"untracked_file\"|\"missing_file\"|\"checksum_mismatch\""}]}`)
	h.do(fiber.MethodGet, "/api/admin/integrity/report/019a1234-5678-7abc-8def-0123456789ab/file", nil, owner...).
		expect(t, 404, `{"message":"Not Found"}`)
	h.do(fiber.MethodDelete, "/api/admin/integrity/report/019a1234-5678-7abc-8def-0123456789ab", nil, owner...).
		expect(t, 500, `{"message":"Failed to delete integrity report"}`)
	h.do(fiber.MethodDelete, "/api/admin/integrity/report/0f0e0d0c-0b0a-4908-8706-050403020100", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["id"],"message":"Invalid UUID"}]}`)
	csv := h.do(fiber.MethodGet, "/api/admin/integrity/report/untracked_file/csv", nil, owner...)
	if csv.Status != 200 || string(csv.Body) != "id,type,assetId,fileAssetId,path\n" ||
		csv.Header.Get(fiber.HeaderContentType) != "text/csv; charset=utf-8" ||
		!regexp.MustCompile(`^attachment; filename="\d{13}-untracked_file\.csv"$`).MatchString(csv.Header.Get(fiber.HeaderContentDisposition)) {
		t.Errorf("csv = %d %v %q", csv.Status, csv.Header, csv.Body)
	}

	h.do(fiber.MethodGet, "/api/admin/database-backups", nil, owner...).expect(t, 200, `{"backups":[]}`)
	h.do(fiber.MethodGet, "/api/admin/database-backups/x.sql", nil, owner...).expect(t, 404, `{"message":"Not Found"}`)
	h.do(fiber.MethodGet, "/api/admin/database-backups/a%20b.sql", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^[a-zA-Z0-9_\\-.]+$/","path":["filename"],"message":"Filename contains invalid characters"}]}`)
	h.do(fiber.MethodDelete, "/api/admin/database-backups", `{"backups":["x.sql","x.txt"]}`, owner...).
		expect(t, 400, `{"message":"Invalid backup name!"}`)
	h.do(fiber.MethodDelete, "/api/admin/database-backups", `{"backups":["x.sql"]}`, owner...).
		expect(t, 200, "")
	h.multipart("/api/admin/database-backups/upload", []part{{field: "file", filename: "x.txt", content: []byte("x")}}, owner...).
		expect(t, 400, `{"message":"Invalid backup name!"}`)
	h.multipart("/api/admin/database-backups/upload", []part{{field: "file", filename: "../x.sql", content: []byte("x")}}, owner...).
		expect(t, 201, "")
	h.do(fiber.MethodGet, "/api/admin/database-backups/uploaded-x.sql", nil, owner...).expect(t, 200, "x")
	h.multipart("/api/admin/database-backups/upload", []part{{field: "name", content: []byte("x")}}, owner...).
		expect(t, 500, `{"message":"Failed to upload database backup"}`)
	h.do(fiber.MethodPost, "/api/admin/database-backups/start-restore", nil).
		expect(t, 400, `{"message":"Admin setup is not available"}`)

	css := h.do(fiber.MethodGet, "/custom.css", nil)
	if css.Status != 200 || len(css.Body) != 0 || css.Header.Get(fiber.HeaderContentType) != "text/css; charset=utf-8" {
		t.Errorf("custom.css = %d %v %q", css.Status, css.Header, css.Body)
	}
}

func TestClusterGroups(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	miraID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	me := h.do(fiber.MethodGet, "/api/users/me", nil, owner...).object(t)
	group := me["clusterGroupId"].(string)
	other := "0f0e0d0c-0b0a-4908-8706-050403020100"

	h.do(fiber.MethodGet, "/api/cluster-groups/requests", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/cluster-groups/"+group+"/requests", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/cluster-groups/"+other+"/requests", nil, owner...).
		expect(t, 400, `{"message":"Not found or no clusterGroup.read access"}`)
	users := h.do(fiber.MethodGet, "/api/cluster-groups/"+group+"/users", nil, owner...).array(t)
	if len(users) != 1 || users[0].(map[string]any)["id"] != me["id"] {
		t.Errorf("users = %v", users)
	}
	h.do(fiber.MethodGet, "/api/cluster-groups/"+other+"/users", nil, owner...).
		expect(t, 400, `{"message":"Not found or no clusterGroup.read access"}`)

	h.do(fiber.MethodPut, "/api/cluster-groups/"+group+"/requests", `{"userId":"`+me["id"].(string)+`"}`, owner...).
		expect(t, 400, `{"message":"Cannot request to join your own cluster group"}`)
	h.do(fiber.MethodPut, "/api/cluster-groups/"+group+"/requests", `{"userId":"`+other+`"}`, owner...).
		expect(t, 400, `{"message":"User not found"}`)
	h.do(fiber.MethodPut, "/api/cluster-groups/"+group+"/requests", `{"userId":"`+miraID+`"}`, owner...).
		expect(t, 400, `{"message":"Cluster group invitations are not supported by korda"}`)
	h.do(fiber.MethodPut, "/api/cluster-groups/"+other+"/requests", `{"userId":"`+miraID+`"}`, owner...).
		expect(t, 400, `{"message":"Not found or no clusterGroupRequest.create access"}`)
	h.do(fiber.MethodPost, "/api/cluster-groups/requests/"+other+"/accept", nil, owner...).
		expect(t, 400, `{"message":"Not found or no clusterGroupRequest.read access"}`)
	h.do(fiber.MethodDelete, "/api/cluster-groups/requests/"+other, nil, owner...).
		expect(t, 400, `{"message":"Not found or no clusterGroupRequest.delete access"}`)
	h.do(fiber.MethodPost, "/api/cluster-groups/"+group+"/leave", nil, owner...).
		expect(t, 400, `{"message":"Cannot leave a cluster group without any other members"}`)
	h.do(fiber.MethodPost, "/api/cluster-groups/"+group+"/regenerate-people", nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodPost, "/api/cluster-groups/"+other+"/regenerate-people", nil, owner...).
		expect(t, 400, `{"message":"Not found or no clusterGroup.read access"}`)
}

func TestWorkflows(t *testing.T) {
	h := newHarness(t)
	owner := bearer(h.admin())
	id := "0f0e0d0c-0b0a-4908-8706-050403020100"

	h.do(fiber.MethodGet, "/api/plugins", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/plugins/methods?trigger=AssetCreate", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/plugins/methods?type=AssetV1", nil, owner...).expect(t, 500, `{"message":"Failed to search plugin methods"}`)
	h.do(fiber.MethodGet, "/api/plugins/templates", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/plugins/"+id, nil, owner...).expect(t, 400, `{"message":"Plugin not found"}`)

	h.do(fiber.MethodGet, "/api/workflows/triggers", nil, owner...).
		expect(t, 200, `[{"trigger":"AssetCreate","types":["AssetV1"]},{"trigger":"AssetMetadataExtraction","types":["AssetV1"]},{"trigger":"AssetTagged","types":["AssetV1"]}]`)
	h.do(fiber.MethodGet, "/api/workflows", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodPost, "/api/workflows", `{"trigger":"AssetCreate"}`, owner...).
		expect(t, 400, `{"message":"Workflows are not supported by korda"}`)
	h.do(fiber.MethodPost, "/api/workflows", `{"trigger":"AssetCreate","steps":[{"method":"immich-plugin-core#assetArchive","config":null}]}`, owner...).
		expect(t, 400, `{"message":"Unknown method immich-plugin-core#assetArchive"}`)
	h.do(fiber.MethodGet, "/api/workflows/"+id, nil, owner...).expect(t, 400, `{"message":"Not found or no workflow.read access"}`)
	h.do(fiber.MethodGet, "/api/workflows/"+id+"/share", nil, owner...).expect(t, 400, `{"message":"Not found or no workflow.read access"}`)
	h.do(fiber.MethodPut, "/api/workflows/"+id, `{}`, owner...).expect(t, 400, `{"message":"Not found or no workflow.update access"}`)
	h.do(fiber.MethodPatch, "/api/workflows/"+id, `{"enabled":"true"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["enabled"],"message":"Invalid input: expected boolean, received string"}]}`)
	h.do(fiber.MethodDelete, "/api/workflows/"+id, nil, owner...).expect(t, 400, `{"message":"Not found or no workflow.delete access"}`)
	h.do(fiber.MethodGet, "/api/workflows/"+id+"/logs?limit=5", nil, owner...).expect(t, 400, `{"message":"Not found or no workflow.logs access"}`)
}

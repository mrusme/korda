package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestConformanceAdmin(t *testing.T) {
	base := os.Getenv(envReferenceURL)
	email := os.Getenv(envReferenceEmail)
	password := os.Getenv(envReferencePassword)
	if base == "" || email == "" || password == "" {
		t.Skipf("set %s, %s and %s to compare korda with an Immich", envReferenceURL, envReferenceEmail, envReferencePassword)
	}

	immich := immichSide(t, strings.TrimSuffix(base, "/"))
	login := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	if login.Status != fiber.StatusCreated {
		t.Fatalf("login to %s = %d %s", base, login.Status, login.Body)
	}
	immich.tokens["session"] = login.str(t, "accessToken")
	defer immich.send(fiber.MethodPost, "/api/auth/logout", "", bearer(immich.tokens["session"])...)
	me := immich.send(fiber.MethodGet, "/api/users/me", "", bearer(immich.tokens["session"])...).object(t)

	h := newHarness(t)
	korda := kordaSide(h)
	h.do(fiber.MethodPost, "/api/auth/admin-sign-up", map[string]string{"email": email, "password": password, "name": me["name"].(string)}).expect(t, 201, "")
	korda.tokens["session"] = h.login(email, password)

	for _, s := range []*side{immich, korda} {
		key := s.send(fiber.MethodPost, "/api/api-keys", `{"name":"conformance","permissions":["user.read"]}`, bearer(s.tokens["session"])...)
		if key.Status != fiber.StatusCreated {
			t.Fatalf("%s api key = %d %s", s.name, key.Status, key.Body)
		}
		s.tokens["key"] = key.str(t, "secret")
		s.vars["readkey.id"] = key.str(t, "id")
		s.vars["group"] = s.send(fiber.MethodGet, "/api/users/me", "", bearer(s.tokens["session"])...).str(t, "clusterGroupId")
	}
	defer immich.send(fiber.MethodDelete, "/api/api-keys/"+immich.vars["readkey.id"], "", bearer(immich.tokens["session"])...)

	const (
		zero = "00000000-0000-4000-8000-000000000000"
		v7   = "019a1234-5678-7abc-8def-0123456789ab"
	)
	steps := []step{
		{name: "maintenance status", method: "GET", path: "/api/admin/maintenance/status"},
		{name: "maintenance status with session", method: "GET", path: "/api/admin/maintenance/status", auth: "session"},
		{name: "maintenance login", method: "POST", path: "/api/admin/maintenance/login", body: `{}`},
		{name: "maintenance login invalid", method: "POST", path: "/api/admin/maintenance/login", body: `{"token":1}`},
		{name: "maintenance end", method: "POST", path: "/api/admin/maintenance", body: `{"action":"end"}`, auth: "session"},
		{name: "maintenance restore without file", method: "POST", path: "/api/admin/maintenance", body: `{"action":"restore_database"}`, auth: "session"},
		{name: "maintenance invalid", method: "POST", path: "/api/admin/maintenance", body: `{"action":1,"restoreBackupFilename":2}`, auth: "session"},
		{name: "maintenance without auth", method: "POST", path: "/api/admin/maintenance", body: `{"action":"end"}`},
		{name: "maintenance with key", method: "POST", path: "/api/admin/maintenance", body: `{"action":"end"}`, auth: "key"},
		{name: "integrity summary", method: "GET", path: "/api/admin/integrity/summary", auth: "session", mode: compareShape},
		{name: "integrity report", method: "GET", path: "/api/admin/integrity/report?type=missing_file", auth: "session", mode: compareShape},
		{name: "integrity report cursor", method: "GET", path: "/api/admin/integrity/report?type=missing_file&cursor=x", auth: "session"},
		{name: "integrity report invalid", method: "GET", path: "/api/admin/integrity/report?limit=1.5&cursor=a&cursor=b", auth: "session"},
		{name: "integrity file v4", method: "GET", path: "/api/admin/integrity/report/" + zero + "/file", auth: "session"},
		{name: "integrity file missing", method: "GET", path: "/api/admin/integrity/report/" + v7 + "/file", auth: "session"},
		{name: "integrity delete missing", method: "DELETE", path: "/api/admin/integrity/report/" + v7, auth: "session"},
		{name: "integrity csv invalid", method: "GET", path: "/api/admin/integrity/report/x/csv", auth: "session"},
		{name: "integrity csv", method: "GET", path: "/api/admin/integrity/report/checksum_mismatch/csv", auth: "session", header: fiber.HeaderContentType},
		{name: "backup download invalid", method: "GET", path: "/api/admin/database-backups/a%20b.sql", auth: "session"},
		{name: "backup download missing", method: "GET", path: "/api/admin/database-backups/korda-missing.sql", auth: "session"},
		{name: "backup download not a backup", method: "GET", path: "/api/admin/database-backups/korda.txt", auth: "session"},
		{name: "backup delete missing", method: "DELETE", path: "/api/admin/database-backups", body: `{"backups":["korda-missing.sql"]}`, auth: "session"},
		{name: "backup delete invalid", method: "DELETE", path: "/api/admin/database-backups", body: `{"backups":["korda.sql",1]}`, auth: "session"},
		{name: "backup delete name", method: "DELETE", path: "/api/admin/database-backups", body: `{"backups":["korda.sql","korda.txt"]}`, auth: "session"},
		{name: "backup upload without file", method: "POST", path: "/api/admin/database-backups/upload", body: `{}`, auth: "session"},
		{name: "backup restore flow", method: "POST", path: "/api/admin/database-backups/start-restore"},
		{name: "backup list with key", method: "GET", path: "/api/admin/database-backups", auth: "key"},
		{name: "cluster requests", method: "GET", path: "/api/cluster-groups/requests", auth: "session"},
		{name: "cluster own requests", method: "GET", path: "/api/cluster-groups/{group}/requests", auth: "session"},
		{name: "cluster own users", method: "GET", path: "/api/cluster-groups/{group}/users", auth: "session", mode: compareShape},
		{name: "cluster other users", method: "GET", path: "/api/cluster-groups/" + zero + "/users", auth: "session"},
		{name: "cluster invalid id", method: "GET", path: "/api/cluster-groups/x/users", auth: "session"},
		{name: "cluster request other group", method: "PUT", path: "/api/cluster-groups/" + zero + "/requests", body: `{"userId":"` + zero + `"}`, auth: "session"},
		{name: "cluster request invalid", method: "PUT", path: "/api/cluster-groups/{group}/requests", body: `{"userId":"` + v7 + `"}`, auth: "session"},
		{name: "cluster request unknown user", method: "PUT", path: "/api/cluster-groups/{group}/requests", body: `{"userId":"` + zero + `"}`, auth: "session"},
		{name: "cluster accept", method: "POST", path: "/api/cluster-groups/requests/" + zero + "/accept", auth: "session"},
		{name: "cluster decline", method: "DELETE", path: "/api/cluster-groups/requests/" + zero, auth: "session"},
		{name: "cluster leave other", method: "POST", path: "/api/cluster-groups/" + zero + "/leave", auth: "session"},
		{name: "cluster regenerate other", method: "POST", path: "/api/cluster-groups/" + zero + "/regenerate-people", auth: "session"},
		{name: "plugin missing", method: "GET", path: "/api/plugins/" + zero, auth: "session"},
		{name: "plugin invalid", method: "GET", path: "/api/plugins/x", auth: "session"},
		{name: "plugin search invalid", method: "GET", path: "/api/plugins?enabled=true&id=x", auth: "session"},
		{name: "plugin methods by type", method: "GET", path: "/api/plugins/methods?type=AssetV1", auth: "session"},
		{name: "plugin methods invalid", method: "GET", path: "/api/plugins/methods?type=x&trigger=x", auth: "session"},
		{name: "workflow triggers", method: "GET", path: "/api/workflows/triggers", auth: "session"},
		{name: "workflow triggers with key", method: "GET", path: "/api/workflows/triggers", auth: "key"},
		{name: "workflow search invalid", method: "GET", path: "/api/workflows?trigger=x&logging=false", auth: "session"},
		{name: "workflow create invalid", method: "POST", path: "/api/workflows", body: `{"trigger":"AssetCreate","steps":[{"config":[]}]}`, auth: "session"},
		{name: "workflow create unknown method", method: "POST", path: "/api/workflows", body: `{"trigger":"AssetCreate","steps":[{"method":"korda#missing","config":null}]}`, auth: "session"},
		{name: "workflow missing", method: "GET", path: "/api/workflows/" + zero, auth: "session"},
		{name: "workflow share missing", method: "GET", path: "/api/workflows/" + zero + "/share", auth: "session"},
		{name: "workflow update missing", method: "PUT", path: "/api/workflows/" + zero, body: `{}`, auth: "session"},
		{name: "workflow update invalid", method: "PATCH", path: "/api/workflows/" + zero, body: `{"trigger":"x","enabled":"true"}`, auth: "session"},
		{name: "workflow delete missing", method: "DELETE", path: "/api/workflows/" + zero, auth: "session"},
		{name: "workflow logs invalid", method: "GET", path: "/api/workflows/" + zero + "/logs?result=x&before=2024-01-01&limit=0", auth: "session"},
		{name: "workflow logs missing", method: "GET", path: "/api/workflows/" + zero + "/logs", auth: "session"},
		{name: "config defaults", method: "GET", path: "/api/system-config/defaults", auth: "session", mode: compareShape},
		{name: "config storage template options", method: "GET", path: "/api/system-config/storage-template-options", auth: "session"},
		{name: "config update empty", method: "PUT", path: "/api/system-config", body: `{}`, auth: "session"},
		{name: "config", method: "GET", path: "/api/system-config", auth: "session", mode: compareShape},
		{name: "queues", method: "GET", path: "/api/queues", auth: "session", mode: compareShape},
		{name: "queue", method: "GET", path: "/api/queues/thumbnailGeneration", auth: "session", mode: compareShape},
		{name: "queue invalid", method: "GET", path: "/api/queues/x", auth: "session"},
		{name: "queue update invalid", method: "PUT", path: "/api/queues/thumbnailGeneration", body: `{"isPaused":"x"}`, auth: "session"},
		{name: "queue jobs invalid", method: "GET", path: "/api/queues/thumbnailGeneration/jobs?status=x", auth: "session"},
		{name: "jobs legacy", method: "GET", path: "/api/jobs", auth: "session", mode: compareShape},
		{name: "jobs create invalid", method: "POST", path: "/api/jobs", body: `{"name":"x"}`, auth: "session"},
		{name: "jobs create backup", method: "POST", path: "/api/jobs", body: `{"name":"backup-database"}`, auth: "session"},
		{name: "jobs command invalid", method: "PUT", path: "/api/jobs/x", body: `{"command":"x"}`, auth: "session"},
		{name: "notification missing", method: "GET", path: "/api/notifications/" + zero, auth: "session"},
		{name: "notifications invalid", method: "GET", path: "/api/notifications?unread=x&level=x", auth: "session"},
		{name: "notification create invalid", method: "POST", path: "/api/admin/notifications", body: `{}`, auth: "session"},
		{name: "notification test email invalid", method: "POST", path: "/api/admin/notifications/test-email", body: `{}`, auth: "session"},
		{name: "oauth authorize", method: "POST", path: "/api/oauth/authorize", body: `{"redirectUri":"http://localhost/auth/login"}`},
		{name: "oauth authorize invalid", method: "POST", path: "/api/oauth/authorize", body: `{}`},
		{name: "oauth callback invalid", method: "POST", path: "/api/oauth/callback", body: `{}`},
		{name: "oauth link invalid", method: "POST", path: "/api/oauth/link", body: `{}`, auth: "session"},
		{name: "libraries", method: "GET", path: "/api/libraries", auth: "session"},
		{name: "library missing", method: "GET", path: "/api/libraries/" + zero, auth: "session"},
		{name: "library create invalid", method: "POST", path: "/api/libraries", body: `{}`, auth: "session"},
		{name: "people", method: "GET", path: "/api/people", auth: "session"},
		{name: "person missing", method: "GET", path: "/api/people/" + zero, auth: "session"},
		{name: "people invalid", method: "GET", path: "/api/people?withHidden=x&page=0", auth: "session"},
		{name: "duplicates", method: "GET", path: "/api/duplicates", auth: "session"},
		{name: "reverse geocode invalid", method: "GET", path: "/api/map/reverse-geocode?lat=x", auth: "session"},
		{name: "timeline bucket unparsable", method: "GET", path: "/api/timeline/bucket?timeBucket=abc", auth: "session"},
		{name: "timeline bbox invalid", method: "GET", path: "/api/timeline/buckets?bbox=0,10,10,0", auth: "session"},
		{name: "download info without target", method: "POST", path: "/api/download/info", body: `{}`, auth: "session"},
		{name: "download archive invalid", method: "POST", path: "/api/download/archive", body: `{"assetIds":"x","edited":"true"}`, auth: "session"},
		{name: "custom css", method: "GET", path: "/custom.css", header: fiber.HeaderContentType},
		{name: "custom css below api", method: "GET", path: "/api/custom.css"},
	}

	compareSteps(t, immich, korda, steps)
}

package request

import (
	"testing"

	"tty.fail/mrus/korda/logic/root/clustergroups"
	"tty.fail/mrus/korda/logic/root/maintenance"
	"tty.fail/mrus/korda/logic/root/workflows"
)

func TestMiscBodiesMatchImmich(t *testing.T) {
	newOf := map[string]func() any{
		"SetMaintenanceModeDto":        func() any { return new(maintenance.SetModeInput) },
		"MaintenanceLoginDto":          func() any { return new(maintenance.LoginInput) },
		"IntegrityGetReportDto":        func() any { return new(maintenance.ReportInput) },
		"IntegrityReportTypeParamDto":  func() any { return new(maintenance.ReportTypeParams) },
		"UUIDv7ParamDto":               func() any { return new(maintenance.ReportIDParams) },
		"FilenameParamDto":             func() any { return new(maintenance.FilenameParams) },
		"DatabaseBackupDeleteDto":      func() any { return new(maintenance.DeleteBackupsInput) },
		"ClusterGroupRequestCreateDto": func() any { return new(clustergroups.RequestInput) },
		"PluginSearchDto":              func() any { return new(workflows.PluginSearchInput) },
		"PluginMethodSearchDto":        func() any { return new(workflows.MethodSearchInput) },
		"WorkflowSearchDto":            func() any { return new(workflows.SearchInput) },
		"WorkflowCreateDto":            func() any { return new(workflows.CreateInput) },
		"WorkflowUpdateDto":            func() any { return new(workflows.UpdateInput) },
		"WorkflowGetLogsDto":           func() any { return new(workflows.LogsInput) },
	}
	tests := []struct {
		name string
		dto  string
		mode string
		body string
		want string
	}{
		{name: "SetMaintenanceModeDto end", dto: "SetMaintenanceModeDto", mode: "body", body: `{"action":"end"}`, want: ``},
		{name: "SetMaintenanceModeDto bad", dto: "SetMaintenanceModeDto", mode: "body", body: `{"action":"x"}`, want: `[{"code":"invalid_value","values":["start","end","select_database_restore","restore_database"],"path":["action"],"message":"Invalid option: expected one of \"start\"|\"end\"|\"select_database_restore\"|\"restore_database\""}]`},
		{name: "SetMaintenanceModeDto restore no file", dto: "SetMaintenanceModeDto", mode: "body", body: `{"action":"restore_database"}`, want: `[{"code":"custom","path":["restoreBackupFilename"],"message":"Backup filename is required when action is restore_database"}]`},
		{name: "SetMaintenanceModeDto restore empty", dto: "SetMaintenanceModeDto", mode: "body", body: `{"action":"restore_database","restoreBackupFilename":""}`, want: `[{"code":"custom","path":["restoreBackupFilename"],"message":"Backup filename is required when action is restore_database"}]`},
		{name: "SetMaintenanceModeDto bad both", dto: "SetMaintenanceModeDto", mode: "body", body: `{"action":1,"restoreBackupFilename":2}`, want: `[{"code":"invalid_value","values":["start","end","select_database_restore","restore_database"],"path":["action"],"message":"Invalid option: expected one of \"start\"|\"end\"|\"select_database_restore\"|\"restore_database\""},{"expected":"string","code":"invalid_type","path":["restoreBackupFilename"],"message":"Invalid input: expected string, received number"}]`},
		{name: "SetMaintenanceModeDto empty", dto: "SetMaintenanceModeDto", mode: "body", body: `{}`, want: `[{"code":"invalid_value","values":["start","end","select_database_restore","restore_database"],"path":["action"],"message":"Invalid option: expected one of \"start\"|\"end\"|\"select_database_restore\"|\"restore_database\""}]`},
		{name: "MaintenanceLoginDto empty", dto: "MaintenanceLoginDto", mode: "body", body: `{}`, want: ``},
		{name: "MaintenanceLoginDto number", dto: "MaintenanceLoginDto", mode: "body", body: `{"token":1}`, want: `[{"expected":"string","code":"invalid_type","path":["token"],"message":"Invalid input: expected string, received number"}]`},
		{name: "IntegrityGetReportDto empty", dto: "IntegrityGetReportDto", mode: "query", body: `{}`, want: `[{"code":"invalid_value","values":["untracked_file","missing_file","checksum_mismatch"],"path":["type"],"message":"Invalid option: expected one of \"untracked_file\"|\"missing_file\"|\"checksum_mismatch\""}]`},
		{name: "IntegrityGetReportDto ok", dto: "IntegrityGetReportDto", mode: "query", body: `{"type":"missing_file","cursor":"","limit":" 7 "}`, want: ``},
		{name: "IntegrityGetReportDto limit zero", dto: "IntegrityGetReportDto", mode: "query", body: `{"type":"missing_file","limit":"0"}`, want: `[{"origin":"number","code":"too_small","minimum":0,"inclusive":false,"path":["limit"],"message":"Too small: expected number to be >0"}]`},
		{name: "IntegrityGetReportDto limit empty", dto: "IntegrityGetReportDto", mode: "query", body: `{"type":"missing_file","limit":""}`, want: `[{"origin":"number","code":"too_small","minimum":0,"inclusive":false,"path":["limit"],"message":"Too small: expected number to be >0"}]`},
		{name: "IntegrityGetReportDto limit fraction", dto: "IntegrityGetReportDto", mode: "query", body: `{"type":"missing_file","limit":"1.5"}`, want: `[{"expected":"int","format":"safeint","code":"invalid_type","path":["limit"],"message":"Invalid input: expected int, received number"}]`},
		{name: "IntegrityGetReportDto limit text", dto: "IntegrityGetReportDto", mode: "query", body: `{"type":"missing_file","limit":"abc","cursor":["a","b"]}`, want: `[{"expected":"string","code":"invalid_type","path":["cursor"],"message":"Invalid input: expected string, received array"},{"expected":"number","code":"invalid_type","received":"NaN","path":["limit"],"message":"Invalid input: expected number, received NaN"}]`},
		{name: "IntegrityReportTypeParamDto bad", dto: "IntegrityReportTypeParamDto", mode: "query", body: `{"type":"x"}`, want: `[{"code":"invalid_value","values":["untracked_file","missing_file","checksum_mismatch"],"path":["type"],"message":"Invalid option: expected one of \"untracked_file\"|\"missing_file\"|\"checksum_mismatch\""}]`},
		{name: "UUIDv7ParamDto v4", dto: "UUIDv7ParamDto", mode: "query", body: `{"id":"0f0e0d0c-0b0a-4908-8706-050403020100"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["id"],"message":"Invalid UUID"}]`},
		{name: "UUIDv7ParamDto v7", dto: "UUIDv7ParamDto", mode: "query", body: `{"id":"019a1234-5678-7abc-8def-0123456789ab"}`, want: ``},
		{name: "FilenameParamDto space", dto: "FilenameParamDto", mode: "query", body: `{"filename":"a b.sql"}`, want: `[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^[a-zA-Z0-9_\\-.]+$/","path":["filename"],"message":"Filename contains invalid characters"}]`},
		{name: "FilenameParamDto ok", dto: "FilenameParamDto", mode: "query", body: `{"filename":"x.txt"}`, want: ``},
		{name: "DatabaseBackupDeleteDto empty", dto: "DatabaseBackupDeleteDto", mode: "body", body: `{}`, want: `[{"expected":"array","code":"invalid_type","path":["backups"],"message":"Invalid input: expected array, received undefined"}]`},
		{name: "DatabaseBackupDeleteDto bad item", dto: "DatabaseBackupDeleteDto", mode: "body", body: `{"backups":["a.sql",1]}`, want: `[{"expected":"string","code":"invalid_type","path":["backups",1],"message":"Invalid input: expected string, received number"}]`},
		{name: "ClusterGroupRequestCreateDto empty", dto: "ClusterGroupRequestCreateDto", mode: "body", body: `{}`, want: `[{"expected":"string","code":"invalid_type","path":["userId"],"message":"Invalid input: expected string, received undefined"}]`},
		{name: "ClusterGroupRequestCreateDto v7", dto: "ClusterGroupRequestCreateDto", mode: "body", body: `{"userId":"019a1234-5678-7abc-8def-0123456789ab"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["userId"],"message":"Invalid UUID"}]`},
		{name: "PluginSearchDto bool", dto: "PluginSearchDto", mode: "query", body: `{"enabled":"true","id":"x"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["id"],"message":"Invalid UUID"},{"expected":"boolean","code":"invalid_type","path":["enabled"],"message":"Invalid input: expected boolean, received string"}]`},
		{name: "PluginMethodSearchDto type", dto: "PluginMethodSearchDto", mode: "query", body: `{"type":"x","trigger":"x"}`, want: `[{"code":"invalid_value","values":["AssetV1"],"path":["type"],"message":"Invalid input: expected \"AssetV1\""},{"code":"invalid_value","values":["AssetCreate","AssetMetadataExtraction","AssetTagged"],"path":["trigger"],"message":"Invalid option: expected one of \"AssetCreate\"|\"AssetMetadataExtraction\"|\"AssetTagged\""}]`},
		{name: "WorkflowSearchDto bool", dto: "WorkflowSearchDto", mode: "query", body: `{"enabled":"true","logging":"false","id":"x"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["id"],"message":"Invalid UUID"},{"expected":"boolean","code":"invalid_type","path":["enabled"],"message":"Invalid input: expected boolean, received string"},{"expected":"boolean","code":"invalid_type","path":["logging"],"message":"Invalid input: expected boolean, received string"}]`},
		{name: "WorkflowSearchDto trigger", dto: "WorkflowSearchDto", mode: "query", body: `{"trigger":"x"}`, want: `[{"code":"invalid_value","values":["AssetCreate","AssetMetadataExtraction","AssetTagged"],"path":["trigger"],"message":"Invalid option: expected one of \"AssetCreate\"|\"AssetMetadataExtraction\"|\"AssetTagged\""}]`},
		{name: "WorkflowCreateDto empty", dto: "WorkflowCreateDto", mode: "body", body: `{}`, want: `[{"code":"invalid_value","values":["AssetCreate","AssetMetadataExtraction","AssetTagged"],"path":["trigger"],"message":"Invalid option: expected one of \"AssetCreate\"|\"AssetMetadataExtraction\"|\"AssetTagged\""}]`},
		{name: "WorkflowCreateDto ok", dto: "WorkflowCreateDto", mode: "body", body: `{"trigger":"AssetCreate","name":null,"steps":[{"method":"x","config":null}]}`, want: ``},
		{name: "WorkflowCreateDto bad step", dto: "WorkflowCreateDto", mode: "body", body: `{"trigger":"AssetCreate","steps":[{"config":[]}]}`, want: `[{"expected":"string","code":"invalid_type","path":["steps",0,"method"],"message":"Invalid input: expected string, received undefined"},{"expected":"record","code":"invalid_type","path":["steps",0,"config"],"message":"Invalid input: expected record, received array"}]`},
		{name: "WorkflowUpdateDto empty", dto: "WorkflowUpdateDto", mode: "body", body: `{}`, want: ``},
		{name: "WorkflowUpdateDto bad", dto: "WorkflowUpdateDto", mode: "body", body: `{"trigger":"x","enabled":"true"}`, want: `[{"code":"invalid_value","values":["AssetCreate","AssetMetadataExtraction","AssetTagged"],"path":["trigger"],"message":"Invalid option: expected one of \"AssetCreate\"|\"AssetMetadataExtraction\"|\"AssetTagged\""},{"expected":"boolean","code":"invalid_type","path":["enabled"],"message":"Invalid input: expected boolean, received string"}]`},
		{name: "WorkflowGetLogsDto bad", dto: "WorkflowGetLogsDto", mode: "query", body: `{"result":"x","before":"2024-01-01","limit":"0"}`, want: `[{"code":"invalid_value","values":["completed","halted","error"],"path":["result"],"message":"Invalid option: expected one of \"completed\"|\"halted\"|\"error\""},{"origin":"string","code":"invalid_format","format":"datetime","pattern":"/^(?:(?:\\d\\d[2468][048]|\\d\\d[13579][26]|\\d\\d0[48]|[02468][048]00|[13579][26]00)-02-29|\\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\\d|30)|(?:02)-(?:0[1-9]|1\\d|2[0-8])))T(?:(?:[01]\\d|2[0-3]):[0-5]\\d(?::[0-5]\\d(?:\\.\\d+)?)?(?:Z|([+-](?:[01]\\d|2[0-3]):[0-5]\\d)))$/","path":["before"],"message":"Invalid input: expected ISO 8601 datetime string, received string"},{"origin":"number","code":"too_small","minimum":0,"inclusive":false,"path":["limit"],"message":"Too small: expected number to be >0"}]`},
		{name: "WorkflowGetLogsDto empty", dto: "WorkflowGetLogsDto", mode: "query", body: `{}`, want: ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, present, err := decodeJSON([]byte(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			m := modeBody
			if tt.mode == "query" {
				m = modeQuery
			}
			err = bind(raw, present, newOf[tt.dto](), m)
			switch {
			case tt.want == "" && err != nil:
				t.Errorf("error %v, want none", err)
			case tt.want != "" && err == nil:
				t.Errorf("no error, want %s", tt.want)
			case tt.want != "":
				if got := issuesOf(t, err); got != tt.want {
					t.Errorf("issues %s\n  want %s", got, tt.want)
				}
			}
		})
	}
}

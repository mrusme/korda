package request

import (
	"testing"

	"tty.fail/mrus/korda/logic/root/people"
)

func TestPeopleBodiesMatchImmich(t *testing.T) {
	newOf := map[string]func() any{
		"PersonSearchDto":      func() any { return new(people.SearchInput) },
		"PersonCreateDto":      func() any { return new(people.CreateInput) },
		"PersonUpdateDto":      func() any { return new(people.UpdateInput) },
		"PeopleUpdateDto":      func() any { return new(people.UpdateAllInput) },
		"MergePersonDto":       func() any { return new(people.MergeInput) },
		"PersonDeleteDto":      func() any { return new(people.DeleteInput) },
		"PeopleDeleteDto":      func() any { return new(people.DeleteAllInput) },
		"AssetFaceUpdateDto":   func() any { return new(people.ReassignInput) },
		"FaceDto":              func() any { return new(people.FaceInput) },
		"AssetFaceCreateDto":   func() any { return new(people.FaceCreateInput) },
		"AssetFaceDeleteDto":   func() any { return new(people.FaceDeleteInput) },
		"PersonUsersSearchDto": func() any { return new(people.UsersSearchInput) },
		"PeopleUsersUpsertDto": func() any { return new(people.UsersUpsertInput) },
		"PersonUsersDeleteDto": func() any { return new(people.UsersDeleteInput) },
	}
	tests := []struct {
		name string
		dto  string
		mode string
		body string
		want string
	}{
		{name: "PersonSearchDto empty", dto: "PersonSearchDto", mode: "query", body: `{}`, want: ``},
		{name: "PersonSearchDto all", dto: "PersonSearchDto", mode: "query", body: `{"withHidden":"true","closestPersonId":"0f0e0d0c-0b0a-4908-8706-050403020100","closestAssetId":"0f0e0d0c-0b0a-4908-8706-050403020100","page":"2","size":"10","sharedById":"019a1234-5678-7abc-8def-0123456789ab","sharedWithId":"0f0e0d0c-0b0a-4908-8706-050403020100","isFavorite":"false","isHidden":"true","name":"a"}`, want: ``},
		{name: "PersonSearchDto bad bools", dto: "PersonSearchDto", mode: "query", body: `{"withHidden":"yes","isFavorite":"1"}`, want: `[{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["withHidden"],"message":"Invalid option: expected one of \"true\"|\"false\""},{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["isFavorite"],"message":"Invalid option: expected one of \"true\"|\"false\""}]`},
		{name: "PersonSearchDto page zero", dto: "PersonSearchDto", mode: "query", body: `{"page":"0"}`, want: `[{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["page"],"message":"Too small: expected number to be >=1"}]`},
		{name: "PersonSearchDto page text", dto: "PersonSearchDto", mode: "query", body: `{"page":"x"}`, want: `[{"expected":"number","code":"invalid_type","received":"NaN","path":["page"],"message":"Invalid input: expected number, received NaN"}]`},
		{name: "PersonSearchDto page fraction", dto: "PersonSearchDto", mode: "query", body: `{"page":"1.5"}`, want: `[{"expected":"int","format":"safeint","code":"invalid_type","path":["page"],"message":"Invalid input: expected int, received number"}]`},
		{name: "PersonSearchDto size big", dto: "PersonSearchDto", mode: "query", body: `{"size":"1001"}`, want: `[{"origin":"number","code":"too_big","maximum":1000,"inclusive":true,"path":["size"],"message":"Too big: expected number to be <=1000"}]`},
		{name: "PersonSearchDto size empty", dto: "PersonSearchDto", mode: "query", body: `{"size":""}`, want: `[{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["size"],"message":"Too small: expected number to be >=1"}]`},
		{name: "PersonSearchDto closest bad", dto: "PersonSearchDto", mode: "query", body: `{"closestPersonId":"abc","closestAssetId":"019a1234-5678-7abc-8def-0123456789ab"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["closestPersonId"],"message":"Invalid UUID"},{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["closestAssetId"],"message":"Invalid UUID"}]`},
		{name: "PersonSearchDto shared bad", dto: "PersonSearchDto", mode: "query", body: `{"sharedById":"abc","sharedWithId":"00000000-0000-0000-0000-000000000000"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$/","path":["sharedById"],"message":"Invalid UUID"}]`},
		{name: "PersonSearchDto name array", dto: "PersonSearchDto", mode: "query", body: `{"name":["a","b"]}`, want: `[{"expected":"string","code":"invalid_type","path":["name"],"message":"Invalid input: expected string, received array"}]`},
		{name: "PersonCreateDto empty", dto: "PersonCreateDto", mode: "body", body: `{}`, want: ``},
		{name: "PersonCreateDto all", dto: "PersonCreateDto", mode: "body", body: `{"name":"Ann","birthDate":"1985-12-24","isHidden":false,"isFavorite":true,"color":"abc"}`, want: ``},
		{name: "PersonCreateDto future birth", dto: "PersonCreateDto", mode: "body", body: `{"birthDate":"2999-01-01"}`, want: `[{"code":"custom","path":["birthDate"],"message":"Birth date cannot be in the future"}]`},
		{name: "PersonCreateDto bad birth", dto: "PersonCreateDto", mode: "body", body: `{"birthDate":"abc"}`, want: `[{"code":"custom","path":["birthDate"],"message":"Birth date cannot be in the future"}]`},
		{name: "PersonCreateDto loose births", dto: "PersonCreateDto", mode: "body", body: `{"birthDate":"1990-02-30"}`, want: ``},
		{name: "PersonCreateDto null birth", dto: "PersonCreateDto", mode: "body", body: `{"birthDate":null,"color":null}`, want: ``},
		{name: "PersonCreateDto empty birth", dto: "PersonCreateDto", mode: "body", body: `{"birthDate":""}`, want: ``},
		{name: "PersonCreateDto number birth", dto: "PersonCreateDto", mode: "body", body: `{"birthDate":1990}`, want: `[{"expected":"string","code":"invalid_type","path":["birthDate"],"message":"Invalid input: expected string, received number"}]`},
		{name: "PersonCreateDto bad color", dto: "PersonCreateDto", mode: "body", body: `{"color":"#12"}`, want: `[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^#?([0-9A-Fa-f]{3}|[0-9A-Fa-f]{4}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$/","path":["color"],"message":"Invalid string: must match pattern /^#?([0-9A-Fa-f]{3}|[0-9A-Fa-f]{4}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$/"}]`},
		{name: "PersonCreateDto bad types", dto: "PersonCreateDto", mode: "body", body: `{"name":1,"isHidden":"true","isFavorite":null}`, want: `[{"expected":"string","code":"invalid_type","path":["name"],"message":"Invalid input: expected string, received number"},{"expected":"boolean","code":"invalid_type","path":["isHidden"],"message":"Invalid input: expected boolean, received string"},{"expected":"boolean","code":"invalid_type","path":["isFavorite"],"message":"Invalid input: expected boolean, received null"}]`},
		{name: "PersonCreateDto body array", dto: "PersonCreateDto", mode: "body", body: `[]`, want: `[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received array"}]`},
		{name: "PersonUpdateDto empty", dto: "PersonUpdateDto", mode: "body", body: `{}`, want: `[{"code":"custom","path":[],"message":"At least one of the following fields is required: name, birthDate, isHidden, isFavorite, color, featureFaceAssetId"}]`},
		{name: "PersonUpdateDto userId only", dto: "PersonUpdateDto", mode: "body", body: `{"userId":"0f0e0d0c-0b0a-4908-8706-050403020100"}`, want: `[{"code":"custom","path":[],"message":"At least one of the following fields is required: name, birthDate, isHidden, isFavorite, color, featureFaceAssetId"}]`},
		{name: "PersonUpdateDto name", dto: "PersonUpdateDto", mode: "body", body: `{"name":"x"}`, want: ``},
		{name: "PersonUpdateDto null color", dto: "PersonUpdateDto", mode: "body", body: `{"color":null}`, want: ``},
		{name: "PersonUpdateDto feature v7", dto: "PersonUpdateDto", mode: "body", body: `{"featureFaceAssetId":"019a1234-5678-7abc-8def-0123456789ab"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["featureFaceAssetId"],"message":"Invalid UUID"}]`},
		{name: "PersonUpdateDto userId v7", dto: "PersonUpdateDto", mode: "body", body: `{"userId":"019a1234-5678-7abc-8def-0123456789ab","name":"x"}`, want: ``},
		{name: "PersonUpdateDto userId bad", dto: "PersonUpdateDto", mode: "body", body: `{"userId":"abc","name":"x"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$/","path":["userId"],"message":"Invalid UUID"}]`},
		{name: "PersonUpdateDto bad and empty", dto: "PersonUpdateDto", mode: "body", body: `{"isHidden":"x"}`, want: `[{"expected":"boolean","code":"invalid_type","path":["isHidden"],"message":"Invalid input: expected boolean, received string"}]`},
		{name: "PersonUpdateDto unknown only", dto: "PersonUpdateDto", mode: "body", body: `{"foo":1}`, want: `[{"code":"custom","path":[],"message":"At least one of the following fields is required: name, birthDate, isHidden, isFavorite, color, featureFaceAssetId"}]`},
		{name: "PeopleUpdateDto empty", dto: "PeopleUpdateDto", mode: "body", body: `{}`, want: `[{"expected":"array","code":"invalid_type","path":["people"],"message":"Invalid input: expected array, received undefined"}]`},
		{name: "PeopleUpdateDto no people", dto: "PeopleUpdateDto", mode: "body", body: `{"people":[]}`, want: ``},
		{name: "PeopleUpdateDto item ok", dto: "PeopleUpdateDto", mode: "body", body: `{"people":[{"id":"0f0e0d0c-0b0a-4908-8706-050403020100","name":"x"}]}`, want: ``},
		{name: "PeopleUpdateDto item without fields", dto: "PeopleUpdateDto", mode: "body", body: `{"people":[{"id":"0f0e0d0c-0b0a-4908-8706-050403020100"}]}`, want: ``},
		{name: "PeopleUpdateDto item without id", dto: "PeopleUpdateDto", mode: "body", body: `{"people":[{"name":"x"}]}`, want: `[{"expected":"string","code":"invalid_type","path":["people",0,"id"],"message":"Invalid input: expected string, received undefined"}]`},
		{name: "PeopleUpdateDto item bad id", dto: "PeopleUpdateDto", mode: "body", body: `{"people":[{"id":"019a1234-5678-7abc-8def-0123456789ab","isHidden":true}]}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["people",0,"id"],"message":"Invalid UUID"}]`},
		{name: "MergePersonDto empty", dto: "MergePersonDto", mode: "body", body: `{}`, want: `[{"expected":"array","code":"invalid_type","path":["ids"],"message":"Invalid input: expected array, received undefined"}]`},
		{name: "MergePersonDto ids", dto: "MergePersonDto", mode: "body", body: `{"ids":["0f0e0d0c-0b0a-4908-8706-050403020100","1f0e0d0c-0b0a-4908-8706-050403020100"]}`, want: ``},
		{name: "MergePersonDto bad id", dto: "MergePersonDto", mode: "body", body: `{"ids":["abc"]}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["ids",0],"message":"Invalid UUID"}]`},
		{name: "MergePersonDto no ids", dto: "MergePersonDto", mode: "body", body: `{"ids":[]}`, want: ``},
		{name: "PersonDeleteDto empty", dto: "PersonDeleteDto", mode: "body", body: `{}`, want: ``},
		{name: "PersonDeleteDto userId", dto: "PersonDeleteDto", mode: "body", body: `{"userId":"abc"}`, want: ``},
		{name: "PersonDeleteDto userId number", dto: "PersonDeleteDto", mode: "body", body: `{"userId":1}`, want: `[{"expected":"string","code":"invalid_type","path":["userId"],"message":"Invalid input: expected string, received number"}]`},
		{name: "PeopleDeleteDto empty", dto: "PeopleDeleteDto", mode: "body", body: `{}`, want: `[{"expected":"array","code":"invalid_type","path":["ids"],"message":"Invalid input: expected array, received undefined"}]`},
		{name: "PeopleDeleteDto ids", dto: "PeopleDeleteDto", mode: "body", body: `{"ids":["0f0e0d0c-0b0a-4908-8706-050403020100"],"userId":"x"}`, want: ``},
		{name: "PeopleDeleteDto bad ids", dto: "PeopleDeleteDto", mode: "body", body: `{"ids":["x"],"userId":2}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["ids",0],"message":"Invalid UUID"},{"expected":"string","code":"invalid_type","path":["userId"],"message":"Invalid input: expected string, received number"}]`},
		{name: "AssetFaceUpdateDto empty", dto: "AssetFaceUpdateDto", mode: "body", body: `{}`, want: `[{"expected":"array","code":"invalid_type","path":["data"],"message":"Invalid input: expected array, received undefined"}]`},
		{name: "AssetFaceUpdateDto ok", dto: "AssetFaceUpdateDto", mode: "body", body: `{"data":[{"personId":"0f0e0d0c-0b0a-4908-8706-050403020100","assetId":"0f0e0d0c-0b0a-4908-8706-050403020100","userId":"1f0e0d0c-0b0a-4908-8706-050403020100"}]}`, want: ``},
		{name: "AssetFaceUpdateDto bad item", dto: "AssetFaceUpdateDto", mode: "body", body: `{"data":[{"personId":"x","userId":"019a1234-5678-7abc-8def-0123456789ab"}]}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["data",0,"personId"],"message":"Invalid UUID"},{"expected":"string","code":"invalid_type","path":["data",0,"assetId"],"message":"Invalid input: expected string, received undefined"},{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["data",0,"userId"],"message":"Invalid UUID"}]`},
		{name: "FaceDto empty", dto: "FaceDto", mode: "query", body: `{}`, want: `[{"expected":"string","code":"invalid_type","path":["id"],"message":"Invalid input: expected string, received undefined"}]`},
		{name: "FaceDto bad", dto: "FaceDto", mode: "query", body: `{"id":"abc"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["id"],"message":"Invalid UUID"}]`},
		{name: "FaceDto ok", dto: "FaceDto", mode: "query", body: `{"id":"0f0e0d0c-0b0a-4908-8706-050403020100"}`, want: ``},
		{name: "FaceDto body number", dto: "FaceDto", mode: "body", body: `{"id":1}`, want: `[{"expected":"string","code":"invalid_type","path":["id"],"message":"Invalid input: expected string, received number"}]`},
		{name: "AssetFaceCreateDto empty", dto: "AssetFaceCreateDto", mode: "body", body: `{}`, want: `[{"expected":"string","code":"invalid_type","path":["personId"],"message":"Invalid input: expected string, received undefined"},{"expected":"string","code":"invalid_type","path":["assetId"],"message":"Invalid input: expected string, received undefined"},{"expected":"number","code":"invalid_type","path":["imageWidth"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["imageHeight"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["width"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}]`},
		{name: "AssetFaceCreateDto ok", dto: "AssetFaceCreateDto", mode: "body", body: `{"personId":"0f0e0d0c-0b0a-4908-8706-050403020100","assetId":"0f0e0d0c-0b0a-4908-8706-050403020100","imageWidth":10,"imageHeight":10,"x":0,"y":0,"width":5,"height":5}`, want: ``},
		{name: "AssetFaceCreateDto floats", dto: "AssetFaceCreateDto", mode: "body", body: `{"personId":"0f0e0d0c-0b0a-4908-8706-050403020100","assetId":"0f0e0d0c-0b0a-4908-8706-050403020100","imageWidth":1.5,"imageHeight":"10","x":-1,"y":null,"width":5,"height":5}`, want: `[{"expected":"int","format":"safeint","code":"invalid_type","path":["imageWidth"],"message":"Invalid input: expected int, received number"},{"expected":"number","code":"invalid_type","path":["imageHeight"],"message":"Invalid input: expected number, received string"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received null"}]`},
		{name: "AssetFaceDeleteDto empty", dto: "AssetFaceDeleteDto", mode: "body", body: `{}`, want: `[{"expected":"boolean","code":"invalid_type","path":["force"],"message":"Invalid input: expected boolean, received undefined"}]`},
		{name: "AssetFaceDeleteDto ok", dto: "AssetFaceDeleteDto", mode: "body", body: `{"force":false}`, want: ``},
		{name: "AssetFaceDeleteDto string", dto: "AssetFaceDeleteDto", mode: "body", body: `{"force":"true"}`, want: `[{"expected":"boolean","code":"invalid_type","path":["force"],"message":"Invalid input: expected boolean, received string"}]`},
		{name: "PersonUsersSearchDto empty", dto: "PersonUsersSearchDto", mode: "query", body: `{}`, want: ``},
		{name: "PersonUsersSearchDto all", dto: "PersonUsersSearchDto", mode: "query", body: `{"personId":"019a1234-5678-7abc-8def-0123456789ab","direction":"shared-by","sharedById":"0f0e0d0c-0b0a-4908-8706-050403020100","sharedWithId":"0f0e0d0c-0b0a-4908-8706-050403020100","role":"admin"}`, want: ``},
		{name: "PersonUsersSearchDto bad", dto: "PersonUsersSearchDto", mode: "query", body: `{"personId":"x","direction":"up","role":"owner"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$/","path":["personId"],"message":"Invalid UUID"},{"code":"invalid_value","values":["shared-by","shared-with"],"path":["direction"],"message":"Invalid option: expected one of \"shared-by\"|\"shared-with\""},{"code":"invalid_value","values":["read","write","admin"],"path":["role"],"message":"Invalid option: expected one of \"read\"|\"write\"|\"admin\""}]`},
		{name: "PeopleUsersUpsertDto empty", dto: "PeopleUsersUpsertDto", mode: "body", body: `{}`, want: `[{"expected":"array","code":"invalid_type","path":["sharedWithIds"],"message":"Invalid input: expected array, received undefined"},{"code":"invalid_value","values":["read","write","admin"],"path":["role"],"message":"Invalid option: expected one of \"read\"|\"write\"|\"admin\""}]`},
		{name: "PeopleUsersUpsertDto ok ids", dto: "PeopleUsersUpsertDto", mode: "body", body: `{"personIds":["0f0e0d0c-0b0a-4908-8706-050403020100"],"sharedWithIds":["1f0e0d0c-0b0a-4908-8706-050403020100"],"role":"read"}`, want: ``},
		{name: "PeopleUsersUpsertDto ok type", dto: "PeopleUsersUpsertDto", mode: "body", body: `{"type":"everyone","sharedWithIds":[],"role":"write"}`, want: ``},
		{name: "PeopleUsersUpsertDto neither", dto: "PeopleUsersUpsertDto", mode: "body", body: `{"sharedWithIds":["0f0e0d0c-0b0a-4908-8706-050403020100"],"role":"read"}`, want: `[{"code":"custom","path":["personIds"],"message":"Either personIds or type must be provided"}]`},
		{name: "PeopleUsersUpsertDto dupes", dto: "PeopleUsersUpsertDto", mode: "body", body: `{"personIds":["0f0e0d0c-0b0a-4908-8706-050403020100","0f0e0d0c-0b0a-4908-8706-050403020100"],"sharedWithIds":["1f0e0d0c-0b0a-4908-8706-050403020100","1f0e0d0c-0b0a-4908-8706-050403020100"],"role":"read"}`, want: `[{"code":"custom","path":["personIds"],"message":"Items must be unique"},{"code":"custom","path":["sharedWithIds"],"message":"Items must be unique"}]`},
		{name: "PeopleUsersUpsertDto dupes and bad", dto: "PeopleUsersUpsertDto", mode: "body", body: `{"personIds":["x","x"],"sharedWithIds":["1f0e0d0c-0b0a-4908-8706-050403020100"],"role":"read"}`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$/","path":["personIds",0],"message":"Invalid UUID"},{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$/","path":["personIds",1],"message":"Invalid UUID"},{"code":"custom","path":["personIds"],"message":"Items must be unique"}]`},
		{name: "PeopleUsersUpsertDto bad role", dto: "PeopleUsersUpsertDto", mode: "body", body: `{"type":"all","sharedWithIds":["1f0e0d0c-0b0a-4908-8706-050403020100"],"role":"owner"}`, want: `[{"code":"invalid_value","values":["everyone"],"path":["type"],"message":"Invalid input: expected \"everyone\""},{"code":"invalid_value","values":["read","write","admin"],"path":["role"],"message":"Invalid option: expected one of \"read\"|\"write\"|\"admin\""}]`},
		{name: "PeopleUsersUpsertDto null type", dto: "PeopleUsersUpsertDto", mode: "body", body: `{"type":null,"sharedWithIds":["1f0e0d0c-0b0a-4908-8706-050403020100"],"role":"read"}`, want: `[{"code":"invalid_value","values":["everyone"],"path":["type"],"message":"Invalid input: expected \"everyone\""}]`},
		{name: "PersonUsersDeleteDto empty object", dto: "PersonUsersDeleteDto", mode: "body", body: `{}`, want: `[{"expected":"array","code":"invalid_type","path":[],"message":"Invalid input: expected array, received object"}]`},
		{name: "PersonUsersDeleteDto empty array", dto: "PersonUsersDeleteDto", mode: "body", body: `[]`, want: ``},
		{name: "PersonUsersDeleteDto ok", dto: "PersonUsersDeleteDto", mode: "body", body: `[{"personId":"0f0e0d0c-0b0a-4908-8706-050403020100","sharedWithId":"019a1234-5678-7abc-8def-0123456789ab","sharedById":"1f0e0d0c-0b0a-4908-8706-050403020100"}]`, want: ``},
		{name: "PersonUsersDeleteDto bad", dto: "PersonUsersDeleteDto", mode: "body", body: `[{"personId":"x"}]`, want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$/","path":[0,"personId"],"message":"Invalid UUID"},{"expected":"string","code":"invalid_type","path":[0,"sharedWithId"],"message":"Invalid input: expected string, received undefined"}]`},
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

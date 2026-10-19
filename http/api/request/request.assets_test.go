package request

import (
	"maps"
	"testing"

	"tty.fail/mrus/korda/logic/root/assets"
)

const datetimeSource = `/^(?:(?:\\d\\d[2468][048]|\\d\\d[13579][26]|\\d\\d0[48]|[02468][048]00|[13579][26]00)-02-29|\\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\\d|30)|(?:02)-(?:0[1-9]|1\\d|2[0-8])))T(?:(?:[01]\\d|2[0-3]):[0-5]\\d(?::[0-5]\\d(?:\\.\\d+)?)?(?:Z|([+-](?:[01]\\d|2[0-3]):[0-5]\\d)))$/`

func TestUploadFieldsMatchImmich(t *testing.T) {
	ok := map[string]any{"fileCreatedAt": "2024-01-15T10:30:00Z", "fileModifiedAt": "2024-01-15T10:30:00Z"}
	with := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		maps.Copy(out, ok)
		maps.Copy(out, extra)
		return out
	}

	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{
			name: "missing dates",
			raw:  map[string]any{},
			want: `[{"expected":"string","code":"invalid_type","path":["fileCreatedAt"],"message":"Invalid input: expected ISO 8601 datetime string, received undefined"},{"expected":"string","code":"invalid_type","path":["fileModifiedAt"],"message":"Invalid input: expected ISO 8601 datetime string, received undefined"}]`,
		},
		{
			name: "dates without offset",
			raw:  map[string]any{"fileCreatedAt": "2024-01-15T10:30:00", "fileModifiedAt": "2024-01-15"},
			want: `[{"origin":"string","code":"invalid_format","format":"datetime","pattern":"` + datetimeSource + `","path":["fileCreatedAt"],"message":"Invalid input: expected ISO 8601 datetime string, received string"},{"origin":"string","code":"invalid_format","format":"datetime","pattern":"` + datetimeSource + `","path":["fileModifiedAt"],"message":"Invalid input: expected ISO 8601 datetime string, received string"}]`,
		},
		{
			name: "bad isFavorite",
			raw:  with(map[string]any{"isFavorite": "TRUE"}),
			want: `[{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["isFavorite"],"message":"Invalid option: expected one of \"true\"|\"false\""}]`,
		},
		{
			name: "bad duration",
			raw:  with(map[string]any{"duration": "abc"}),
			want: `[{"expected":"number","code":"invalid_type","received":"NaN","path":["duration"],"message":"Invalid input: expected number, received NaN"}]`,
		},
		{
			name: "fractional duration",
			raw:  with(map[string]any{"duration": "1.5"}),
			want: `[{"expected":"int","format":"safeint","code":"invalid_type","path":["duration"],"message":"Invalid input: expected int, received number"}]`,
		},
		{
			name: "bad metadata json",
			raw:  with(map[string]any{"metadata": "{nope"}),
			want: `[{"code":"custom","message":"Invalid input: expected JSON string, received string","path":["metadata"]}]`,
		},
		{
			name: "metadata not an array",
			raw:  with(map[string]any{"metadata": `{"key":"a","value":{}}`}),
			want: `[{"expected":"array","code":"invalid_type","path":["metadata"],"message":"Invalid input: expected array, received object"}]`,
		},
		{
			name: "v7 live photo and bad visibility",
			raw:  with(map[string]any{"livePhotoVideoId": "019a0000-0000-7000-8000-000000000000", "visibility": "public"}),
			want: `[{"code":"invalid_value","values":["archive","timeline","hidden","locked"],"path":["visibility"],"message":"Invalid option: expected one of \"archive\"|\"timeline\"|\"hidden\"|\"locked\""},{"origin":"string","code":"invalid_format","format":"uuid","pattern":"` + uuidSource + `","path":["livePhotoVideoId"],"message":"Invalid UUID"}]`,
		},
		{
			name: "repeated isFavorite",
			raw:  with(map[string]any{"isFavorite": []any{"true", "true"}}),
			want: `[{"expected":"string","code":"invalid_type","path":["isFavorite"],"message":"Invalid input: expected string, received array"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst assets.UploadInput
			if got := issuesOf(t, bind(tt.raw, true, &dst, modeQuery)); got != tt.want {
				t.Errorf("issues\n got %s\nwant %s", got, tt.want)
			}
		})
	}

	var full assets.UploadInput
	err := bind(map[string]any{
		"deviceAssetId": "abc", "filename": "IMG_1.HEIC",
		"fileCreatedAt": "2024-01-15T10:30+02:00", "fileModifiedAt": "2024-01-15T10:30:00.123456Z",
		"isFavorite": "true", "duration": "1500", "visibility": "hidden",
		"livePhotoVideoId": "0f0e0d0c-0b0a-4908-8706-050403020100",
		"metadata":         `[{"key":"mobile-app","value":{"cloudId":"x","b":1}}]`,
	}, true, &full, modeQuery)
	if err != nil {
		t.Fatal(err)
	}
	if got := full.FileCreatedAt.Format("2006-01-02T15:04:05.000Z07:00"); got != "2024-01-15T08:30:00.000Z" {
		t.Errorf("fileCreatedAt = %s", got)
	}
	if got := full.FileModifiedAt.Format("2006-01-02T15:04:05.000Z07:00"); got != "2024-01-15T10:30:00.123Z" {
		t.Errorf("fileModifiedAt = %s", got)
	}
	if full.Duration == nil || *full.Duration != 1500 || full.IsFavorite == nil || !*full.IsFavorite {
		t.Errorf("upload = %+v", full)
	}
	if len(full.Metadata) != 1 || full.Metadata[0].Key != "mobile-app" || string(full.Metadata[0].Value) != `{"cloudId":"x","b":1}` {
		t.Errorf("metadata = %+v", full.Metadata)
	}

	var empty assets.UploadInput
	if err = bind(with(map[string]any{"duration": ""}), true, &empty, modeQuery); err != nil || empty.Duration == nil || *empty.Duration != 0 {
		t.Errorf("empty duration = %v, %v; want 0", empty.Duration, err)
	}
}

func TestAssetBodiesMatchImmich(t *testing.T) {
	tests := []struct {
		name string
		body string
		dst  func() any
		want string
	}{
		{
			name: "update lat only",
			body: `{"latitude":1}`,
			dst:  func() any { return new(assets.UpdateInput) },
			want: `[{"code":"custom","path":[],"message":"Latitude and longitude must be provided together"}]`,
		},
		{
			name: "update rating 0",
			body: `{"rating":0}`,
			dst:  func() any { return new(assets.UpdateInput) },
			want: `[{"code":"custom","path":["rating"],"message":"Rating must be -1 (rejected), 1–5 (starred), or null (unrated); 0 is not valid"}]`,
		},
		{
			name: "update rating 0 and lat only",
			body: `{"rating":0,"latitude":1}`,
			dst:  func() any { return new(assets.UpdateInput) },
			want: `[{"code":"custom","path":["rating"],"message":"Rating must be -1 (rejected), 1–5 (starred), or null (unrated); 0 is not valid"},{"code":"custom","path":[],"message":"Latitude and longitude must be provided together"}]`,
		},
		{
			name: "update rating 6",
			body: `{"rating":6}`,
			dst:  func() any { return new(assets.UpdateInput) },
			want: `[{"origin":"number","code":"too_big","maximum":5,"inclusive":true,"path":["rating"],"message":"Too big: expected number to be <=5"}]`,
		},
		{
			name: "update rating -2",
			body: `{"rating":-2}`,
			dst:  func() any { return new(assets.UpdateInput) },
			want: `[{"origin":"number","code":"too_small","minimum":-1,"inclusive":true,"path":["rating"],"message":"Too small: expected number to be >=-1"}]`,
		},
		{
			name: "update lat 91",
			body: `{"latitude":91,"longitude":0}`,
			dst:  func() any { return new(assets.UpdateInput) },
			want: `[{"origin":"number","code":"too_big","maximum":90,"inclusive":true,"path":["latitude"],"message":"Too big: expected number to be <=90"}]`,
		},
		{
			name: "update bad type and lat only",
			body: `{"isFavorite":"yes","latitude":1}`,
			dst:  func() any { return new(assets.UpdateInput) },
			want: `[{"expected":"boolean","code":"invalid_type","path":["isFavorite"],"message":"Invalid input: expected boolean, received string"}]`,
		},
		{
			name: "bulk update sibling",
			body: `{"ids":["0f0e0d0c-0b0a-4908-8706-050403020100"],"dateTimeRelative":60,"dateTimeOriginal":"2024-01-01T00:00:00Z"}`,
			dst:  func() any { return new(assets.BulkUpdateInput) },
			want: `[{"code":"custom","path":[],"message":"dateTimeRelative cannot exist alongside dateTimeOriginal"}]`,
		},
		{
			name: "bulk update sibling and lat only",
			body: `{"ids":["0f0e0d0c-0b0a-4908-8706-050403020100"],"dateTimeRelative":60,"dateTimeOriginal":"x","latitude":1}`,
			dst:  func() any { return new(assets.BulkUpdateInput) },
			want: `[{"code":"custom","path":[],"message":"Latitude and longitude must be provided together"}]`,
		},
		{
			name: "bulk update missing ids",
			body: `{"isFavorite":true}`,
			dst:  func() any { return new(assets.BulkUpdateInput) },
			want: `[{"expected":"array","code":"invalid_type","path":["ids"],"message":"Invalid input: expected array, received undefined"}]`,
		},
		{
			name: "bulk update v7 id",
			body: `{"ids":["019a0000-0000-7000-8000-000000000000"]}`,
			dst:  func() any { return new(assets.BulkUpdateInput) },
			want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"` + uuidSource + `","path":["ids",0],"message":"Invalid UUID"}]`,
		},
		{
			name: "bulk delete string ids",
			body: `{"ids":"x"}`,
			dst:  func() any { return new(assets.BulkDeleteInput) },
			want: `[{"expected":"array","code":"invalid_type","path":["ids"],"message":"Invalid input: expected array, received string"}]`,
		},
		{
			name: "jobs bad name",
			body: `{"assetIds":["0f0e0d0c-0b0a-4908-8706-050403020100"],"name":"nope"}`,
			dst:  func() any { return new(assets.JobsInput) },
			want: `[{"code":"invalid_value","values":["refresh-faces","refresh-metadata","regenerate-thumbnail","transcode-video"],"path":["name"],"message":"Invalid option: expected one of \"refresh-faces\"|\"refresh-metadata\"|\"regenerate-thumbnail\"|\"transcode-video\""}]`,
		},
		{
			name: "metadata value array",
			body: `{"items":[{"key":"k","value":[1]}]}`,
			dst:  func() any { return new(assets.MetadataUpsertInput) },
			want: `[{"expected":"record","code":"invalid_type","path":["items",0,"value"],"message":"Invalid input: expected record, received array"}]`,
		},
		{
			name: "metadata bulk delete without key",
			body: `{"items":[{"assetId":"0f0e0d0c-0b0a-4908-8706-050403020100"}]}`,
			dst:  func() any { return new(assets.MetadataBulkDeleteInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["items",0,"key"],"message":"Invalid input: expected string, received undefined"}]`,
		},
		{
			name: "copy null favorite",
			body: `{"sourceId":"0f0e0d0c-0b0a-4908-8706-050403020100","targetId":"0f0e0d0c-0b0a-4908-8706-050403020100","albums":false,"favorite":null}`,
			dst:  func() any { return new(assets.CopyInput) },
			want: `[{"expected":"boolean","code":"invalid_type","path":["favorite"],"message":"Invalid input: expected boolean, received null"}]`,
		},
		{
			name: "bulk check",
			body: `{"assets":[{"id":1}]}`,
			dst:  func() any { return new(assets.BulkUploadCheckInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["assets",0,"id"],"message":"Invalid input: expected string, received number"},{"expected":"string","code":"invalid_type","path":["assets",0,"checksum"],"message":"Invalid input: expected string, received undefined"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := issuesOf(t, bindJSON(t, tt.body, tt.dst())); got != tt.want {
				t.Errorf("issues\n got %s\nwant %s", got, tt.want)
			}
		})
	}

	var update assets.UpdateInput
	if err := bindJSON(t, `{"isFavorite":true,"visibility":"archive","dateTimeOriginal":"2024-01-15T10:30:00+02:00","latitude":1.5,"longitude":2.5,"rating":null,"description":"x","livePhotoVideoId":null,"extra":1}`, &update); err != nil {
		t.Fatal(err)
	}
	if !update.Rating.Set || !update.Rating.Null || !update.LivePhotoVideoID.Null || update.Latitude == nil || *update.Latitude != 1.5 {
		t.Errorf("update = %+v", update)
	}

	var media assets.MediaOptionsInput
	err := bind(map[string]any{"size": "large", "edited": "1"}, true, &media, modeQuery)
	want := `[{"code":"invalid_value","values":["original","fullsize","preview","thumbnail"],"path":["size"],"message":"Invalid option: expected one of \"original\"|\"fullsize\"|\"preview\"|\"thumbnail\""},{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["edited"],"message":"Invalid option: expected one of \"true\"|\"false\""}]`
	if got := issuesOf(t, err); got != want {
		t.Errorf("media options\n got %s\nwant %s", got, want)
	}
}

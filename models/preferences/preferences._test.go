package preferences

import (
	"encoding/json"
	"testing"

	"tty.fail/mrus/korda/models/user"
)

const immichDefaults = `{"albums":{"defaultAssetOrder":"desc"},"folders":{"enabled":false,"sidebarWeb":false},"memories":{"enabled":true,"duration":5,"sidebarWeb":false},"people":{"enabled":true,"sidebarWeb":false,"minimumFaces":3,"updateStrategy":"everyone"},"sharedLinks":{"enabled":true,"sidebarWeb":false},"ratings":{"enabled":false},"tags":{"enabled":false,"sidebarWeb":false},"emailNotifications":{"enabled":true,"albumInvite":true,"albumUpdate":true},"download":{"archiveSize":4294967296,"includeEmbeddedVideos":false},"purchase":{"showSupportBadge":true,"hideBuyButtonUntil":"2022-02-12T00:00:00.000Z"},"cast":{"gCastEnabled":false},"recentlyAdded":{"sidebarWeb":false}}`

const immichUpdated = `{"albums":{"defaultAssetOrder":"desc"},"folders":{"enabled":false,"sidebarWeb":false},"memories":{"enabled":true,"duration":7,"sidebarWeb":false},"people":{"enabled":true,"sidebarWeb":false,"minimumFaces":3,"updateStrategy":"everyone"},"sharedLinks":{"enabled":true,"sidebarWeb":false},"ratings":{"enabled":false},"tags":{"enabled":false,"sidebarWeb":false},"emailNotifications":{"enabled":true,"albumInvite":true,"albumUpdate":true},"download":{"archiveSize":1,"includeEmbeddedVideos":false},"purchase":{"showSupportBadge":true,"hideBuyButtonUntil":"2022-02-12T00:00:00.000Z"},"cast":{"gCastEnabled":false},"recentlyAdded":{"sidebarWeb":false},"avatar":{"color":"red"}}`

func encode(t *testing.T, v any) string {
	t.Helper()

	encoded, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

func TestDefaultsAreImmichsBodyByteForByte(t *testing.T) {
	if got := encode(t, Defaults()); got != immichDefaults {
		t.Errorf("Defaults =\n%s\nwant\n%s", got, immichDefaults)
	}
}

func TestMergeMatchesImmichsUpdateResponse(t *testing.T) {
	var u Update
	if err := json.Unmarshal([]byte(`{"memories":{"duration":7},"avatar":{"color":"red"},"download":{"archiveSize":1}}`), &u); err != nil {
		t.Fatal(err)
	}

	merged := Merge(Defaults(), u)
	if got := encode(t, merged); got != immichUpdated {
		t.Errorf("Merge =\n%s\nwant\n%s", got, immichUpdated)
	}

	stored, err := Stored(merged)
	if err != nil {
		t.Fatal(err)
	}
	if stored != `{"download":{"archiveSize":1},"memories":{"duration":7}}` {
		t.Errorf("Stored = %s, want the two changed leaves and no avatar", stored)
	}

	read, err := FromStored(stored)
	if err != nil {
		t.Fatal(err)
	}
	if read.Avatar != nil || read.Memories.Duration != 7 || read.Download.ArchiveSize != 1 {
		t.Errorf("FromStored = %+v", read)
	}
}

func TestStoredLeavesOutDefaultsAndEmptyStrings(t *testing.T) {
	p := Defaults()
	p.Memories.Enabled = true
	p.Purchase.HideBuyButtonUntil = ""
	color := user.ColorRed
	p.Avatar = &Avatar{Color: color}

	stored, err := Stored(p)
	if err != nil || stored != `{}` {
		t.Errorf("Stored = %s, %v, want {}", stored, err)
	}

	if read, err := FromStored(""); err != nil || encode(t, read) != immichDefaults {
		t.Errorf("FromStored of nothing = %+v, %v", read, err)
	}
}

func TestMergeOfAnEmptyGroupChangesNothing(t *testing.T) {
	var u Update
	if err := json.Unmarshal([]byte(`{"avatar":{},"memories":{}}`), &u); err != nil {
		t.Fatal(err)
	}

	if got := encode(t, Merge(Defaults(), u)); got != immichDefaults {
		t.Errorf("Merge of empty groups =\n%s", got)
	}
}

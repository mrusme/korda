package request

import (
	"testing"

	"tty.fail/mrus/korda/logic/root/libraries"
)

func TestLibraryBodiesMatchImmich(t *testing.T) {
	newOf := map[string]func() any{
		"CreateLibraryDto":   func() any { return new(libraries.CreateInput) },
		"UpdateLibraryDto":   func() any { return new(libraries.UpdateInput) },
		"ValidateLibraryDto": func() any { return new(libraries.ValidateInput) },
	}
	tests := []struct {
		name string
		dto  string
		mode string
		body string
		want string
	}{
		{name: "CreateLibraryDto empty", dto: "CreateLibraryDto", mode: "body", body: `{}`, want: `[{"expected":"string","code":"invalid_type","path":["ownerId"],"message":"Invalid input: expected string, received undefined"}]`},
		{name: "CreateLibraryDto ok", dto: "CreateLibraryDto", mode: "body", body: `{"ownerId":"0f0e0d0c-0b0a-4908-8706-050403020100","name":"x","importPaths":["/a"],"exclusionPatterns":[]}`, want: ``},
		{name: "CreateLibraryDto empty name", dto: "CreateLibraryDto", mode: "body", body: `{"ownerId":"0f0e0d0c-0b0a-4908-8706-050403020100","name":""}`, want: `[{"origin":"string","code":"too_small","minimum":1,"inclusive":true,"path":["name"],"message":"Too small: expected string to have >=1 characters"}]`},
		{name: "CreateLibraryDto blank and dupe", dto: "CreateLibraryDto", mode: "body", body: `{"ownerId":"0f0e0d0c-0b0a-4908-8706-050403020100","importPaths":[" ","/a","/a"]}`, want: `[{"code":"custom","path":["importPaths"],"message":"Array items must not be empty"},{"code":"custom","path":["importPaths"],"message":"Array must have unique items"}]`},
		{name: "CreateLibraryDto blank only", dto: "CreateLibraryDto", mode: "body", body: `{"ownerId":"0f0e0d0c-0b0a-4908-8706-050403020100","importPaths":["","/b"]}`, want: `[{"code":"custom","path":["importPaths"],"message":"Array items must not be empty"}]`},
		{name: "CreateLibraryDto too many", dto: "CreateLibraryDto", mode: "body", body: `{"ownerId":"0f0e0d0c-0b0a-4908-8706-050403020100","importPaths":["/0","/1","/2","/3","/4","/5","/6","/7","/8","/9","/10","/11","/12","/13","/14","/15","/16","/17","/18","/19","/20","/21","/22","/23","/24","/25","/26","/27","/28","/29","/30","/31","/32","/33","/34","/35","/36","/37","/38","/39","/40","/41","/42","/43","/44","/45","/46","/47","/48","/49","/50","/51","/52","/53","/54","/55","/56","/57","/58","/59","/60","/61","/62","/63","/64","/65","/66","/67","/68","/69","/70","/71","/72","/73","/74","/75","/76","/77","/78","/79","/80","/81","/82","/83","/84","/85","/86","/87","/88","/89","/90","/91","/92","/93","/94","/95","/96","/97","/98","/99","/100","/101","/102","/103","/104","/105","/106","/107","/108","/109","/110","/111","/112","/113","/114","/115","/116","/117","/118","/119","/120","/121","/122","/123","/124","/125","/126","/127","/128"]}`, want: `[{"origin":"array","code":"too_big","maximum":128,"inclusive":true,"path":["importPaths"],"message":"Too big: expected array to have <=128 items"}]`},
		{name: "CreateLibraryDto not strings", dto: "CreateLibraryDto", mode: "body", body: `{"ownerId":"0f0e0d0c-0b0a-4908-8706-050403020100","importPaths":[1,"/a"],"exclusionPatterns":"x"}`, want: `[{"expected":"string","code":"invalid_type","path":["importPaths",0],"message":"Invalid input: expected string, received number"},{"expected":"array","code":"invalid_type","path":["exclusionPatterns"],"message":"Invalid input: expected array, received string"}]`},
		{name: "UpdateLibraryDto empty", dto: "UpdateLibraryDto", mode: "body", body: `{}`, want: ``},
		{name: "UpdateLibraryDto null name", dto: "UpdateLibraryDto", mode: "body", body: `{"name":null}`, want: `[{"expected":"string","code":"invalid_type","path":["name"],"message":"Invalid input: expected string, received null"}]`},
		{name: "ValidateLibraryDto empty", dto: "ValidateLibraryDto", mode: "body", body: `{}`, want: ``},
		{name: "ValidateLibraryDto dupes", dto: "ValidateLibraryDto", mode: "body", body: `{"importPaths":["/a","/a"],"exclusionPatterns":["",""]}`, want: `[{"code":"custom","path":["importPaths"],"message":"Array must have unique items"},{"code":"custom","path":["exclusionPatterns"],"message":"Array items must not be empty"},{"code":"custom","path":["exclusionPatterns"],"message":"Array must have unique items"}]`},
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

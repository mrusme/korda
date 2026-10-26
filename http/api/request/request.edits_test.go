package request

import (
	"encoding/json"
	"testing"

	"tty.fail/mrus/korda/models/assetedit"
)

type editBody struct {
	Edits assetedit.Inputs `json:"edits" v:"min=1"`
}

func TestEditBodiesMatchImmich(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "E01 crop only",
			body: `{"edits":[{"action":"crop","parameters":{"x":10,"y":20,"width":100,"height":80}}]}`,
			want: `{"edits":[{"action":"crop","parameters":{"x":10,"y":20,"width":100,"height":80}}]}`,
		},
		{
			name: "E02 crop, mirror h, rotate 90",
			body: `{"edits":[{"action":"crop","parameters":{"height":80,"width":100,"y":20,"x":10}},{"action":"mirror","parameters":{"axis":"horizontal"}},{"action":"rotate","parameters":{"angle":90}}]}`,
			want: `{"edits":[{"action":"crop","parameters":{"x":10,"y":20,"width":100,"height":80}},{"action":"mirror","parameters":{"axis":"horizontal"}},{"action":"rotate","parameters":{"angle":90}}]}`,
		},
		{
			name: "E03 rotate 0",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":0}}]}`,
			want: `{"edits":[{"action":"rotate","parameters":{"angle":0}}]}`,
		},
		{
			name: "E04 body missing edits",
			body: `{}`,
			want: `[{"expected":"array","code":"invalid_type","path":["edits"],"message":"Invalid input: expected array, received undefined"}]`,
		},
		{
			name: "E05 edits empty",
			body: `{"edits":[]}`,
			want: `[{"origin":"array","code":"too_small","minimum":1,"inclusive":true,"path":["edits"],"message":"Too small: expected array to have >=1 items"}]`,
		},
		{
			name: "E06 edits null",
			body: `{"edits":null}`,
			want: `[{"expected":"array","code":"invalid_type","path":["edits"],"message":"Invalid input: expected array, received null"}]`,
		},
		{
			name: "E07 edits object",
			body: `{"edits":{"action":"rotate","parameters":{"angle":90}}}`,
			want: `[{"expected":"array","code":"invalid_type","path":["edits"],"message":"Invalid input: expected array, received object"}]`,
		},
		{
			name: "E08 body array",
			body: `[{"action":"rotate","parameters":{"angle":90}}]`,
			want: `[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received array"}]`,
		},
		{
			name: "E09 unknown action blur",
			body: `{"edits":[{"action":"blur","parameters":{"angle":90}}]}`,
			want: `[{"code":"invalid_value","values":["crop","rotate","mirror"],"path":["edits",0,"action"],"message":"Invalid option: expected one of \"crop\"|\"rotate\"|\"mirror\""}]`,
		},
		{
			name: "E10 action missing",
			body: `{"edits":[{"parameters":{"angle":90}}]}`,
			want: `[{"code":"invalid_value","values":["crop","rotate","mirror"],"path":["edits",0,"action"],"message":"Invalid option: expected one of \"crop\"|\"rotate\"|\"mirror\""}]`,
		},
		{
			name: "E11 parameters missing",
			body: `{"edits":[{"action":"rotate"}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}],[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}],[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E12 parameters null",
			body: `{"edits":[{"action":"rotate","parameters":null}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received null"}],[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received null"}],[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received null"}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E13 parameters empty object",
			body: `{"edits":[{"action":"crop","parameters":{}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["width"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E14 crop x -1",
			body: `{"edits":[{"action":"crop","parameters":{"x":-1,"y":0,"width":10,"height":10}}]}`,
			want: `[{"origin":"number","code":"too_small","minimum":0,"inclusive":true,"path":["edits",0,"parameters","x"],"message":"Too small: expected number to be >=0"},{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'crop', expecting keys: x, y, width, height"}]`,
		},
		{
			name: "E15 crop width 0",
			body: `{"edits":[{"action":"crop","parameters":{"x":0,"y":0,"width":0,"height":10}}]}`,
			want: `[{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["edits",0,"parameters","width"],"message":"Too small: expected number to be >=1"},{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'crop', expecting keys: x, y, width, height"}]`,
		},
		{
			name: "E16 crop x 1.5",
			body: `{"edits":[{"action":"crop","parameters":{"x":1.5,"y":0,"width":10,"height":10}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"int","format":"safeint","code":"invalid_type","path":["x"],"message":"Invalid input: expected int, received number"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E17 crop x string",
			body: `{"edits":[{"action":"crop","parameters":{"x":"10","y":0,"width":10,"height":10}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received string"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E18 crop height missing",
			body: `{"edits":[{"action":"crop","parameters":{"x":0,"y":0,"width":10}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E19 crop x 1e20",
			body: `{"edits":[{"action":"crop","parameters":{"x":100000000000000000000,"y":0,"width":10,"height":10}}]}`,
			want: `[{"code":"too_big","maximum":9007199254740991,"note":"Integers must be within the safe integer range.","origin":"int","inclusive":true,"path":["edits",0,"parameters","x"],"message":"Too big: expected int to be <=9007199254740991"},{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'crop', expecting keys: x, y, width, height"}]`,
		},
		{
			name: "E20 rotate 45",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":45}}]}`,
			want: `[{"code":"custom","path":["edits",0,"parameters","angle"],"message":"Angle must be one of the following values: 0, 90, 180, 270"},{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'rotate', expecting keys: angle"}]`,
		},
		{
			name: "E21 rotate 360",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":360}}]}`,
			want: `[{"origin":"number","code":"too_big","maximum":270,"inclusive":true,"path":["edits",0,"parameters","angle"],"message":"Too big: expected number to be <=270"},{"code":"custom","path":["edits",0,"parameters","angle"],"message":"Angle must be one of the following values: 0, 90, 180, 270"},{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'rotate', expecting keys: angle"}]`,
		},
		{
			name: "E22 rotate -90",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":-90}}]}`,
			want: `[{"origin":"number","code":"too_small","minimum":0,"inclusive":true,"path":["edits",0,"parameters","angle"],"message":"Too small: expected number to be >=0"},{"code":"custom","path":["edits",0,"parameters","angle"],"message":"Angle must be one of the following values: 0, 90, 180, 270"},{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'rotate', expecting keys: angle"}]`,
		},
		{
			name: "E23 rotate 90.5",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":90.5}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["width"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}],[{"expected":"int","format":"safeint","code":"invalid_type","path":["angle"],"message":"Invalid input: expected int, received number"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E24 rotate string 90",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":"90"}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["width"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received string"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E25 mirror diagonal",
			body: `{"edits":[{"action":"mirror","parameters":{"axis":"diagonal"}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["width"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E26 mirror axis missing",
			body: `{"edits":[{"action":"mirror","parameters":{}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["width"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E27 rotate with crop parameters",
			body: `{"edits":[{"action":"rotate","parameters":{"x":0,"y":0,"width":100,"height":100}}]}`,
			want: `[{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'rotate', expecting keys: angle"}]`,
		},
		{
			name: "E28 crop with rotate parameters",
			body: `{"edits":[{"action":"crop","parameters":{"angle":90}}]}`,
			want: `[{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'crop', expecting keys: x, y, width, height"}]`,
		},
		{
			name: "E29 mirror with rotate parameters",
			body: `{"edits":[{"action":"mirror","parameters":{"angle":90}}]}`,
			want: `[{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'mirror', expecting keys: axis"}]`,
		},
		{
			name: "E30 rotate with angle and crop keys",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":90,"x":0,"y":0,"width":10,"height":10}}]}`,
			want: `[{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'rotate', expecting keys: angle"}]`,
		},
		{
			name: "E31 crop with crop keys and angle",
			body: `{"edits":[{"action":"crop","parameters":{"x":0,"y":0,"width":10,"height":10,"angle":90}}]}`,
			want: `{"edits":[{"action":"crop","parameters":{"x":0,"y":0,"width":10,"height":10}}]}`,
		},
		{
			name: "E32 mirror with axis and angle",
			body: `{"edits":[{"action":"mirror","parameters":{"axis":"vertical","angle":90}}]}`,
			want: `[{"code":"custom","path":["edits",0,"parameters"],"message":"Invalid parameters for action 'mirror', expecting keys: axis"}]`,
		},
		{
			name: "E33 two rotates",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":90}},{"action":"rotate","parameters":{"angle":180}}]}`,
			want: `[{"code":"custom","path":["edits"],"message":"Duplicate edit actions are not allowed"}]`,
		},
		{
			name: "E34 two crops",
			body: `{"edits":[{"action":"crop","parameters":{"x":0,"y":0,"width":10,"height":10}},{"action":"crop","parameters":{"x":1,"y":1,"width":5,"height":5}}]}`,
			want: `[{"code":"custom","path":["edits"],"message":"Duplicate edit actions are not allowed"}]`,
		},
		{
			name: "E35 mirror h and v",
			body: `{"edits":[{"action":"mirror","parameters":{"axis":"horizontal"}},{"action":"mirror","parameters":{"axis":"vertical"}}]}`,
			want: `{"edits":[{"action":"mirror","parameters":{"axis":"horizontal"}},{"action":"mirror","parameters":{"axis":"vertical"}}]}`,
		},
		{
			name: "E36 mirror h twice",
			body: `{"edits":[{"action":"mirror","parameters":{"axis":"horizontal"}},{"action":"mirror","parameters":{"axis":"horizontal"}}]}`,
			want: `[{"code":"custom","path":["edits"],"message":"Duplicate edit actions are not allowed"}]`,
		},
		{
			name: "E37 extra keys everywhere",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":90,"foo":1},"id":"x","bar":2}],"extra":true}`,
			want: `{"edits":[{"action":"rotate","parameters":{"angle":90}}]}`,
		},
		{
			name: "E38 duplicate plus invalid item",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":90}},{"action":"rotate","parameters":{"angle":90}},{"action":"blur","parameters":{}}]}`,
			want: `[{"code":"invalid_value","values":["crop","rotate","mirror"],"path":["edits",2,"action"],"message":"Invalid option: expected one of \"crop\"|\"rotate\"|\"mirror\""},{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["x"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["width"],"message":"Invalid input: expected number, received undefined"},{"expected":"number","code":"invalid_type","path":["height"],"message":"Invalid input: expected number, received undefined"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",2,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E39 duplicate plus bad angle",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":90}},{"action":"rotate","parameters":{"angle":45}}]}`,
			want: `[{"code":"custom","path":["edits",1,"parameters","angle"],"message":"Angle must be one of the following values: 0, 90, 180, 270"},{"code":"custom","path":["edits",1,"parameters"],"message":"Invalid parameters for action 'rotate', expecting keys: angle"},{"code":"custom","path":["edits"],"message":"Duplicate edit actions are not allowed"}]`,
		},
		{
			name: "E40 parameters array",
			body: `{"edits":[{"action":"rotate","parameters":[90]}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received array"}],[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received array"}],[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received array"}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E41 action number",
			body: `{"edits":[{"action":1,"parameters":{"angle":90}}]}`,
			want: `[{"code":"invalid_value","values":["crop","rotate","mirror"],"path":["edits",0,"action"],"message":"Invalid option: expected one of \"crop\"|\"rotate\"|\"mirror\""}]`,
		},
		{
			name: "E42 action uppercase",
			body: `{"edits":[{"action":"Rotate","parameters":{"angle":90}}]}`,
			want: `[{"code":"invalid_value","values":["crop","rotate","mirror"],"path":["edits",0,"action"],"message":"Invalid option: expected one of \"crop\"|\"rotate\"|\"mirror\""}]`,
		},
		{
			name: "E43 rotate 270 then crop",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":270}},{"action":"crop","parameters":{"x":0,"y":0,"width":10,"height":10}}]}`,
			want: `{"edits":[{"action":"rotate","parameters":{"angle":270}},{"action":"crop","parameters":{"x":0,"y":0,"width":10,"height":10}}]}`,
		},
		{
			name: "E44 crop y null",
			body: `{"edits":[{"action":"crop","parameters":{"x":0,"y":null,"width":10,"height":10}}]}`,
			want: `[{"code":"invalid_union","errors":[[{"expected":"number","code":"invalid_type","path":["y"],"message":"Invalid input: expected number, received null"}],[{"expected":"number","code":"invalid_type","path":["angle"],"message":"Invalid input: expected number, received undefined"}],[{"code":"invalid_value","values":["horizontal","vertical"],"path":["axis"],"message":"Invalid option: expected one of \"horizontal\"|\"vertical\""}]],"path":["edits",0,"parameters"],"message":"Invalid parameters for action, expected keys by action: crop: [x, y, width, height]; rotate: [angle]; mirror: [axis]"}]`,
		},
		{
			name: "E45 two items one bad parameters",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":90}},{"action":"crop","parameters":{"angle":90}}]}`,
			want: `[{"code":"custom","path":["edits",1,"parameters"],"message":"Invalid parameters for action 'crop', expecting keys: x, y, width, height"}]`,
		},
		{
			name: "E46 rotate 180 mirror v mirror h",
			body: `{"edits":[{"action":"rotate","parameters":{"angle":180}},{"action":"mirror","parameters":{"axis":"vertical"}},{"action":"mirror","parameters":{"axis":"horizontal"}}]}`,
			want: `{"edits":[{"action":"rotate","parameters":{"angle":180}},{"action":"mirror","parameters":{"axis":"vertical"}},{"action":"mirror","parameters":{"axis":"horizontal"}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst editBody
			err := bindJSON(t, tt.body, &dst)
			if err == nil {
				got, _ := json.Marshal(dst)
				if string(got) != tt.want {
					t.Errorf("parsed %s\n  want %s", got, tt.want)
				}
				return
			}
			if got := issuesOf(t, err); got != tt.want {
				t.Errorf("issues %s\n  want %s", got, tt.want)
			}
		})
	}
}

package gamemap

import "encoding/json"

func Load(fname string) {
	b := []byte(`{"Name":"Wednesday","Age":6,"Parents":["Gomez","Morticia"]}`)
	var f interface{}
	err := json.Unmarshal(b, &f)

	if err != nil {
		return
	}
}

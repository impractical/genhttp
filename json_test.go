package genhttp

import (
	"encoding/json"
)

type jsonEncoder struct{}

func (jsonEncoder) encode(resp Response) ([]byte, error) {
	return json.Marshal(resp)
}

func (jsonEncoder) contentType() string {
	return "application/json"
}

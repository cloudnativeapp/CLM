package http

import (
	"encoding/json"
	"log"
	"testing"
)

func TestDoHTTPProbe(t *testing.T) {
	test := "{\"code\":200,\"msg\":\"\",\"data\":\"Abnormal\",\"success\":true}"
	data := ResponseData{}
	err := json.Unmarshal([]byte(test), &data)
	if err != nil {
		log.Printf("%v", err)
	}
}

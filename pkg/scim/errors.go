package scim

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type ErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
	Detail   string   `json:"detail"`
}

func WriteError(w http.ResponseWriter, statusCode int, scimType, detail string) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(statusCode)

	errPayload := ErrorResponse{
		Schemas:  []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		Status:   strconv.Itoa(statusCode),
		ScimType: scimType,
		Detail:   detail,
	}

	json.NewEncoder(w).Encode(errPayload)
}

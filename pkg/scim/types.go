package scim

import "time"

type User struct {
	ID       string   `json:"id"`
	UserName string   `json:"userName"`
	Active   bool     `json:"active"`
	Groups   []string `json:"groups"` // Holds enterprise group IDs/Names
	Meta     Meta     `json:"meta"`
}

type Agent struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Active      bool     `json:"active"`
	OwnerID     string   `json:"ownerId"`
	OwnerType   string   `json:"ownerType"`
	Scopes      []string `json:"scopes"`
	Meta        Meta     `json:"meta"`
}

type Meta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
}

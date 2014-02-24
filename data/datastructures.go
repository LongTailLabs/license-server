package data

import (
	"fmt"
	"net/http"

	"github.com/martini-contrib/binding"
)

type APIError struct {
	Error   string      `json:error`
	Context interface{} `json:context`
}

// Application Object
type Application struct {
	Id   string `json:"_id" bson:"_id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// Consumer Object
type Consumer struct {
	Id   string `json:"id" bson:"_id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

func (c Consumer) Validate(errors *binding.Errors, req *http.Request) {
	fmt.Println("Patterns for name/key?")
}

// Client Request Object
type Client struct {
	Id string `json:"id" bson:"id" binding:"required"`
}

// Client Count Object
type Counter struct {
	Consumer    string
	Application string
	Counts      map[string]int
}

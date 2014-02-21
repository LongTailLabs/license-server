package data

import (
	"fmt"
	"net"
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
	Id   string `json:"_id" bson:"_id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

func (c Consumer) Validate(errors *binding.Errors, req *http.Request) {
	fmt.Println("Patterns for name/key?")
}

// Client Object
type Client struct {
	Id string `json:"id" bson:"id" binding:"required"`
}

// Restrictions
type CounterType string
type RestrictionType string

var (
	Usage    CounterType     = "usage"
	Access   CounterType     = "access"
	MaxCount RestrictionType = "maxCount"
	NetAddr  RestrictionType = "netAddr"
)

type Restriction struct {
	Consumer     string
	Application  string
	Restrictions []map[string]interface{}
}

type Restrictor interface {
	Validate()
}

type MaxCountRestriction struct {
	Counter string
	Maximum int
}

func (r MaxCountRestriction) Validate() {}

type NetRestriction struct {
	CIDR net.IPNet
}

func (r NetRestriction) Validate() {}

/**
* Give Martini a try, it uses dependency injection and some other
* sorts of strange magic. But overall it seems like the wishful
* simplicity that might be good for this project. Unless.
*
* Keeping DB/Intelligence as separated as possible so it won't be
* hard to rip out Martini (the HTTP part) if it becomes an issue.

TODO: Consider using constants for the collection names.

*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/LongTailLabs/license-server/data"
	"github.com/sigu-399/gojsonschema"

	"github.com/codegangsta/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

func main() {
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(DB())

	setupHandlers(m)

	m.Run()
}

// Meant to be a middleware. This middleware registers a service that is
// to be used as a dependency injected into a handler.
func DB() martini.Handler {
	session, err := mgo.Dial("mongodb://localhost")
	if err != nil {
		panic(err)
	}

	return func(c martini.Context) {
		s := session.Clone()
		db := s.DB("jca-license-server")
		c.Map(db)

		restrictionIndex := mgo.Index{
			Key:    []string{"consumer", "application"},
			Unique: true,
		}
		err := db.C("restrictions").EnsureIndex(restrictionIndex)
		if err != nil {
			panic(err.Error())
		}

		defer s.Close()
		c.Next()
	}
}

// Separating, easy to move later.
func setupHandlers(m *martini.ClassicMartini) {
	m.Get("/", func(r render.Render) {
		r.JSON(200, map[string]interface{}{
			"@": "IRATEMONK 0.1.0",
			"links": []map[string]string{
				{"rel": "consumers", "href": "/consumers"},
				{"rel": "applications", "href": "/applications"},
				{"rel": "restrictions", "href": "/restrictions/:consumer/:product"},
			},
		})
	})

	// CONSUMERS CRUD (-UD)
	m.Get("/consumers", listByType("consumers"))
	m.Get("/consumers/:id", findById("consumers"))
	m.Post("/consumers", binding.Bind(data.Consumer{}),
		func(db *mgo.Database, c data.Consumer, r render.Render) {
			genericIdInsert(db, "consumers", c, r)
		})

	// PRODUCTS CRUD (-UD)
	m.Get("/applications", listByType("applications"))
	m.Get("/application/:id", findById("applications"))
	m.Post("/applications", binding.Bind(data.Application{}),
		func(db *mgo.Database, a data.Application, r render.Render) {
			genericIdInsert(db, "applications", a, r)
		})

	// Restrictions
	m.Get("/restrictions/:consumer/:application",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		func(db *mgo.Database, params martini.Params, r render.Render) {
			// listing := make([]interface{}, 0, 10)
			// db.C("consumers").Find(nil).All(&listing)
			// r.JSON(20
		})
	m.Post("/restrictions/:consumer/:application",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		validateJSONPayload("schema/restriction.schema"),
		func(db *mgo.Database, requestJSON JSONObject, params martini.Params, r render.Render) {

			// restriction := data.Restriction{
			// 	Consumer:    params["consumer"],
			// 	Application: params["application"],
			// }

			_, err := db.C("restrictions").Upsert(
				bson.M{
					"consumer":    params["consumer"],
					"application": params["application"],
				},
				bson.M{
					"$setOnInsert": bson.M{
						"consumer":     params["consumer"],
						"application":  params["application"],
						"restrictions": []string{}, // I imagine it doesn't care what the array type is at this point.
					},
					"$addToSet": requestJSON,
				},
			)

			if err != nil {
				r.JSON(500, data.APIError{Error: err.Error(), Context: "restriction"})
			}

			// if err != nil && mgo.IsDup(err) {
			// r.JSON(409, data.APIError{Error: "Already exists", Context: obj})
			// }

			// if res.Type == string(data.MaxCount) {
			// 	fmt.Println("COUNT TYPE", res)
			// 	if res.Maximum > 0 && (res.Counter == string(data.Usage) || res.Counter == string(data.Access)) {
			// 		fmt.Println("VALID")
			// 	} else {
			// 		r.JSON(400, data.APIError{Error: "Invalid Count Type, needs (counter:str, maximum:int)", Context: res})
			// 	}
			// } else if res.Type == string(data.NetAddr) {
			// 	fmt.Println("NET TYPE")
			// 	if res.Counter != "" && res.Maximum > 0 {
			// 		fmt.Println("VALID")
			// 	} else {
			// 		r.JSON(400, data.APIError{Error: "Invalid NetAddr Type, needs (IP:str, mask:str)", Context: res})
			// 	}
			// }

			// mcr, ok := res.(data.MaxCountRestriction)
			// if ok {
			// fmt.Println("YES!")
			// fmt.Println(mcr)
			// } else {
			// fmt.Println("Nooooop")
			// }
		})

	// COUNTING
	m.Post("/signal/:consumer/:application/:action",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		binding.Bind(data.Client{}),
		func(db *mgo.Database, params martini.Params, c data.Client, r render.Render) {
			// Ok, we have our consumer and application and the client should be validated at this point
			// if anything is required... Time to check restrictions.

		})

}

// Move to another file.
type listFn func(db *mgo.Database, r render.Render)
type lookupFn func(db *mgo.Database, params martini.Params, r render.Render)
type insertFn func(db *mgo.Database, obj interface{}, r render.Render)
type paramCheckFn func(db *mgo.Database, params martini.Params, r render.Render)
type bodyValidateFn func(req *http.Request, c martini.Context, r render.Render)

func listByType(collection string) listFn {
	return func(db *mgo.Database, r render.Render) {
		listing := make([]interface{}, 0, 10)
		db.C("consumers").Find(nil).All(&listing)
		r.JSON(200, listing)
	}
}

func findById(collection string) lookupFn {
	return func(db *mgo.Database, params martini.Params, r render.Render) {
		var obj interface{}

		err := db.C(collection).Find(bson.M{"_id": params["id"]}).One(&obj)

		if err != nil {
			if err == mgo.ErrNotFound {
				r.JSON(404, data.APIError{Error: "Not Found", Context: params})
			} else {
				r.JSON(500, data.APIError{Error: "Server Error", Context: err})
			}
		} else {
			r.JSON(200, obj)
		}
	}
}

func genericIdInsert(db *mgo.Database, collection string, obj interface{}, r render.Render) {
	err := db.C(collection).Insert(obj)

	if err != nil && mgo.IsDup(err) {
		r.JSON(409, data.APIError{Error: "Already exists", Context: obj})
	}
}

func checkIdParam(name string, collection string) paramCheckFn {
	return func(db *mgo.Database, params martini.Params, r render.Render) {
		var obj interface{}
		err := db.C(collection).Find(bson.M{"_id": params[name]}).One(&obj)
		if err != nil {
			if err == mgo.ErrNotFound {
				r.JSON(404, data.APIError{Error: fmt.Sprintf("%s not found in %s", name, collection),
					Context: params})
			} else {
				r.JSON(500, data.APIError{Error: "Server Error", Context: err})
			}
			return
		}
	}
}

type JSONObject struct {
	Object map[string]interface{}
}

func validateJSONPayload(schemaAssetName string) bodyValidateFn {
	return func(req *http.Request, c martini.Context, r render.Render) {
		var err error
		var schemaObject map[string]interface{}
		var requestBytes []byte
		var requestObject map[string]interface{}

		// Read request body into bytes
		if requestBytes, err = ioutil.ReadAll(req.Body); err != nil {
			r.JSON(400, data.APIError{Error: err.Error()})
			return
		}

		// Load schema from asset manager as bytes
		schemaBytes, err := Asset(schemaAssetName)
		if err != nil {
			r.JSON(400, data.APIError{Error: err.Error()})
			return
		}

		// Convert request to JSON
		err = json.Unmarshal(requestBytes, &requestObject)
		if err != nil {
			r.JSON(400, data.APIError{Error: err.Error()})
			return
		}

		// Convert schema to JSON
		err = json.Unmarshal(schemaBytes, &schemaObject)
		if err != nil {
			r.JSON(500, data.APIError{Error: err.Error()})
			return
		}

		// Initialize Schema based on JSON
		schema, err := gojsonschema.NewJsonSchemaDocument(schemaObject)
		if err != nil {
			r.JSON(500, data.APIError{Error: "Invalid Schema", Context: err.Error()})
			return
		}

		validationResult := schema.Validate(requestObject)

		if !validationResult.IsValid() {
			r.JSON(400, data.APIError{Error: "Validation Error", Context: validationResult.GetErrorMessages()})
			return
		}

		// Put this in the context so we can get it in the next handler.
		c.Map(JSONObject{
			Object: requestObject,
		})
	}
}

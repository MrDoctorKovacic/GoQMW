package main

import (
	"fmt"

	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions/gps"
	"github.com/MrDoctorKovacic/MDroid-Core/settings"
	"github.com/graphql-go/graphql"
	"github.com/rs/zerolog/log"
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"gps":          gps.Query,
			"sessionList":  sessions.SessionQuery,
			"settingsList": settings.SettingQuery,
		},
	})

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		/* Create new product item
		http://localhost:8080/product?query=mutation+_{create(name:"Inca Kola",info:"Inca Kola is a soft drink that was created in Peru in 1935 by British immigrant Joseph Robinson Lindley using lemon verbena (wiki)",price:1.99){id,name,info,price}}
		*/
		"setSession": sessions.SessionMutation,
		"setSetting": settings.SettingMutation,
	},
})

var schema, _ = graphql.NewSchema(
	graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	},
)

func executeQuery(query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		log.Error().Msg(fmt.Sprintf("Unexpected errors: %v", result.Errors))
	}
	return result
}

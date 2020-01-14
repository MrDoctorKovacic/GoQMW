package main

import (
	"github.com/graphql-go/graphql"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/sessions/gps"
	"github.com/qcasey/MDroid-Core/sessions/system"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"gps":          gps.Query,
			"stat":         system.Query,
			"sessionList":  sessions.SessionQuery,
			"settingsList": settings.SettingQuery,
		},
	})

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
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
		log.Error().Msgf("Unexpected errors: %v", result.Errors)
	}
	return result
}

package main

import (
	"fmt"

	"github.com/MrDoctorKovacic/MDroid-Core/sessions"
	"github.com/MrDoctorKovacic/MDroid-Core/sessions/gps"
	"github.com/graphql-go/graphql"
	"github.com/rs/zerolog/log"
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"gps":     gps.Query,
			"session": sessions.SessionQuery,
		},
	})

var schema, _ = graphql.NewSchema(
	graphql.SchemaConfig{
		Query: queryType,
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

package gps

import "github.com/graphql-go/graphql"

// FixType is a GraphQL type for GPS fixes
var FixType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Fix",
		Fields: graphql.Fields{
			"Latitude": &graphql.Field{
				Type: graphql.String,
			},
			"Longitude": &graphql.Field{
				Type: graphql.FieldType,
			},
			"Speed": &graphql.Field{
				Type: graphql.String,
			},
			"Course": &graphql.Field{
				Type: graphql.FieldType,
			},
		},
	},
)

// LocationType is a GraphQL type for GPS location
var LocationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Location",
		Fields: graphql.Fields{
			"Timezone": &graphql.Field{
				Type:        graphql.String,
				Description: "Working IANA Time Zone",
			},
			"Fix": &graphql.Field{
				Type:        FixType,
				Description: "Latest GPS fix",
			},
		},
	},
)

// Query is GraphQL schema for GPS GET requests
var Query = &graphql.Field{
	Type:        LocationType,
	Description: "Various location data",
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		return Get(), nil
	},
}

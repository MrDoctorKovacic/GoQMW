package stat

import (
	"fmt"

	"github.com/graphql-go/graphql"
)

// StatType is a GraphQL type for GPS location
var StatType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Stat",
		Fields: graphql.Fields{
			"UsedRAM": &graphql.Field{
				Type: graphql.String,
			},
			"UsedCPU": &graphql.Field{
				Type: graphql.FieldType,
			},
			"UsedDisk": &graphql.Field{
				Type: graphql.String,
			},
			"UsedNetwork": &graphql.Field{
				Type: graphql.FieldType,
			},
			"TempCPU": &graphql.Field{
				Type: graphql.FieldType,
			},
		},
	},
)

// Query is GraphQL schema for Stat GET requests
var Query = &graphql.Field{
	Type:        graphql.NewList(StatType),
	Description: "Stats on networked machines",
	Args: graphql.FieldConfigArgument{
		"names": &graphql.ArgumentConfig{
			Type:        graphql.NewList(graphql.String),
			Description: "Name of machine to fetch. If not provided, will get all machines",
		},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		names, ok := p.Args["names"].([]string)
		outputList := []stat{}
		if ok {
			for _, name := range names {
				mstats, ok := get(name)
				if !ok {
					return nil, fmt.Errorf("%s does not exist", name)
				}
				outputList = append(outputList, mstats)
			}
			return outputList, nil
		}

		// Return all stats
		for _, stat := range getAll() {
			outputList = append(outputList, stat)
		}
		return outputList, nil
	},
}

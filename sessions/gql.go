package sessions

import "github.com/graphql-go/graphql"

var sessionType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Session",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type:        graphql.String,
				Description: "Value Name",
			},
			"value": &graphql.Field{
				Type:        graphql.String,
				Description: "Value as a string",
			},
			"lastUpdate": &graphql.Field{
				Type:        graphql.String,
				Description: "UTC Time when inserted",
			},
		},
	},
)

// SessionMutation is a GraphQL schema for session POST requests
var SessionMutation = &graphql.Field{
	Type:        sessionType,
	Description: "Post new session value",
	Args: graphql.FieldConfigArgument{
		"name": &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
		"value": &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
	},
	Resolve: func(params graphql.ResolveParams) (interface{}, error) {
		return SetValue(params.Args["name"].(string), params.Args["value"].(string)), nil
	},
}

// SessionQuery is a GraphQL schema for session GET requests
var SessionQuery = &graphql.Field{
	Type:        graphql.NewList(sessionType),
	Description: "Get session values",
	Args: graphql.FieldConfigArgument{
		"names": &graphql.ArgumentConfig{
			Type:        graphql.NewList(graphql.String),
			Description: "List of names to fetch. If not provided, will get entire session",
		},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		var outputList []Data
		names, ok := p.Args["names"].([]string)
		if ok {
			for _, name := range names {
				s, err := Get(name)
				if err != nil {
					return nil, err
				}
				s.Name = name
				outputList = append(outputList, s)

			}
			return outputList, nil
		}

		// Return entire session
		s := GetAll()
		for _, val := range s {
			outputList = append(outputList, val)
		}
		return outputList, nil
	},
}

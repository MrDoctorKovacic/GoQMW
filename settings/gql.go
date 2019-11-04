package settings

import "github.com/graphql-go/graphql"

var componentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Component",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type:        graphql.String,
				Description: "Component Name",
			},
			"settings": &graphql.Field{
				Type:        graphql.NewList(settingType),
				Description: "Component Settings",
			},
		},
	},
)

var settingType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Setting",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type:        graphql.String,
				Description: "Setting Name",
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

// SettingMutation is a GraphQL schema for setting POST requests
var SettingMutation = &graphql.Field{
	Type:        graphql.Boolean,
	Description: "Post new setting value",
	Args: graphql.FieldConfigArgument{
		"component": &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
		"setting": &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
		"value": &graphql.ArgumentConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
	},
	Resolve: func(params graphql.ResolveParams) (interface{}, error) {
		return Set(params.Args["component"].(string), params.Args["setting"].(string), params.Args["value"].(string)), nil
	},
}

// SettingQuery is a GraphQL schema for setting GET requests
var SettingQuery = &graphql.Field{
	Type:        graphql.NewList(componentType),
	Description: "Get setting values",
	Args: graphql.FieldConfigArgument{
		"components": &graphql.ArgumentConfig{
			Type:        graphql.NewList(graphql.String),
			Description: "List of components to fetch. If not provided, will get all settings",
		},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		var outputList []Component
		components, ok := p.Args["components"].([]Component)
		if ok {
			for _, component := range components {
				c := Component{Name: component.Name, Settings: []Setting{}}
				s, err := GetComponent(component.Name)
				if err != nil {
					return nil, err
				}
				for name, value := range s {
					c.Settings = append(c.Settings, Setting{Name: name, Value: value})
				}
				outputList = append(outputList, c)
			}
			return outputList, nil
		}

		// Return entire setting
		/*
			s := GetAll()
			for _, val := range s {
				outputList = append(outputList, val)
			}
			return outputList, nil*/
		return nil, nil
	},
}

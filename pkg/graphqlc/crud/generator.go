package crud

import (
	"github.com/samlitowitz/graphqlc-gen-echo/pkg/graphqlc/echo"
	"github.com/samlitowitz/graphqlc/pkg/graphqlc"
	"strings"
)

type Generator struct {
	*echo.Generator

	config   *Config
	genFiles map[string]bool
}

func New() *Generator {
	g := new(Generator)
	g.Generator = echo.New()
	g.LogPrefix = "graphqlc-gen-crud"
	return g
}

func (g *Generator) CommandLineArguments(parameter string) {
	g.Generator.CommandLineArguments(parameter)

	for k, v := range g.Param {
		switch k {
		case "config":
			config, err := LoadConfig(v)
			if err != nil {
				g.Error(err, "failed to load configuration")
			}
			g.config = config
		}
	}
	if g.config == nil {
		g.Fail("a configuration must be provided")
	}
}

func (g *Generator) buildMapSpecDefs(typeFieldMap map[string]map[string]graphqlc.FieldDefinitionDescriptorProto) {
	for _, typeSpec := range g.config.Types {
		typeSpec.Create.Input.SkipMap = make(map[string]struct{})
		for _, fieldMap := range typeSpec.Create.Input.FieldMap {
			// clear def
			fieldMap.Def = nil
			if _, ok := typeFieldMap[fieldMap.Type]; !ok {
				continue
			}
			if _, ok := typeFieldMap[fieldMap.Type][fieldMap.Field]; !ok {
				continue
			}
			fieldDef := typeFieldMap[fieldMap.Type][fieldMap.Field]
			fieldDef.Name = fieldMap.Name
			fieldMap.Def = &fieldDef
		}
		typeSpec.Delete.Input.SkipMap = make(map[string]struct{})
		for _, typeName := range typeSpec.Delete.Input.Skip {
			typeSpec.Delete.Input.SkipMap[typeName] = struct{}{}
		}
		typeSpec.Update.Input.SkipMap = make(map[string]struct{})
		for _, typeName := range typeSpec.Update.Input.Skip {
			typeSpec.Update.Input.SkipMap[typeName] = struct{}{}
		}
	}
}

func (g *Generator) buildSkipMaps() {
	for _, typeSpec := range g.config.Types {
		typeSpec.Create.Input.SkipMap = make(map[string]struct{})
		for _, typeName := range typeSpec.Create.Input.Skip {
			typeSpec.Create.Input.SkipMap[typeName] = struct{}{}
		}
		typeSpec.Delete.Input.SkipMap = make(map[string]struct{})
		for _, typeName := range typeSpec.Delete.Input.Skip {
			typeSpec.Delete.Input.SkipMap[typeName] = struct{}{}
		}

		typeSpec.Update.Input.SkipMap = make(map[string]struct{})
		for _, typeName := range typeSpec.Update.Input.Skip {
			typeSpec.Update.Input.SkipMap[typeName] = struct{}{}
		}
	}
}

func buildTypeFieldMap(objDefs []*graphqlc.ObjectTypeDefinitionDescriptorProto) map[string]map[string]graphqlc.FieldDefinitionDescriptorProto {
	typeFieldMap := make(map[string]map[string]graphqlc.FieldDefinitionDescriptorProto)
	for _, objDef := range objDefs {
		if _, ok := typeFieldMap[objDef.Name]; !ok {
			typeFieldMap[objDef.Name] = make(map[string]graphqlc.FieldDefinitionDescriptorProto)
		}
		for _, fieldDef := range objDef.Fields {
			typeFieldMap[objDef.Name][fieldDef.Name] = *fieldDef
		}
	}
	return typeFieldMap
}

func (g *Generator) buildGenFiles() {
	g.genFiles = make(map[string]bool)
	for _, file := range g.Request.FileToGenerate {
		g.genFiles[file] = true
	}
}

func (g *Generator) CreateMutationsForToGenerateFiles() {
	g.buildSkipMaps()
	g.buildGenFiles()

	for _, fd := range g.Request.GraphqlFile {
		if gen, ok := g.genFiles[fd.Name]; !ok || !gen {
			continue
		}

		g.buildMapSpecDefs(buildTypeFieldMap(fd.Objects))

		objects := []*graphqlc.ObjectTypeDefinitionDescriptorProto{}
		inputObjects := []*graphqlc.InputObjectTypeDefinitionDescriptorProto{}
		fields := []*graphqlc.FieldDefinitionDescriptorProto{}

		for _, objDef := range fd.Objects {
			if objDef.Name == fd.Schema.Query.Name {
				continue
			}

			if objDef.Name == fd.Schema.Mutation.Name {
				fd.Schema.Mutation = objDef
				continue
			}

			if fd.Schema.Subscription != nil && objDef.Name == fd.Schema.Subscription.Name {
				continue
			}

			if _, ok := g.config.Types[objDef.Name]; !ok {
				continue
			}

			// create, all non-identifying fields
			createInput, createOutput, createMutation := buildCreateDefinitions(objDef, g.config.Types[objDef.Name])
			objects = append(objects, createOutput)
			inputObjects = append(inputObjects, createInput)
			fields = append(fields, createMutation)

			// delete, identifying fields required
			deleteInput, deleteOutput, deleteMutation := buildDeleteDefinitions(objDef, g.config.Types[objDef.Name])
			objects = append(objects, deleteOutput)
			inputObjects = append(inputObjects, deleteInput)
			fields = append(fields, deleteMutation)

			// update, all non-identifying fields optional, identifying fields required
			updateInput, updateOutput, updateMutation := buildUpdateDefinitions(objDef, g.config.Types[objDef.Name])
			objects = append(objects, updateOutput)
			inputObjects = append(inputObjects, updateInput)
			fields = append(fields, updateMutation)
		}

		if fd.Schema.Mutation == nil {
			fd.Schema.Mutation = BuildMutationRoot()
			fd.Objects = append(fd.Objects, fd.Schema.Mutation)
		}
		fd.InputObjects = append(fd.InputObjects, inputObjects...)
		fd.Objects = append(fd.Objects, objects...)
		fd.Schema.Mutation.Fields = append(fd.Schema.Mutation.Fields, fields...)
	}
}

func buildCreateDefinitions(objDef *graphqlc.ObjectTypeDefinitionDescriptorProto, typeSpec TypeSpec) (*graphqlc.InputObjectTypeDefinitionDescriptorProto, *graphqlc.ObjectTypeDefinitionDescriptorProto, *graphqlc.FieldDefinitionDescriptorProto) {
	inputFields := make([]*graphqlc.FieldDefinitionDescriptorProto, 0)
	for _, fieldDef := range objDef.Fields {
		// Skip identifier fields
		if fieldDef.Name == typeSpec.Identifier {
			continue
		}
		// Skip skip fields
		if _, ok := typeSpec.Create.Input.SkipMap[fieldDef.Name]; ok {
			continue
		}
		// replace map
		if fieldMap, ok := typeSpec.Create.Input.FieldMap[fieldDef.Name]; ok {
			fieldDef = fieldMap.Def
		}
		inputFields = append(inputFields, fieldDef)
	}

	inputDef := buildCreateInputDefinition(objDef.Name, inputFields)
	outputDef := buildCreateOutputDefinition(objDef.Name, []*graphqlc.FieldDefinitionDescriptorProto{
		{
			Name: strings.ToLower(objDef.Name),
			Type: &graphqlc.TypeDescriptorProto{
				Type: &graphqlc.TypeDescriptorProto_NamedType{
					NamedType: &graphqlc.NamedTypeDescriptorProto{
						Name: objDef.Name,
					},
				},
			},
		},
	})

	return inputDef, outputDef, buildMutation("Create"+objDef.Name, inputDef, outputDef)
}

func buildCreateInputDefinition(typeName string, fields []*graphqlc.FieldDefinitionDescriptorProto) *graphqlc.InputObjectTypeDefinitionDescriptorProto {
	return &graphqlc.InputObjectTypeDefinitionDescriptorProto{
		Name:   "Create" + typeName + "Input",
		Fields: BuildInputValueDefsFromFieldDefs(fields),
	}
}

func buildCreateOutputDefinition(typeName string, fields []*graphqlc.FieldDefinitionDescriptorProto) *graphqlc.ObjectTypeDefinitionDescriptorProto {
	return &graphqlc.ObjectTypeDefinitionDescriptorProto{
		Name:   "Create" + typeName + "Output",
		Fields: fields,
	}
}

func buildDeleteDefinitions(objDef *graphqlc.ObjectTypeDefinitionDescriptorProto, typeSpec TypeSpec) (*graphqlc.InputObjectTypeDefinitionDescriptorProto, *graphqlc.ObjectTypeDefinitionDescriptorProto, *graphqlc.FieldDefinitionDescriptorProto) {
	inputFields := make([]*graphqlc.FieldDefinitionDescriptorProto, 0)
	for _, fieldDef := range objDef.Fields {
		// Skip non-identifier fields
		if fieldDef.Name != typeSpec.Identifier {
			continue
		}
		inputFields = append(inputFields, fieldDef)
	}

	inputDef := buildDeleteInputDefinition(objDef.Name, inputFields)
	outputDef := buildDeleteOutputDefinition(objDef.Name, []*graphqlc.FieldDefinitionDescriptorProto{
		{
			Name: strings.ToLower(objDef.Name),
			Type: &graphqlc.TypeDescriptorProto{
				Type: &graphqlc.TypeDescriptorProto_NamedType{
					NamedType: &graphqlc.NamedTypeDescriptorProto{
						Name: objDef.Name,
					},
				},
			},
		},
	})

	return inputDef, outputDef, buildMutation("Delete"+objDef.Name, inputDef, outputDef)
}

func buildDeleteInputDefinition(typeName string, fields []*graphqlc.FieldDefinitionDescriptorProto) *graphqlc.InputObjectTypeDefinitionDescriptorProto {
	return &graphqlc.InputObjectTypeDefinitionDescriptorProto{
		Name:   "Delete" + typeName + "Input",
		Fields: BuildInputValueDefsFromFieldDefs(fields),
	}
}

func buildDeleteOutputDefinition(typeName string, fields []*graphqlc.FieldDefinitionDescriptorProto) *graphqlc.ObjectTypeDefinitionDescriptorProto {
	return &graphqlc.ObjectTypeDefinitionDescriptorProto{
		Name:   "Delete" + typeName + "Output",
		Fields: fields,
	}
}

func buildUpdateDefinitions(objDef *graphqlc.ObjectTypeDefinitionDescriptorProto, typeSpec TypeSpec) (*graphqlc.InputObjectTypeDefinitionDescriptorProto, *graphqlc.ObjectTypeDefinitionDescriptorProto, *graphqlc.FieldDefinitionDescriptorProto) {
	inputFields := make([]*graphqlc.FieldDefinitionDescriptorProto, 0)
	for _, fieldDef := range objDef.Fields {
		// Skip skip fields
		if _, ok := typeSpec.Update.Input.SkipMap[fieldDef.Name]; ok {
			continue
		}
		// replace map
		if fieldMap, ok := typeSpec.Update.Input.FieldMap[fieldDef.Name]; ok {
			fieldDef = fieldMap.Def
		}
		inputFields = append(inputFields, fieldDef)
	}

	inputDef := buildUpdateInputDefinition(objDef.Name, inputFields)
	outputDef := buildUpdateOutputDefinition(objDef.Name, []*graphqlc.FieldDefinitionDescriptorProto{
		{
			Name: strings.ToLower(objDef.Name),
			Type: &graphqlc.TypeDescriptorProto{
				Type: &graphqlc.TypeDescriptorProto_NamedType{
					NamedType: &graphqlc.NamedTypeDescriptorProto{
						Name: objDef.Name,
					},
				},
			},
		},
	})

	return inputDef, outputDef, buildMutation("Update"+objDef.Name, inputDef, outputDef)
}

func buildUpdateInputDefinition(typeName string, fields []*graphqlc.FieldDefinitionDescriptorProto) *graphqlc.InputObjectTypeDefinitionDescriptorProto {
	return &graphqlc.InputObjectTypeDefinitionDescriptorProto{
		Name:   "Update" + typeName + "Input",
		Fields: BuildInputValueDefsFromFieldDefs(fields),
	}
}

func buildUpdateOutputDefinition(typeName string, fields []*graphqlc.FieldDefinitionDescriptorProto) *graphqlc.ObjectTypeDefinitionDescriptorProto {
	return &graphqlc.ObjectTypeDefinitionDescriptorProto{
		Name:   "Update" + typeName + "Output",
		Fields: fields,
	}
}

func buildMutation(name string, input *graphqlc.InputObjectTypeDefinitionDescriptorProto, output *graphqlc.ObjectTypeDefinitionDescriptorProto) *graphqlc.FieldDefinitionDescriptorProto {
	return &graphqlc.FieldDefinitionDescriptorProto{
		Name: name,
		Arguments: []*graphqlc.InputValueDefinitionDescriptorProto{
			{
				Name: "input",
				Type: &graphqlc.TypeDescriptorProto{
					Type: &graphqlc.TypeDescriptorProto_NonNullType{
						NonNullType: &graphqlc.NonNullTypeDescriptorProto{
							Type: &graphqlc.NonNullTypeDescriptorProto_NamedType{
								NamedType: &graphqlc.NamedTypeDescriptorProto{
									Name: input.Name,
								},
							},
						},
					},
				},
			},
		},
		Type: &graphqlc.TypeDescriptorProto{
			Type: &graphqlc.TypeDescriptorProto_NamedType{
				NamedType: &graphqlc.NamedTypeDescriptorProto{
					Name: output.Name,
				},
			},
		},
	}
}

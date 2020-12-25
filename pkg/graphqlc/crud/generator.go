package crud

import (
	"github.com/samlitowitz/graphqlc-gen-echo/pkg/graphqlc/echo"
	"github.com/samlitowitz/graphqlc/pkg/graphqlc"
)

type Generator struct {
	*echo.Generator

	config       *Config
	crudifyTypes map[string]struct{}
	genFiles     map[string]bool

	IdentifierRemover IdentifierRemover
}

func New() *Generator {
	g := new(Generator)
	g.Generator = echo.New()
	g.IdentifierRemover = &NodeIdentifierRemover{}
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
				g.Error(err)
			}
			g.config = config
		}
	}
	if g.config == nil {
		g.Fail("a configuration must be provided")
	}
}

func (g *Generator) buildCrudifyTypes() {
	g.crudifyTypes = make(map[string]struct{})
	for _, typeName := range g.config.Crudify {
		g.crudifyTypes[typeName] = struct{}{}
	}
}

func (g *Generator) buildGenFiles() {
	g.genFiles = make(map[string]bool)
	for _, file := range g.Request.FileToGenerate {
		g.genFiles[file] = true
	}
}

func (g *Generator) CreateMutationsForToGenerateFiles() {
	g.buildCrudifyTypes()
	g.buildGenFiles()

	if g.IdentifierRemover == nil {
		g.IdentifierRemover = &NoIdentifierRemover{}
	}

	for _, fd := range g.Request.GraphqlFile {
		if gen, ok := g.genFiles[fd.Name]; !ok || !gen {
			continue
		}
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

			if _, ok := g.crudifyTypes[objDef.Name]; !ok {
				continue
			}

			// create, all non-identifying fields
			createInput, createOutput, createMutation := buildCreateDefinitions(objDef.Name, objDef.Fields, g.IdentifierRemover)
			objects = append(objects, createOutput)
			inputObjects = append(inputObjects, createInput)
			fields = append(fields, createMutation)

			// delete, identifying fields required
			deleteInput, deleteOutput, deleteMutation := buildDeleteDefinitions(objDef.Name, objDef.Fields, g.IdentifierRemover)
			objects = append(objects, deleteOutput)
			inputObjects = append(inputObjects, deleteInput)
			fields = append(fields, deleteMutation)

			// update, all non-identifying fields optional, identifying fields required
			updateInput, updateOutput, updateMutation := buildUpdateDefinitions(objDef.Name, objDef.Fields, g.IdentifierRemover)
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

func buildCreateDefinitions(name string, fields []*graphqlc.FieldDefinitionDescriptorProto, ir IdentifierRemover) (*graphqlc.InputObjectTypeDefinitionDescriptorProto, *graphqlc.ObjectTypeDefinitionDescriptorProto, *graphqlc.FieldDefinitionDescriptorProto) {
	nonIdentifierFields := ir.RemoveIdentifierFields(append([]*graphqlc.FieldDefinitionDescriptorProto(nil), fields...))
	identifierFields := ir.RemoveNonIdentifierFields(append([]*graphqlc.FieldDefinitionDescriptorProto(nil), fields...))

	inputDef := buildCreateInputDefinition(name, nonIdentifierFields)
	outputDef := buildCreateOutputDefinition(name, append(identifierFields, nonIdentifierFields...))
	return inputDef, outputDef, buildMutation("Create"+name, inputDef, outputDef)
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

func buildDeleteDefinitions(name string, fields []*graphqlc.FieldDefinitionDescriptorProto, ir IdentifierRemover) (*graphqlc.InputObjectTypeDefinitionDescriptorProto, *graphqlc.ObjectTypeDefinitionDescriptorProto, *graphqlc.FieldDefinitionDescriptorProto) {
	identifierFields := ir.RemoveNonIdentifierFields(append([]*graphqlc.FieldDefinitionDescriptorProto(nil), fields...))

	inputDef := buildDeleteInputDefinition(name, identifierFields)
	outputDef := buildDeleteOutputDefinition(name, fields)
	return inputDef, outputDef, buildMutation("Delete"+name, inputDef, outputDef)
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

func buildUpdateDefinitions(name string, fields []*graphqlc.FieldDefinitionDescriptorProto, ir IdentifierRemover) (*graphqlc.InputObjectTypeDefinitionDescriptorProto, *graphqlc.ObjectTypeDefinitionDescriptorProto, *graphqlc.FieldDefinitionDescriptorProto) {
	identifierFields := append([]*graphqlc.FieldDefinitionDescriptorProto(nil), fields...)
	nonIdentifierFields := append([]*graphqlc.FieldDefinitionDescriptorProto(nil), fields...)
	for i, field := range fields {
		identDef := *field
		identifierFields[i] = &identDef
		nonIdentDef := *field
		nonIdentifierFields[i] = &nonIdentDef
	}

	identifierFields = AddNonNullToFieldDefs(ir.RemoveNonIdentifierFields(append([]*graphqlc.FieldDefinitionDescriptorProto(nil), identifierFields...)))
	nonIdentifierFields = RemoveNonNullFromFieldDefs(ir.RemoveIdentifierFields(append([]*graphqlc.FieldDefinitionDescriptorProto(nil), nonIdentifierFields...)))

	inputDef := buildUpdateInputDefinition(name, append(identifierFields, nonIdentifierFields...))
	outputDef := buildUpdateOutputDefinition(name, append([]*graphqlc.FieldDefinitionDescriptorProto(nil), fields...))
	return inputDef, outputDef, buildMutation("Update"+name, inputDef, outputDef)
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

type IdentifierRemover interface {
	RemoveInterfaces([]*graphqlc.InterfaceTypeDefinitionDescriptorProto) []*graphqlc.InterfaceTypeDefinitionDescriptorProto
	RemoveIdentifierFields([]*graphqlc.FieldDefinitionDescriptorProto) []*graphqlc.FieldDefinitionDescriptorProto
	RemoveNonIdentifierFields([]*graphqlc.FieldDefinitionDescriptorProto) []*graphqlc.FieldDefinitionDescriptorProto
}

type NoIdentifierRemover struct{}

func (ni *NoIdentifierRemover) RemoveInterfaces(ifaces []*graphqlc.InterfaceTypeDefinitionDescriptorProto) []*graphqlc.InterfaceTypeDefinitionDescriptorProto {
	return ifaces
}

func (ni *NoIdentifierRemover) RemoveIdentifierFields(fields []*graphqlc.FieldDefinitionDescriptorProto) []*graphqlc.FieldDefinitionDescriptorProto {
	return fields
}

func (ni *NoIdentifierRemover) RemoveNonIdentifierFields(fields []*graphqlc.FieldDefinitionDescriptorProto) []*graphqlc.FieldDefinitionDescriptorProto {
	return fields
}

type NodeIdentifierRemover struct{}

func (nir *NodeIdentifierRemover) RemoveInterfaces(ifaces []*graphqlc.InterfaceTypeDefinitionDescriptorProto) []*graphqlc.InterfaceTypeDefinitionDescriptorProto {
	namesMap := map[string]bool{"Node": true}
	result := make([]*graphqlc.InterfaceTypeDefinitionDescriptorProto, 0)

	for _, iface := range ifaces {
		if _, ok := namesMap[iface.Name]; ok {
			continue
		}
		result = append(result, iface)
	}
	return result
}

func (nir *NodeIdentifierRemover) RemoveIdentifierFields(fieldDefs []*graphqlc.FieldDefinitionDescriptorProto) []*graphqlc.FieldDefinitionDescriptorProto {
	return RemoveFieldDefByNames(fieldDefs, map[string]byte{"id": 0x00})
}

func (nir *NodeIdentifierRemover) RemoveNonIdentifierFields(fields []*graphqlc.FieldDefinitionDescriptorProto) []*graphqlc.FieldDefinitionDescriptorProto {
	fieldNamesMap := make(map[string]byte)
	for _, field := range fields {
		if field.Name == "id" {
			continue
		}
		fieldNamesMap[field.Name] = 0x00
	}
	return RemoveFieldDefByNames(fields, fieldNamesMap)
}

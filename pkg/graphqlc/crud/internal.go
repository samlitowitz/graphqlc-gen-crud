package crud

import (
	core "github.com/samlitowitz/graphqlc/pkg/graphqlc"
)

func BuildInputObject(name string, fieldDefs []*core.InputValueDefinitionDescriptorProto) *core.InputObjectTypeDefinitionDescriptorProto {
	return &core.InputObjectTypeDefinitionDescriptorProto{
		Name:   name,
		Fields: fieldDefs,
	}
}

func BuildReturnObject(name string, fieldDefs []*core.FieldDefinitionDescriptorProto) *core.ObjectTypeDefinitionDescriptorProto {
	return &core.ObjectTypeDefinitionDescriptorProto{
		Name:   name,
		Fields: fieldDefs,
	}
}

func BuildInputValueDefsFromFieldDefs(fieldDefs []*core.FieldDefinitionDescriptorProto) []*core.InputValueDefinitionDescriptorProto {
	inputValueDefs := make([]*core.InputValueDefinitionDescriptorProto, 0)
	for _, fieldDef := range fieldDefs {
		inputValueDefs = append(inputValueDefs, BuildInputValueDefFromFieldDef(fieldDef))
	}
	return inputValueDefs
}

func BuildInputValueDefFromFieldDef(fieldDef *core.FieldDefinitionDescriptorProto) *core.InputValueDefinitionDescriptorProto {
	return &core.InputValueDefinitionDescriptorProto{
		Description:  fieldDef.Description,
		Name:         fieldDef.Name,
		Type:         fieldDef.Type,
		DefaultValue: nil,
		Directives:   fieldDef.Directives,
	}
}

func BuildMutationRoot() *core.ObjectTypeDefinitionDescriptorProto {
	return &core.ObjectTypeDefinitionDescriptorProto{
		Name:   "Mutation",
		Fields: []*core.FieldDefinitionDescriptorProto{},
	}
}

func AddNonNullToFieldDefs(fields []*core.FieldDefinitionDescriptorProto) []*core.FieldDefinitionDescriptorProto {
	for _, field := range fields {
		if field.Type.GetNonNullType() != nil {
			continue
		}

		if field.Type.GetNamedType() != nil {
			field.Type = &core.TypeDescriptorProto{
				Type: &core.TypeDescriptorProto_NonNullType{
					NonNullType: &core.NonNullTypeDescriptorProto{
						Type: &core.NonNullTypeDescriptorProto_NamedType{
							NamedType: &core.NamedTypeDescriptorProto{
								Name: field.Type.GetNamedType().Name,
							},
						},
					},
				},
			}
		}

		if field.Type.GetListType() != nil {
			field.Type = &core.TypeDescriptorProto{
				Type: &core.TypeDescriptorProto_NonNullType{
					NonNullType: &core.NonNullTypeDescriptorProto{
						Type: &core.NonNullTypeDescriptorProto_ListType{
							ListType: field.Type.GetListType(),
						},
					},
				},
			}
		}
	}
	return fields
}

func RemoveNonNullFromFieldDefs(fields []*core.FieldDefinitionDescriptorProto) []*core.FieldDefinitionDescriptorProto {
	for _, field := range fields {
		nonNullType := field.Type.GetNonNullType()
		if nonNullType == nil {
			continue
		}

		if nonNullType.GetNamedType() != nil {
			field.Type = &core.TypeDescriptorProto{
				Type: &core.TypeDescriptorProto_NamedType{
					NamedType: nonNullType.GetNamedType(),
				},
			}
		}

		if nonNullType.GetListType() != nil {
			field.Type = &core.TypeDescriptorProto{
				Type: &core.TypeDescriptorProto_ListType{
					ListType: nonNullType.GetListType().Type.GetListType(),
				},
			}
		}
	}
	return fields
}

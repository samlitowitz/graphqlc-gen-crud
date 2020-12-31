package crud

import (
	"github.com/samlitowitz/graphqlc/pkg/graphqlc"
)

type Config struct {
	Types map[string]TypeSpec `json:"types,omitempty"`
}

type TypeSpec struct {
	Identifier string         `json:"identifier,omitempty"`
	Create     TypeInputSpec `json:"create,omitempty"`
	Update     TypeInputSpec `json:"update,omitempty"`
	Delete     TypeInputSpec `json:"delete,omitempty"`
}

type TypeInputSpec struct {
	Input InputSpec `json:"input,omitempty"`
}

type InputSpec struct {
	FieldMap map[string]MapSpec  `json:"fieldMap,omitempty"`
	Skip     []string            `json:"skip,omitempty"`
	SkipMap  map[string]struct{} `json:"-"`
}

type MapSpec struct {
	Name  string                                   `json:"name,omitempty"`
	Type  string                                   `json:"type,omitempty"`
	Field string                                   `json:"field,omitempty"`
	Def   *graphqlc.FieldDefinitionDescriptorProto `json:"-"`
}

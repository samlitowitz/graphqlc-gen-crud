package main

import (
	"github.com/golang/protobuf/proto"
	"github.com/samlitowitz/graphqlc-gen-crud/pkg/graphqlc/crud"

	"io/ioutil"
	"os"
)

func main() {
	g := crud.New()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		g.Error(err, "reading input")
	}

	if err := proto.Unmarshal(data, g.Request); err != nil {
		g.Error(err, "parsing input proto")
	}

	if len(g.Request.FileToGenerate) == 0 {
		g.Fail("no files to generate")
	}

	g.CommandLineArguments(g.Request.Parameter)
	g.CreateMutationsForToGenerateFiles()
	g.GenerateSchemaFiles()

	data, err = proto.Marshal(g.Response)
	if err != nil {
		g.Error(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		g.Error(err, "failed to write output proto")
	}
}

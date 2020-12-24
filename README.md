# graphqlc-gen-crud
[![Go Report Card](https://goreportcard.com/badge/github.com/samlitowitz/graphqlc-gen-crud)](https://goreportcard.com/report/github.com/samlitowitz/graphqlc-gen-crud)
[![Go Reference](https://pkg.go.dev/badge/github.com/samlitowitz/graphqlc-gen-crud.svg)](https://pkg.go.dev/github.com/samlitowitz/graphqlc-gen-crud)

This is a code generator designed to work with [graphqlc](https://github.com/samlitowitz/graphqlc).

Generate a GraphQL schema from a GraphQL schema with added Relay Node interfaces and Connection types for the specified types.
See the [examples/](examples/) directory for more... examples.

# Installation
Install [graphqlc](https://github.com/samlitowitz/graphqlc).

`go get -u github.com/samlitowitz/graphqlc-gen-crud/cmd/graphqlc-gen-crud`

# Usage


## Parameters
  * config, required, name of crud configuration file as defined directly above,
  * suffix, optional, default = .echo.graphql, suffix for output file

`graphqlc --crud_out=config=crud.json:. schema.graphql`
  
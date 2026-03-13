#!/bin/bash

cat <<< '
go 1.26

use (
	./
	./jsonschema
)' > go.work

go run ./jsonschema
rm -f go.work go.work.sum

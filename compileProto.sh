#!/usr/bin/env bash

# CURRENTLY BROKEN; THIS COMMAND NEEDS TO BE RERUN MANUALLY AND THEN FILES MOVED AROUND - WILL FIX IT SOON
#protoc --go_out=. ./config/structs.proto
protoc --go_out=. ./sphinx/sphinx_structs.proto
protoc --go_out=. ./client/rpc/types/types.proto


#protoc ./common/grpc/services/proto/services.proto --go_out=plugins=grpc:../../..


#! /usr/bin/env bash

set -e

if ! [[ -x "$(command -v protoc-gen-go)" ]]; then
  echo "Need to install protoc-gen-go"
  exit 1
fi

PROTOC_OPTS="-I./third_party/googleapis/ -I./apis/ --go_out=temp --go-grpc_out=temp"

mkdir -p temp/

protoc $PROTOC_OPTS ./apis/tinkoff/cloud/stt/v1/*.proto
protoc $PROTOC_OPTS ./apis/tinkoff/cloud/tts/v1/*.proto
protoc $PROTOC_OPTS ./apis/tinkoff/cloud/longrunning/v1/*.proto

rm -rf pkg/tinkoff_voicekit

mv temp/github.com/kanataidarov/tinkoff_voicekit/pkg/* pkg

rm -rf temp/

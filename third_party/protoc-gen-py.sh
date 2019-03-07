#!/bin/bash
PY_OUT=../python-client

python3 -m grpc_tools.protoc -I/usr/local/include -I. \
  --proto_path=api/proto/v1 \
  --proto_path=third_party \
  --python_out=$PY_OUT \
  --grpc_python_out=$PY_OUT \
google/api/annotations.proto google/api/http.proto protoc-gen-swagger/options/annotations.proto protoc-gen-swagger/options/openapiv2.proto simulation-service.proto

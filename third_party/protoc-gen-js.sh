#!/bin/bash
JS_OUT=../web-client/pkg/api/v1

protoc -I/usr/local/include -I. \
 --proto_path=api/proto/v1 \
 --proto_path=third_party \
 --js_out=import_style=commonjs:$JS_OUT \
 --grpc-web_out=import_style=commonjs,mode=grpcwebtext:$JS_OUT \
simulation-service.proto

protoc -I/usr/local/include -I. \
 --proto_path=api/proto/v1 \
 --proto_path=third_party \
 --js_out=import_style=commonjs:$JS_OUT \
 --grpc-web_out=import_style=commonjs,mode=grpcwebtext:$JS_OUT \
google/api/annotations.proto google/api/http.proto

protoc -I/usr/local/include -I. \
 --proto_path=api/proto/v1 \
 --proto_path=third_party \
 --js_out=import_style=commonjs:$JS_OUT \
 --grpc-web_out=import_style=commonjs,mode=grpcwebtext:$JS_OUT \
protoc-gen-swagger/options/annotations.proto protoc-gen-swagger/options/openapiv2.proto
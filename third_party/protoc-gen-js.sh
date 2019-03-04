protoc -I/usr/local/include -I. \
 --proto_path=api/proto/v1 \
 --proto_path=third_party \
 --js_out=import_style=commonjs:pkg/api/v1/js \
 --grpc-web_out=import_style=commonjs,mode=grpcwebtext:pkg/api/v1/js \
todo-service.proto

protoc -I/usr/local/include -I. \
 --proto_path=api/proto/v1 \
 --proto_path=third_party \
 --js_out=import_style=commonjs:pkg/api/v1/js \
 --grpc-web_out=import_style=commonjs,mode=grpcwebtext:pkg/api/v1/js \
google/api/annotations.proto google/api/http.proto

protoc -I/usr/local/include -I. \
 --proto_path=api/proto/v1 \
 --proto_path=third_party \
 --js_out=import_style=commonjs:pkg/api/v1/js \
 --grpc-web_out=import_style=commonjs,mode=grpcwebtext:pkg/api/v1/js \
protoc-gen-swagger/options/annotations.proto protoc-gen-swagger/options/openapiv2.proto
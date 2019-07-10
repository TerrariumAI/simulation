FROM scratch

ADD ./bin/environment /
ADD serviceAccountKey.json /

EXPOSE 8000

ENTRYPOINT ["/environment", "-grpc-port=8000", "-env=prod"]

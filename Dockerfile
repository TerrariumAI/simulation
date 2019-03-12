FROM scratch

ADD main /
ADD serviceAccountKey.json /

EXPOSE 8080 9090

ENTRYPOINT ["/main", "-grpc-port=9090", "-http-port=8080", "-env=prod"]

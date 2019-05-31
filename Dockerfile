FROM scratch

ADD main /
ADD serviceAccountKey.json /

EXPOSE 8000

ENTRYPOINT ["/main", "-grpc-port=8000", "-env=prod"]

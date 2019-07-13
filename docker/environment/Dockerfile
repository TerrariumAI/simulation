FROM scratch

ADD ./bin/environment /
ADD serviceAccountKey.json /
ADD serviceAccountKey_staging.json /
ADD .env /

EXPOSE 9090

ENTRYPOINT ["/environment", "-grpc-port=9090", "-env=prod"]

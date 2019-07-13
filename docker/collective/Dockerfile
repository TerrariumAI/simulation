FROM scratch

ADD ./bin/collective /
ADD serviceAccountKey.json /
ADD .env /

EXPOSE 9090

ENTRYPOINT ["/environment", "-grpc-port=9090", "-env=prod", "-environment-addr=simulation.default.svc.cluster.local"]

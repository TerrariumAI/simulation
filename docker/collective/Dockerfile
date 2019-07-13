FROM scratch

ADD ./bin/collective /
ADD serviceAccountKey.json /
ADD serviceAccountKey_staging.json /
ADD .env /

EXPOSE 9090

ENTRYPOINT ["/environment", "-grpc-port=9090", "-redis-addr=10.0.0.3:6379", "-env=prod", "-environment-addr=simulation.default.svc.cluster.local"]

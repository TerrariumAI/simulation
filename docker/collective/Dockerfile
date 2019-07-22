FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
ENV PATH=/bin
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ADD ./bin/collective /
ADD serviceAccountKey.json /
ADD serviceAccountKey_staging.json /

EXPOSE 9090

ENTRYPOINT ["/collective", "-grpc-port=9090", "-redis-addr=10.0.0.3:6379", "-env=prod", "-environment-addr=environment.default.svc.cluster.local:9090"]

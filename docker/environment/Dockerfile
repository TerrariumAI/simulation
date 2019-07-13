FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
ENV PATH=/bin
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ADD ./bin/environment /
ADD serviceAccountKey.json /
ADD serviceAccountKey_staging.json /
ADD .env /

EXPOSE 9090

ENTRYPOINT ["/environment", "-grpc-port=9090", "-env=prod"]

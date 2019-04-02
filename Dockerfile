FROM alpine:latest as certs
RUN apk --update add ca-certificates


FROM scratch
ENV PATH=/bin
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ADD main /
ADD serviceAccountKey.json /

EXPOSE 9089 9090

ENTRYPOINT ["/main", "-grpc-port=9090", "-http-port=9089", "-env=prod"]

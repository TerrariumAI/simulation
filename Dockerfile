FROM scratch

ADD main /

EXPOSE 8080 9090

CMD ["/main", "-grpc-port=9090", "-http-port=8080"]

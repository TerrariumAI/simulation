FROM scratch

COPY server-cert.pem /
COPY server-key.pem /

ADD main /

EXPOSE 50051

CMD ["/main"]


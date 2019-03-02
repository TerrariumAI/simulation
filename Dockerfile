FROM scratch

COPY server-cert.pem /
COPY server-key.pem /

ADD main /

EXPOSE 8081
EXPOSE 50051


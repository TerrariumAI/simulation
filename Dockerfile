FROM scratch
COPY server-cert.pem /
COPY server-key.pem /
ADD main /
CMD ["/main"]
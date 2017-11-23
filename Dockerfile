FROM scratch

HEALTHCHECK --retries=10 CMD https://localhost:1080/health

EXPOSE 1080
ENTRYPOINT [ "/bin/sh" ]

COPY bin/fibr /bin/sh
COPY web/ web/
COPY cacert.pem /etc/ssl/certs/ca-certificates.crt

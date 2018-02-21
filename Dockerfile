FROM scratch

HEALTHCHECK --retries=10 CMD [ "/fibr", "-url", "https://localhost:1080/health" ]

EXPOSE 1080
ENTRYPOINT [ "/fibr" ]

COPY bin/fibr /fibr
COPY web/ web/
COPY cacert.pem /etc/ssl/certs/ca-certificates.crt

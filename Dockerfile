FROM vibioh/scratch

ENV FIBR_CSP "default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:"
ENV FIBR_FRAME_OPTIONS SAMEORIGIN

EXPOSE 1080
COPY templates/ templates/

HEALTHCHECK --retries=10 CMD [ "/fibr", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/fibr" ]

ARG VERSION
ENV VERSION ${VERSION}
ENV FIBR_VERSION ${VERSION}

ARG TARGETOS
ARG TARGETARCH

COPY mime.types /etc/mime.types
COPY cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY release/fibr_${TARGETOS}_${TARGETARCH} /fibr

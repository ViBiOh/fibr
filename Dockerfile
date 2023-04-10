FROM vibioh/scratch

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/fibr", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/fibr" ]

ARG VERSION
ENV VERSION ${VERSION}

ARG TARGETOS
ARG TARGETARCH

COPY passwd /etc/passwd
USER 995

VOLUME /tmp

COPY wait_${TARGETOS}_${TARGETARCH} /wait

COPY mime.types /etc/mime.types
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY release/fibr_${TARGETOS}_${TARGETARCH} /fibr

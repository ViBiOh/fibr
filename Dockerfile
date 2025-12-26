FROM rg.fr-par.scw.cloud/vibioh/scratch

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/fibr", "-url", "http://127.0.0.1:1080/health" ]
ENTRYPOINT [ "/fibr" ]

ARG VERSION
ENV VERSION ${VERSION}

ARG GIT_SHA
ENV GIT_SHA ${GIT_SHA}

ARG TARGETOS
ARG TARGETARCH

COPY passwd /etc/passwd
USER 995

VOLUME /tmp

COPY wait_${TARGETOS}_${TARGETARCH} /wait

COPY mime.types /etc/mime.types
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY release/fibr_${TARGETOS}_${TARGETARCH} /fibr

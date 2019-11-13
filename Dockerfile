FROM golang:1.13 as builder

WORKDIR /app
COPY . .

RUN make \
 && git diff -- *.go \
 && git diff --quiet -- *.go

ARG CODECOV_TOKEN
RUN curl -sSqL --max-time 10 https://codecov.io/bash | bash

FROM alpine as fetcher

WORKDIR /app

RUN apk --update add curl \
 && curl -sSqL --max-time 10 -o /app/cacert.pem https://curl.haxx.se/ca/cacert.pem

FROM scratch

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/fibr", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/fibr" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

COPY templates/ templates/

COPY --from=fetcher /app/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/bin/fibr /fibr

FROM golang:1.12 as builder

ENV APP_NAME fibr

WORKDIR /app
COPY . .

RUN make ${APP_NAME} \
 && curl -s -o /app/cacert.pem https://curl.haxx.se/ca/cacert.pem

FROM scratch

ENV APP_NAME fibr
HEALTHCHECK --retries=10 CMD [ "/fibr", "-url", "http://localhost:1080/health" ]

EXPOSE 1080
ENTRYPOINT [ "/fibr" ]

COPY templates/ templates/

COPY --from=builder /app/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/bin/${APP_NAME} /fibr

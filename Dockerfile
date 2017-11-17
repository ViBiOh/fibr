FROM scratch

HEALTHCHECK --retries=10 CMD https://localhost:1080/health
VOLUME /data

EXPOSE 1080
ENTRYPOINT [ "/bin/sh" ]

COPY bin/fibr /bin/sh

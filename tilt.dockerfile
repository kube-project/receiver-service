FROM alpine
WORKDIR /
COPY ./bin/receiver /receiver

EXPOSE 8000

ENTRYPOINT ["/receiver"]

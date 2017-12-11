FROM golang

COPY target/go-hello-serf /app/

WORKDIR /app
ENTRYPOINT ["./go-hello-serf"]
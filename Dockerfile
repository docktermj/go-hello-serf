FROM golang
WORKDIR /app
COPY distApp /app/
ENTRYPOINT ["./distApp"]

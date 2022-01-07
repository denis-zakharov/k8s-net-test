# Start by building the application.
FROM golang:1.17.5 as build

WORKDIR /go/src/app
RUN go mod init github.com/denis-zakharov/k8s-net-test
ADD model /go/src/app/model/
ADD pinger /go/src/app/
RUN go build -o /go/bin/app


# Now copy it into our base image.
FROM gcr.io/distroless/base-debian11
COPY --from=build /go/bin/app /
CMD ["/app"]
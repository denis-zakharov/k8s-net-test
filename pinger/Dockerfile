# Start by building the application.
FROM golang:1.17.5 as build

WORKDIR /go/src/app
ADD *.go /go/src/app
RUN GO111MODULE=off go build -o /go/bin/app


# Now copy it into our base image.
FROM gcr.io/distroless/base-debian11
COPY --from=build /go/bin/app /
CMD ["/app"]
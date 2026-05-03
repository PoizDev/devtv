FROM golang:1.24-alpine AS build

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /bin/devtv .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=build /bin/devtv /devtv

COPY conf.yaml  /conf.yaml
COPY log4go.json /log4go.json

COPY frontend/ /frontend/

COPY --from=build --chown=nonroot:nonroot /tmp /logs

ENV TZ=Europe/Istanbul

EXPOSE 2012

USER nonroot:nonroot

ENTRYPOINT ["/devtv"]

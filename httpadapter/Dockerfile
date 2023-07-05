from golang:1.20.3 as build-stage
workdir /build
copy go.mod go.sum .
run go mod download
copy . .
run CGO_ENABLED=0 go build -o app ./httpadapter/cmd/httpadapter

from alpine:3.18.2
run apk --no-cache add ca-certificates
workdir /app
copy --from=build-stage /build/app .
cmd ["/app/app"]

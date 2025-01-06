FROM golang:1.21.3-alpine as build

WORKDIR /app

COPY go.mod go.sum main.go ./

RUN go mod download

RUN go build -o alertapi

##############

FROM alpine:3.20.3

COPY --from=build /app /app

WORKDIR /app

# Copy the pre-compiled Go binary into the container
#COPY alertapi /app/

# Make the binary executable
RUN chmod +x /app/alertapi

CMD [ "/app/gin-alertapi" ]
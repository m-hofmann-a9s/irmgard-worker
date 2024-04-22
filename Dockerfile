FROM golang:1.22-bookworm as builder

ENV APP_HOME /workdir
WORKDIR $APP_HOME

COPY . .

RUN go mod download
RUN go mod verify
RUN go build -ldflags="-s -w" -o irmgardworker

FROM debian:bookworm-slim

ENV APP_HOME /workdir
RUN mkdir -p "$APP_HOME"
RUN mkdir -p /tmp/object_recognition
WORKDIR $APP_HOME


RUN apt-get update && apt-get -y install --no-install-recommends \
  python3 python3-numpy python3-opencv ca-certificates wget

RUN wget https://pjreddie.com/media/files/yolov3.weights

COPY yolo/* $APP_HOME


COPY --from=builder $APP_HOME/irmgardworker $APP_HOME

CMD ["./irmgardworker"]
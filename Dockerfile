FROM golang:1.22-bookworm

RUN apt-get update && apt-get install -y tesseract-ocr

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o main .

CMD ["./main"]
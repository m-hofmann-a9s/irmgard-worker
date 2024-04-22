package main

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
)

const (
	readBucket  = "images"
	writeBucket = "blurredimages"
	// FIXME image blurring script currently writes to static path
	outputFilePath = "/tmp/object_recognition/filtered-image.jpg"
)

func main() {
	log.Info().Msg("Starting irmgard-worker")

	minioEndpoint := os.Getenv("S3_ENDPOINT")
	minioAccessKeyID := os.Getenv("S3_ACCESSKEY")
	minioSecretAccessKey := os.Getenv("S3_SECRET")
	mqUsername := os.Getenv("MQ_USERNAME")
	mqPassword := os.Getenv("MQ_PASSWORD")
	mqEndpoint := os.Getenv("MQ_ENDPOINT")

	if minioEndpoint == "" || minioAccessKeyID == "" || minioSecretAccessKey == "" ||
		mqUsername == "" || mqPassword == "" || mqEndpoint == "" {
		log.Fatal().Msg("Missing required environment variables. You must set S3_ENDPOINT, S3_ACCESSKEY, S3_SECRET, " +
			"MQ_USERNAME, MQ_PASSWORD, MQ_ENDPOINT!")
	}

	minioClient, err := SetupObjectStorage(minioEndpoint, minioAccessKeyID, minioSecretAccessKey, writeBucket)
	failOnError(err, "Failed to set up object storage client")

	msgs, conn, err := SetupRabbitMq(mqUsername, mqPassword, mqEndpoint)
	defer conn.Close()

	if err != nil {
		failOnError(err, "Failed to set up rabbitmq connection")
	}

	forever := make(chan interface{})

	go func() {
		for d := range msgs {
			image := &Image{}
			err := json.Unmarshal(d.Body, &image)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to deserialize message %s", d.Body)
				// retry won't save us here, invalid message received
				_ = d.Ack(false)
				continue
			}

			log.Info().Msgf("Received a message: %v", image)

			filePath, err := downloadFile(minioClient, image)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to download image")
			}

			err = runBlurringScript(filePath)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to run image blurring script")
			}

			err = uploadResult(minioClient, image)
			if err != nil {
				log.Error().Err(err).Msg("Failed to put image to bucket")
				continue
			}

			err = deleteFile(minioClient, image)
			if err != nil {
				log.Error().Err(err).Msg("Failed to delete image")
				continue
			}

			// if file deletion doesn't work, the OS must be broken -> fail fast
			err = os.Remove(filePath)
			failOnError(err, "Failed to delete input file")
			err = os.Remove(outputFilePath)
			failOnError(err, "Failed to delete output file")

			err = d.Ack(false)
			if err != nil {
				log.Error().Err(err).Msg("Failed to ack mq message")
			}
		}
	}()

	<-forever
	log.Info().Msg("Shutting down irmgard-worker")
}

func runBlurringScript(filePath string) error {
	cmd := exec.Command("python3", "yolo_opencv.py", "--image", filePath, "--config", "yolov3.cfg",
		"--weights", "yolov3.weights", "--classes", "yolov3.txt")
	return cmd.Run()
}

// Image represents an image.
type Image struct {
	Id              int64  `json:"id"`
	Name            string `json:"name"`
	StorageLocation string `json:"storage_location"`
}

package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func SetupRabbitMq(mqUsername string, mqPassword string, mqEndpoint string) (<-chan amqp.Delivery, *amqp.Connection, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s/", mqUsername, mqPassword, mqEndpoint))
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open a channel")
		return nil, nil, err
	}

	q, err := ch.QueueDeclare(
		"images", // name
		true,     // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to declare queue")
		return nil, nil, err
	}

	msgs, err := ch.Consume(
		q.Name,
		"irmgard-worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to register consumer")
		return nil, nil, err
	}
	return msgs, conn, nil
}

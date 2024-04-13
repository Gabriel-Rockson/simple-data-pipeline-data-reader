package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"os"
)

type UserInteraction struct {
	EventTime    string
	EventType    string
	ProductId    string
	CategoryId   string
	CategoryCode string
	Brand        string
	Price        string
	UserId       string
	UserSession  string
}

func main() {
	file, err := os.Open("../data-source/2019-Oct.csv")

	if err != nil {
		panic(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file) // ? should I handle the error

	kafkaProducer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost"})
	if err != nil {
		panic(err)
	}
	defer kafkaProducer.Close()

	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = -1

	// Read away the heading
	_, err = csvReader.Read()
	if err != nil {
		return
	}

	continueReading := true
	for continueReading == true {
		data, err := csvReader.Read()
		if err != nil {
			return
		}
		if data == nil {
			continueReading = false
		}

		userInteraction := UserInteraction{
			EventTime:    data[0],
			EventType:    data[1],
			ProductId:    data[2],
			CategoryId:   data[3],
			CategoryCode: data[4],
			Brand:        data[5],
			Price:        data[6],
			UserId:       data[7],
			UserSession:  data[8],
		}

		// Delivery report logs
		go func() {
			for event := range kafkaProducer.Events() {
				switch ev := event.(type) {
				case *kafka.Message:
					if ev.TopicPartition.Error != nil {
						fmt.Printf("Delivery of message failed: %v\n", ev.TopicPartition.Error)
					} else {
						fmt.Printf("Delivery of message successful, message delivered to: %v\n", ev.TopicPartition)
					}
				}
			}
		}()

		// Publish the userInteraction into kafka
		var userInteractionBytes []byte
		userInteractionBytes, err = json.Marshal(userInteraction)
		if err != nil {
			return
		}

		topic := userInteraction.EventType // use the event types as topics
		err = kafkaProducer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          userInteractionBytes,
		}, nil)
		if err != nil {
			return
		}

		kafkaProducer.Flush(15 * 1000)
	}
}

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/afoninsky/noolite-go/noolite"
	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

// https://www.home-assistant.io/components/light.mqtt/

func main() {

	// 2do: viper envs
	// https://github.com/spf13/viper

	// handle interrupt signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	// open noolite connected device
	nooDevice, nooErr := noolite.CreateDevice()
	if nooErr != nil {
		log.Fatalln(nooErr)
	}
	defer nooDevice.Close()

	// connect to MQTT server
	cli := client.New(&client.Options{
		ErrorHandler: func(err error) {
			log.Fatalln(err)
		},
	})
	defer cli.Terminate()

	if err := cli.Connect(&client.ConnectOptions{
		Network:      "tcp",
		Address:      "localhost:1883",
		ClientID:     []byte(clientID),
		CleanSession: true,
		WillTopic:    []byte(willTopic),
		WillMessage:  []byte(willOfflineMessage),
		WillRetain:   true,
	}); err != nil {
		log.Fatalln(err)
	}

	server := &Server{noolite: &nooDevice, mqtt: cli}

	// listen for commands
	if err := cli.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				TopicFilter: []byte(fmt.Sprintf(setTopicPattern, "+", "+")),
				QoS:         mqtt.QoS0,
				Handler:     server.messageHandler,
			},
		},
	}); err != nil {
		log.Fatalln(err)
	}

	// send online message intp status topic
	if err := cli.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS0,
		Retain:    true,
		TopicName: []byte(willTopic),
		Message:   []byte(willOnlineMessage),
	}); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Ready to accept incoming connections")

	<-sigc
	// send offline message
	if err := cli.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS0,
		Retain:    true,
		TopicName: []byte(willTopic),
		Message:   []byte(willOfflineMessage),
	}); err != nil {
		log.Fatalln(err)
	}
	time.Sleep(time.Second)

	// disconnect
	if err := cli.Disconnect(); err != nil {
		log.Fatalln(err)
	}

}
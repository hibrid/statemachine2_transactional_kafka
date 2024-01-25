package simulator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hibrid/statemachine2_transactional_kafka/lib/app"
	"go.uber.org/zap"
)

var totalClientConstructed uint64

type Simulator struct {
	messages Messages
	SimulatorConfig
}

type SimulatorConfig struct {
	InputTopic       string
	OutputTopic      string
	PerSecond        float64
	NumberOfMessages int64
	Verbose          bool
	NumberOfClients  int
	Context          context.Context
	KafkaConfigMap   *kafka.ConfigMap
	kafkaProducer    *kafka.Producer
	WaitGroup        *sync.WaitGroup
}

func New(config SimulatorConfig) *Simulator {
	simulator := &Simulator{
		SimulatorConfig: config,
	}

	producer, err := kafka.NewProducer(simulator.KafkaConfigMap)
	if err != nil {
		panic(err)
	}

	simulator.kafkaProducer = producer

	return simulator
}

func (s *Simulator) LoadMessages(messages []byte) error {
	var sampleMessages Messages

	err := json.Unmarshal(messages, &sampleMessages.Messages)
	if err != nil {
		return err
	}

	s.messages = sampleMessages

	return nil
}

// nextTime can be replaced by using math.ExpFloat64() / rate
// but showcasing something from Donald Knuth's approach
// in §3.4.1 (D) of The Art of Computer Programming
func (s *Simulator) nextTime(rate rate) nextMessage {
	//don't need to seed for this but rand is deterministic in go
	//wanted to see some variance between runs
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	return nextMessage(-(math.Log(float64(1.0)-r1.Float64()) / rate.asFloat64()))
}

func (s *Simulator) sendIngressMessageEvent() {
	toppar := kafka.TopicPartition{Topic: &s.InputTopic, Partition: kafka.PartitionAny}
	message := s.messages.Messages[rand.Intn(len(s.messages.Messages))]
	producer := s.kafkaProducer
	messageBytes, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	err = producer.Produce(&kafka.Message{
		TopicPartition: toppar,
		Key:            []byte(message.TransactionID),
		Value:          messageBytes},
		nil)
	if err != nil {
		if err.(kafka.Error).Code() == kafka.ErrQueueFull {
			// Producer queue is full, skip this event.
			// A proper application should retry the Produce().
			app.Log.Warn(fmt.Sprintf("Generator: Warning: unable to produce event: %v", err))
			return
		}
		// Treat all other errors as fatal.
		panic(fmt.Sprintf("Generator: Failed to produce message: %v", err))
	}

}

func (s *Simulator) GenerateRandomMessages() {
	var messages Messages
	for i := int64(0); i < s.NumberOfMessages; i++ {
		messages.Messages = append(messages.Messages, generateRandomMessage())
	}
	s.messages = messages
}

func (s *Simulator) GenerateRandomMessage() Message {
	return Message{
		AccountID:     generateRandomAccountID(),
		TransactionID: generateRandomTransactionID(),
		EventType:     getRandomEventType().String(),
	}
}

func (s *Simulator) SetMessages(messages Messages) {
	s.messages = messages
}

// Simulate begins the order simulation
func (s *Simulator) Simulate() error {
	defer s.WaitGroup.Done()
	perSecond := s.PerSecond
	numberOfMessages := s.NumberOfMessages
	verbose := s.Verbose
	if perSecond == 0 {
		return errors.New("failed to send orders. perSecond should be > 0")
	}

	if numberOfMessages == 0 {
		return errors.New("failed to send orders. numberOfMessages should be > 0")
	}

	messages := s.messages

	rate := rate(1.0 / perSecond / 100.0)
	var sum float64
	var i int64

	for {
		nextMessage := s.nextTime(rate)
		//sanity check after the orders run
		sum += float64(nextMessage)
		//start tracking time
		testTime := time.Now()
		timer1 := time.NewTimer(nextMessage.asDuration() * time.Millisecond)
		i++
		if i > numberOfMessages {
			break
		}
		select {
		case <-timer1.C:
			//stop tracking time
			testTime2 := time.Now()
			//we will use this to see if the timer was accurate
			//to the interval from nextOrder
			diff := testTime2.Sub(testTime).Nanoseconds()
			message := messages.Messages[rand.Intn(len(messages.Messages))]

			//context.TODO is the same as context.Background but
			//I would like to extend the context to simulate cancellations later
			//the use of context.TODO() is correct and proper production code (google it)
			//ctx := context.TODO()
			s.sendIngressMessageEvent()

			if verbose {
				app.Log.Info("new message",
					zap.Any("message", message),
					zap.Duration("message interval", nextMessage.asDuration()*time.Millisecond),
					zap.Int64("calculate diff", diff))
			}
		case e := <-s.kafkaProducer.Events():
			// Handle delivery reports
			m, ok := e.(*kafka.Message)
			if !ok {
				app.Log.Info(fmt.Sprintf("Generator: Ignoring producer event %v", e))
				continue
			}

			if m.TopicPartition.Error != nil {
				app.Log.Error(fmt.Sprintf("Generator: Message delivery failed: %v: ignoring", m.TopicPartition))
				continue
			}
		case <-s.Context.Done():
			if verbose {
				app.Log.Info("context cancelled")
			}
			return nil
		}

	}
	app.Log.Info("Summary", zap.Int64("Total Orders Sent", i-1), zap.Float64("Average Order Interval", sum/float64(numberOfMessages)))
	s.kafkaProducer.Close()
	return nil
}

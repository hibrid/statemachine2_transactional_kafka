/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hibrid/statemachine2_transactional_kafka/lib/app"
	"github.com/hibrid/statemachine2_transactional_kafka/lib/simulator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var messageCount int64
var ratePerSecond float64
var verbose bool
var inputTopic, outputTopic, bootstrapServers, kafkaUsername, kafkaPassword string
var logsChan chan kafka.LogEvent

func logReader(wg *sync.WaitGroup, termChan chan bool) {
	defer wg.Done()

	for {
		select {
		case logEvent := <-logsChan:
			app.Log.Info(logEvent.String())
		case <-termChan:
			return
		}
	}
}

// simulatorCmd represents the simulator command
var simulatorCmd = &cobra.Command{
	Use:   "simulator",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("simulator called")
		globalWg.Add(1)
		go logReader(&globalWg, termChan)

		generator := simulator.New(simulator.SimulatorConfig{
			PerSecond:        ratePerSecond,
			NumberOfMessages: messageCount,
			Verbose:          verbose,
			NumberOfClients:  1,
			Context:          ctx,
			InputTopic:       inputTopic,
			OutputTopic:      outputTopic,
			KafkaConfigMap: &kafka.ConfigMap{
				"bootstrap.servers":      bootstrapServers,
				"client.id":              "generator",
				"enable.idempotence":     true,
				"go.logs.channel.enable": true,
				"go.logs.channel":        logsChan,
				"security.protocol":      "SASL_PLAINTEXT",
				"sasl.mechanisms":        "SCRAM-SHA-512",
				"sasl.username":          kafkaUsername,
				"sasl.password":          kafkaPassword,
			},
			WaitGroup: &globalWg,
		})
		fmt.Println(kafkaUsername, kafkaPassword)
		generator.GenerateRandomMessages()
		globalWg.Add(1)
		err := generator.Simulate()
		if err != nil {
			log.Error("failed to simulate", zap.Error(err))
		}
	},
}

func init() {
	rootCmd.AddCommand(simulatorCmd)
	viper.AutomaticEnv()
	simulatorCmd.Flags().Float64VarP(&ratePerSecond, "ratePerSecond", "r", 3.25,
		"number of orders per second. Default is 3.25")
	if viper.GetFloat64("ratePerSecond") != 0 {
		ratePerSecond = viper.GetFloat64("ratePerSecond")
	}
	simulatorCmd.Flags().Int64VarP(&messageCount, "messageCount", "o", 100,
		"total number of messages to send. Default is 100")
	if viper.GetInt64("messageCount") != 0 {
		messageCount = viper.GetInt64("messageCount")
	}
	simulatorCmd.Flags().BoolVarP(&verbose, "VERBOSE", "v", false, "show verbose output")

	verbose = true //viper.GetBool("VERBOSE")

	simulatorCmd.Flags().StringVarP(&inputTopic, "INPUT_TOPIC", "i", "input", "input topic")
	if viper.GetString("INPUT_TOPIC") != "" {
		inputTopic = viper.GetString("INPUT_TOPIC")
	}

	simulatorCmd.Flags().StringVarP(&outputTopic, "OUTPUT_TOPIC", "u", "output", "output topic")
	if viper.GetString("OUTPUT_TOPIC") != "" {
		outputTopic = viper.GetString("OUTPUT_TOPIC")
	}

	simulatorCmd.Flags().StringVarP(&bootstrapServers, "KAFKA_BROKERS", "b", "localhost:9092", "bootstrap servers")
	if viper.GetString("KAFKA_BROKERS") != "" {
		bootstrapServers = viper.GetString("KAFKA_BROKERS")
	}

	simulatorCmd.Flags().StringVarP(&kafkaUsername, "KAFKA_USERNAME", "k", "kafka", "kafka username")
	if viper.GetString("KAFKA_USERNAME") != "" {
		kafkaUsername = viper.GetString("KAFKA_USERNAME")
	}

	simulatorCmd.Flags().StringVarP(&kafkaPassword, "KAFKA_PASSWORD", "p", "kafka", "kafka password")
	if viper.GetString("KAFKA_PASSWORD") != "" {
		kafkaPassword = viper.GetString("KAFKA_PASSWORD")
	}

}

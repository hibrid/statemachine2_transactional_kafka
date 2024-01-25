/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"embed"

	"github.com/hibrid/statemachine2_transactional_kafka/cmd"
)

//go:embed staticassets
var dataBox embed.FS

func main() {
	cmd.Execute(dataBox)
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_API(t *testing.T) {
	var n uint
	n = 3
	configs := make([]ProjectConfig, n)
	addresses := []string{"127.0.0.1:1025", "127.0.0.1:1026", "127.0.0.1:1027"}
	configs[0] = ProjectConfig{
		Role:          []string{"CN", "HR"},
		Index:         0,
		Addresses:     addresses,
		HRSecretTable: map[uint]string{123: "12345678"},
		ID:            123,
		Secret:        "12345678",
		Num:           n,
	}
	configs[1] = ProjectConfig{
		Role:          []string{"CN", "AP"},
		Index:         1,
		Addresses:     addresses,
		HRSecretTable: map[uint]string{123: "12345678"},
		ID:            123,
		Secret:        "12345678",
		Num:           n,
	}
	configs[2] = ProjectConfig{
		Role:          []string{"CN", "AP"},
		Index:         2,
		Addresses:     addresses,
		HRSecretTable: map[uint]string{123: "12345678"},
		ID:            123,
		Secret:        "12345678",
		Num:           n,
	}

	for i, config := range configs {
		path := fmt.Sprintf("./config_%d.json", i)
		WriteJSON(config, path)
		config_read := ReadJSON(path)
		require.Equal(t, config, config_read)
	}

}

func WriteJSON(config ProjectConfig, configPath string) {
	// create file
	filePtr, err := os.Create(configPath)
	if err != nil {
		fmt.Println("file create fail", err.Error())
		return
	}
	defer filePtr.Close()

	// create json encoder
	config_buf, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println("json.MarshalIndent fail", err.Error())
		return
	}
	_, err = filePtr.Write(config_buf)
	if err != nil {
		fmt.Println("encode fail", err.Error())
	} else {
		fmt.Println("encode success")
	}
}

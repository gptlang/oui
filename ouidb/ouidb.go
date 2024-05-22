package ouidb

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type OUIEntry struct {
	Prefix       string
	Manufacturer string
}

var ouiData map[string]string

func loadDatabase() error {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	ouiFile := filepath.Join(dataDir, "oui_data.csv")

	file, err := os.Open(ouiFile)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	ouiData = make(map[string]string)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		ouiData[record[0]] = record[1]
	}
	return nil
}

func UpdateDatabase() {
	fmt.Println("Fetching latest OUI database from IEEE...")

	url := "http://standards-oui.ieee.org/oui/oui.csv"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error downloading the OUI database:", err)
		return
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	reader.Comma = ','

	ouiData = make(map[string]string)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error parsing the OUI database:", err)
			return
		}
		ouiData[record[1]] = record[2]
	}

	dataDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Error getting config directory:", err)
		return
	}
	ouiFile := filepath.Join(dataDir, "oui_data.csv")

	file, err := os.Create(ouiFile)
	if err != nil {
		fmt.Println("Error creating OUI database file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	for k, v := range ouiData {
		err := writer.Write([]string{k, v})
		if err != nil {
			fmt.Println("Error writing to OUI database file:", err)
			return
		}
	}
	writer.Flush()

	fmt.Println("OUI database updated successfully.")
}

func Lookup(mac string) (string, error) {
	// IEEE doc uses all caps, so uppercase the input
	mac = strings.ToUpper(mac)

	// Use regex to extract the first 3 octets (ignoring separators)
	re := regexp.MustCompile(`([0-9A-F]{2}[-:]?){3}`)
	match := re.FindString(mac)
	r := strings.NewReplacer(":", "", "-", "")
	match = r.Replace(match)

	if match == "" {
		return "", errors.New("Invalid OUI supplied.")
	}

	err := loadDatabase()
	if err != nil {
		UpdateDatabase()
		return "", err
	}

	manufacturer, ok := ouiData[match]
	if !ok {
		return "", errors.New("Manufacturer not found for OUI.")
	}
	return manufacturer, nil
}

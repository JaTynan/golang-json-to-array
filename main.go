package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	// Load JSON data from a file
	fileName := "response.json" // Replace with your actual file name
	fmt.Printf("Attempting to open json file:%v\n", fileName)
	jsonData, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	fmt.Printf("Successfully opened json file:%v\n", fileName)

	// Unmarshal JSON data into a generic interface
	fmt.Printf("Attempting to unmarshal JSON file:%v\n", fileName)
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	fmt.Printf("Successfully unmarshaled json file:%v\n", fileName)

	// Use a type switch to handle different JSON structures
	fmt.Printf("Attempting to process into a series of tables.\n")
	tableRootName := "root_table" // Replace this
	switch v := data.(type) {
	case map[string]interface{}:
		fmt.Printf("We have found the root table, eg. Customer.\n")
		processJSON(tableRootName, v, "TST") // Start with a root table and no parent GUID
	case []interface{}:
		for i, item := range v {
			if obj, ok := item.(map[string]interface{}); ok {
				tableName := fmt.Sprintf("root_table_%d", i+1)
				processJSON(tableName, obj, "TST") // Pass empty string for root as it has no parent
			} else {
				log.Fatalf("Unexpected JSON structure in array at index %d", i)
			}
		}
	default:
		log.Fatalf("Unexpected JSON structure")
	}
}

func GenerateUUID() (string, error) {
	uuid := make([]byte, 16)

	// Read 16 random bytes
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		return "", err
	}

	// Set the version to 4 (randomly generated UUID)
	uuid[6] = (uuid[6] & 0x0f) | 0x40

	// Set the variant to 2
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	// Return the UUID string
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// processJSON generates SQL queries for the provided JSON data
func processJSON(tableName string, data map[string]interface{}, parentGUID string) {
	// Generate SQL table creation and insert queries
	var createTableQueries, insertQueries []string
	generateSQL(tableName, data, &createTableQueries, &insertQueries, parentGUID)

	// Print the generated SQL queries
	for _, query := range createTableQueries {
		fmt.Println(query)
	}
	for _, query := range insertQueries {
		fmt.Println(query)
	}
}

// generateSQL recursively generates CREATE TABLE and INSERT queries
func generateSQL(tableName string, data map[string]interface{}, createTableQueries *[]string, insertQueries *[]string, parentGUID string) {
	columns := []string{"guid"}
	values := []string{}

	// Retrieve the current object's GUID, or generate one if missing
	var err error
	currentGUID := ""
	if guid, exists := data["guid"]; exists && guid != "" {
		currentGUID = guid.(string)
	} else {
		// Generate a new GUID if it does not exist or is empty
		currentGUID, err = GenerateUUID()
		if err != nil {
			fmt.Println("Error: ", err)
		}
	}
	values = append(values, fmt.Sprintf("'%s'", currentGUID))

	if parentGUID != "" {
		// Add foreign key reference to the parent table if parentGUID is provided
		parentColumn := tableName + "_parent_guid"
		columns = append(columns, parentColumn)
		values = append(values, fmt.Sprintf("'%s'", parentGUID))
	}

	for key, value := range data {
		if key == "guid" {
			continue // Skip the GUID column since it's already handled
		}

		columnName := key
		var valueStr string

		switch v := value.(type) {
		case string:
			if v == "" {
				valueStr = "NULL"
			} else {
				valueStr = fmt.Sprintf("'%s'", v)
			}
			columns = append(columns, columnName)
			values = append(values, valueStr)
		case float64:
			valueStr = fmt.Sprintf("%f", v)
			columns = append(columns, columnName)
			values = append(values, valueStr)
		case bool:
			valueStr = fmt.Sprintf("%t", v)
			columns = append(columns, columnName)
			values = append(values, valueStr)
		case nil:
			// Handle explicit null values in JSON
			valueStr = "NULL"
			columns = append(columns, columnName)
			values = append(values, valueStr)
		case map[string]interface{}:
			// Create a new table for nested objects
			childTableName := tableName + "_" + columnName
			generateSQL(childTableName, v, createTableQueries, insertQueries, currentGUID)
		case []interface{}:
			// Handle arrays (assuming each element is a primitive or object)
			for _, item := range v {
				if obj, ok := item.(map[string]interface{}); ok {
					childTableName := tableName + "_" + columnName
					generateSQL(childTableName, obj, createTableQueries, insertQueries, currentGUID)
				} else {
					if item == nil || item == "" {
						valueStr = "NULL"
					} else {
						valueStr = fmt.Sprintf("'%v'", item)
					}
					columns = append(columns, columnName)
					values = append(values, valueStr)
				}
			}
		}
	}

	// Generate the CREATE TABLE query
	createTableQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, strings.Join(columns, " TEXT, ")+" TEXT")
	*createTableQueries = append(*createTableQueries, createTableQuery)

	// Generate the INSERT query
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", tableName, strings.Join(columns, ", "), strings.Join(values, ", "))
	*insertQueries = append(*insertQueries, insertQuery)
}

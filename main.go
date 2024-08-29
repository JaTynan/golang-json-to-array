package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type UnleashedAPICredentials struct {
	APIID  string `json:"api_id"`
	APIKey string `json:"api_key"`
}

func main() {

	// Request data from Unleashed Sandbox API
	responseHeaderContent, responseBodyContent := UnleashedMakeRequest("Customers", "", "")
	fmt.Printf("\nRaw Response Headers:\n %v\n", responseHeaderContent)

	// Save JSON response as a file
	responseFullFileName := "response.json"
	err := os.WriteFile(responseFullFileName, responseBodyContent, 0644)
	if err != nil {
		log.Fatalf("\nFailed to save the response into a file: %v", err)
	}
	fmt.Printf("Successfully created the file.\n")

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

	/* POSTGRESQL TESTING */
	databasePassword := PostgresqlCredentialObtain()

	// Connect to the PostgreSQL database.
	databaseConn := PostgresqlConnectionMake(databasePassword)

	// Testing the PosgreSQL database connection.
	var greeting string
	err = databaseConn.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(greeting)

	testSchemaName := "test"
	PostgresqlClearSchema(testSchemaName)
	PostgresqlCreateSchema(testSchemaName)

	// Use a type switch to handle different JSON structures
	var createTableQueries []string
	var insertQueries []string
	fmt.Printf("Attempting to process into a series of tables.\n")
	tableRootName := "root_table" // Replace this
	switch v := data.(type) {
	case map[string]interface{}:
		fmt.Printf("We have found the root table.\n")
		createTableQueries, insertQueries = processJSON(testSchemaName+".", tableRootName, v, "") // Start with a root table and no parent GUID
	case []interface{}:
		for i, item := range v {
			fmt.Printf("We did not find the root table.\n")
			if obj, ok := item.(map[string]interface{}); ok {
				tableName := fmt.Sprintf("root_table_%d", i+1)
				createTableQueries, insertQueries = processJSON(testSchemaName+".", tableName, obj, "") // Pass empty string for root as it has no parent
			} else {
				log.Fatalf("Unexpected JSON structure in array at index %d", i)
			}
		}
	default:
		log.Fatalf("Unexpected JSON structure")
	}
	for i := 0; i < len(createTableQueries); i++ {
		_, err := databaseConn.Exec(context.Background(), createTableQueries[i])
		if err != nil {
			fmt.Printf("\nFailed current PostgreSQL query:\n\t%v\n", createTableQueries[i])
			log.Fatalf("Failed to execute the table creation query:\n\t%v\n", err)
		}
	}
	for i := 0; i < len(insertQueries); i++ {
		_, err := databaseConn.Exec(context.Background(), insertQueries[i])
		if err != nil {
			fmt.Printf("\nFailed current PostgreSQL query:\n\t%v\n", insertQueries[i])
			log.Fatalf("Failed to execute the insert query:\n\t%v\n", err)
		}
	}
	databaseConn.Close(context.Background())
}

func SignatureEncrypt(message string, key string) string {

	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	signature := h.Sum(nil)
	base64Signature := base64.StdEncoding.EncodeToString(signature)
	key = "" // Clear the key, tiny bit more secure.
	fmt.Printf("\nOur generated signature: %v", base64Signature)

	return base64Signature
}

// Takes a resource name and returns a complete URL.
func UnleashedURLBuilder(resourceType string, resourceParamters string, resourceFilters string) (string, string) {
	// Test API connection to Unleashed. "https://go.unleashedsoftware.com/v2"
	// Most Unleashed requests can be filtered by &modifiedSince=YYYY-MM-DD
	// We can also change the pageSize returned, default 200. &pageSize=1000

	// resourceParamters for multiple can be entered using &
	if resourceType == "" {
		fmt.Printf("\nNo resource type provided for Unleashed URL.")
	}

	unleashedWebsiteURL := "https://api.unleashedsoftware.com"
	unleashedWebsiteResource := "/" + resourceType + "?"
	unleashedWebsiteParameters := resourceParamters //!!!!!
	unleashedWebsiteFilters := resourceFilters

	unleashedSignatureMessage := unleashedWebsiteParameters + unleashedWebsiteFilters
	unleashedAPIFullURL := unleashedWebsiteURL + unleashedWebsiteResource + unleashedWebsiteParameters + unleashedWebsiteFilters

	return unleashedAPIFullURL, unleashedSignatureMessage
}

func UnleashedAPICredentialObtain() (string, string) {
	/* UNLEASHED API TESTING */
	// https://apidocs.unleashedsoftware.com/
	unleashedAPICredentialFilename := "unleashed_api_key.json"
	unleashedAPICredentialsJson, err := os.Open(unleashedAPICredentialFilename)
	if err != nil {
		log.Fatalf("\nWe could not find the credential file: %v", err)
	}
	fmt.Printf("\nWe opened %v !", unleashedAPICredentialFilename)
	defer unleashedAPICredentialsJson.Close()

	byteValue, _ := io.ReadAll(unleashedAPICredentialsJson)
	var unleashedAPICredentials UnleashedAPICredentials
	json.Unmarshal([]byte(byteValue), &unleashedAPICredentials)
	fmt.Printf("\nWe have found credentials. %v, %v", unleashedAPICredentials.APIID, unleashedAPICredentials.APIKey)

	return unleashedAPICredentials.APIID, unleashedAPICredentials.APIKey
}

func UnleashedMakeRequest(resourceType string, resourceParam string, resourceFilt string) (http.Header, []byte) {

	unleashedRequestURL, signatureMessage := UnleashedURLBuilder(resourceType, resourceParam, resourceFilt)
	unleashedAPIID, unleashedAPIKey := UnleashedAPICredentialObtain()
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	// Create the new HTTP GET request.
	request, err := http.NewRequest("GET", unleashedRequestURL, nil)
	if err != nil {
		log.Fatalf("\nError creating request: %v", err)
	}

	// Adding custom headers for Unleashed API
	// for Content-Type and Accept we can use application/xml or application/json
	// Unleashed returns JSON in unicode
	// Unleashed return XML as utf-8
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("api-auth-id", unleashedAPIID)
	request.Header.Set("api-auth-signature", SignatureEncrypt(signatureMessage, unleashedAPIKey))
	request.Header.Set("client-type", "Sandbox-Billson's Beverages Pty Ltd (Administrators Appointed)/james_tynan_integration_testing")

	response, err := client.Do(request)
	if err != nil {
		log.Fatalf("\nError making request: %v", err)
	}
	defer response.Body.Close()
	// Check the response status code returned from the request.
	unleashedStatusCodeMap := map[int]string{
		200: "OK: Operation was successful.",
		400: "Bad Request: The request was not in a correct form, or the posted data failed a validation test. Check the error returned to see what was wrong with the request.",
		403: "Forbidden: Method authentication failed.",
		404: "Not Found: Endpoint does not exist (eg using /SalesOrder instead of /SalesOrders).",
		405: "Not Allowed: The method used is not allowed (eg PUT, DELETE) or is missing a required parameter (eg POST requires an /{id} parameter).",
		500: "Internal Server Error: The object passed to the API could not be parsed.",
	}
	responseReturnCode := unleashedStatusCodeMap[response.StatusCode]
	fmt.Printf("\nAPI Reponse Status Code:: %v", responseReturnCode)

	responseHeader := response.Header
	// Read the response body from the request.
	responseBodyContent, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("\nFailed to read the response body: %v", err)
	}
	//fmt.Printf("\nRaw Response Status: %v", response.Status)
	//fmt.Printf("\nRaw Response Headers: %v", responseHeader)
	//fmt.Printf("\nRaw Response Body: %v", string(responseBodyContent))

	return responseHeader, responseBodyContent
}

func PostgresqlCredentialObtain() string {
	// Get our postgreSQL database password from our environment variable
	// Create your own environment variable storing your PostgreSQL password, eg. POSTGRESQL_PASSWORD=123456
	// Once set you may have to restart your IDE to update your variables.
	environmentVariableNameHoldingPassword := "POSTGRESQL_PASSWORD"
	databasePassword := os.Getenv(environmentVariableNameHoldingPassword)
	if databasePassword == "" {
		fmt.Printf("\n%v, is not set.\n", environmentVariableNameHoldingPassword)
	} else {
		fmt.Printf("\n%v, has been found.\n", environmentVariableNameHoldingPassword)
	}
	return databasePassword
}

func PostgresqlConnectionMake(databasePassword string) *pgx.Conn {
	// Connect to the PostgreSQL database.
	databaseConnStr := "postgres://postgres:" + databasePassword + "@localhost:5432/postgres"
	databaseConn, err := pgx.Connect(context.Background(), databaseConnStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	return databaseConn
}
func PostgresqlClearSchema(schemaName string) {
	databaseConn := PostgresqlConnectionMake(PostgresqlCredentialObtain())

	// Wipe schema before creating, for testing only
	createSchemaWipeQuery := "DROP SCHEMA IF EXISTS " + schemaName + " CASCADE;"
	_, err := databaseConn.Exec(context.Background(), createSchemaWipeQuery)
	if err != nil {
		log.Fatal(err)
	}
	databaseConn.Close(context.Background())
}

func PostgresqlCreateSchema(schemaName string) {
	databaseConn := PostgresqlConnectionMake(PostgresqlCredentialObtain())
	// Create schema if it does not exist
	createSchemaQuery := "CREATE SCHEMA IF NOT EXISTS " + schemaName + ";"
	_, err := databaseConn.Exec(context.Background(), createSchemaQuery)
	if err != nil {
		log.Fatal(err)
	}
	databaseConn.Close(context.Background())
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
func processJSON(schemaName string, tableName string, data map[string]interface{}, parentGUID string) ([]string, []string) {
	// Generate SQL table creation and insert queries
	var createTableQueries, insertQueries []string
	generateSQL(schemaName, tableName, data, &createTableQueries, &insertQueries, parentGUID)

	// Print the generated SQL queries
	for _, query := range createTableQueries {
		fmt.Println(query)
	}
	for _, query := range insertQueries {
		fmt.Println(query)
	}

	return createTableQueries, insertQueries
}

// generateSQL recursively generates CREATE TABLE and INSERT queries
func generateSQL(schemaName string, tableName string, data map[string]interface{}, createTableQueries *[]string, insertQueries *[]string, parentGUID string) {
	columns := []string{"current_guid"}
	values := []string{}

	// Retrieve the current object's GUID, or generate one if missing
	var err error
	currentGUID := ""
	if guid, exists := data["current_guid"]; exists && guid != "" {
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
		if key == "Guid" {
			continue // Skip the GUID column since it's already handled
		}

		columnName := key
		var valueStr string

		switch v := value.(type) {
		case string:
			if v == "" {
				valueStr = "NULL"
			} else {
				v = strings.Replace(v, "'", "''", -1)
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
			generateSQL(schemaName, childTableName, v, createTableQueries, insertQueries, currentGUID)
		case []interface{}:
			// Handle arrays (assuming each element is a primitive or object)
			for _, item := range v {
				if obj, ok := item.(map[string]interface{}); ok {
					childTableName := tableName + "_" + columnName
					generateSQL(schemaName, childTableName, obj, createTableQueries, insertQueries, currentGUID)
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
	createTableQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s%s (%s);", schemaName, tableName, strings.Join(columns, " TEXT, ")+" TEXT")
	*createTableQueries = append(*createTableQueries, createTableQuery)

	// Generate the INSERT query
	insertQuery := fmt.Sprintf("INSERT INTO %s%s (%s) VALUES (%s);", schemaName, tableName, strings.Join(columns, ", "), strings.Join(values, ", "))
	*insertQueries = append(*insertQueries, insertQuery)

}

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var uniqueKeys map[string]bool

func fetchData(url, method string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Example function to remove HTML tags from a string
func removeHTMLTags(body []byte) []byte {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAll(body, []byte{})
}

func extractKeys(data interface{}, prefix string) {
	switch v := data.(type) {
	case map[string]interface{}:
		extractMapKeys(v, prefix)
	case []interface{}:
		for _, item := range v {
			// fmt.Println("Index of object:", i)
			extractKeys(item, prefix)
		}
	default:
		uniqueKeys[prefix] = true
	}
}

func extractMapKeys(data map[string]interface{}, prefix string) {
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			extractMapKeys(v, fmt.Sprintf("%s%s.", prefix, key))
		case []interface{}:
			extractKeys(v, fmt.Sprintf("%s%s.", prefix, key))
		default:

			uniqueKeys[fmt.Sprintf("%s%s", prefix, key)] = true
		}
	}
}

// accessing values by passing keys

func getField(data interface{}, field string) string {
	fields := strings.Split(field, ".")
	obj := data
	for _, f := range fields {
		switch val := obj.(type) {
		case map[string]interface{}:
			if v, ok := val[f]; ok {
				obj = v
			} else {
				return "nill"
			}
		case []interface{}:
			var result []string
			for _, item := range val {
				switch nested := item.(type) {
				case string:
					result = append(result, nested)
				case map[string]interface{}:
					if v, ok := nested[f]; ok {
						if strVal, isString := v.(string); isString {
							result = append(result, strVal)
						} else {
							result = append(result, fmt.Sprintf("%v", v))
						}
					}
				}
			}
			// obj = result
			obj = strings.Join(result, "||") // Join elements of the slice

		default:
			return fmt.Sprintf("%v", obj)

		}
	}
	return fmt.Sprintf("%v", obj)
}

func writeCSV(columnNames []string, rows [][]string) error {
	file, err := os.Create("Sample_Data.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Create a new slice to hold modified column names for writing CSV header
	modifiedColumnNames := make([]string, len(columnNames))

	// Modify column names by replacing dots with underscores
	for i, name := range columnNames {
		modifiedColumnNames[i] = strings.ReplaceAll(name, ".", "_")
	}

	// Write column names
	if err := writer.Write(modifiedColumnNames); err != nil {
		return err
	}

	// Write rows
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func writeCSV2(columnNames []string, rows [][]string) error {
	file, err := os.Create("Sample_Data.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write column names to CSV
	if err := writer.Write(columnNames); err != nil {
		return err
	}

	// Write rows to CSV
	for _, r := range rows {
		maxElements := findMaxElements(r)
		for i := 0; i < maxElements; i++ {
			newRow := make([]string, len(columnNames))
			for j, v := range r {
				if strings.Contains(v, "||") {
					elements := strings.Split(v, "||")
					if len(elements) > i {
						newRow[j] = strings.TrimSpace(elements[i])
					} else {
						newRow[j] = "nil"
					}
				} else {
					newRow[j] = v
				}
			}
			if err := writer.Write(newRow); err != nil {
				return err
			}
		}
	}

	return nil
}

// findMaxElements finds the maximum number of elements among columns with pipe-separated values
func findMaxElements(row []string) int {
	max := 0
	for _, v := range row {
		elements := strings.Split(v, "||")
		if len(elements) > max {
			max = len(elements)
		}
	}
	return max
}
func main() {
	url := "https://api.zippopotam.us/us/33162"
	method := "GET"

	body, err := fetchData(url, method)
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return
	}

	// Check the type of the root element
	var data interface{} // Use empty interface
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Error unmarshaling data:", err)
		return
	}
	// Initialize uniqueKeys map
	uniqueKeys = make(map[string]bool)
	extractKeys(data, "")

	// Create a slice to store column names
	columnNames := make([]string, 0, len(uniqueKeys))

	// Add keys to columnNames slice
	for key := range uniqueKeys {
		columnNames = append(columnNames, key)
	}
	fmt.Println("count on unique keys:", len(uniqueKeys))

	switch data.(type) {
	case map[string]interface{}:
		fmt.Println("this is  map[string]interface{}")

		dataMap, ok := data.(map[string]interface{})
		if !ok {
			fmt.Println("Data is not in the expected format")
			return
		}
		// Fetch the prefix from the keys of dataMap dynamically
		var prefix string
		var prefixFound bool

		// Iterate over the keys of dataMap
		for key, value := range dataMap {
			// Check if the value associated with the key is an array
			if _, isArray := value.([]interface{}); isArray {
				// If it's an array, set it as the prefix key
				prefix = key + "."
				fmt.Println(prefix)
				prefixFound = true
				break
			}
		}

		// Check if the prefix is found
		if !prefixFound {
			fmt.Println("Prefix not found in response")
			return
		}

		// Remove the prefix from column names
		for i, columnName := range columnNames {
			columnNames[i] = strings.TrimPrefix(columnName, prefix)
		}

		for _, columnName := range columnNames {
			fmt.Println(columnName)
		}
		fmt.Println("Total number of columns:", len(columnNames))
		// Iterate over the keys of dataMap to dynamically find the desired field
		var usersInterface interface{}
		var fieldFound bool
		for _, value := range dataMap {
			if field, ok := value.([]interface{}); ok {
				usersInterface = field
				fieldFound = true
				break
			}
		}
		fmt.Println("Values")

		// Check if the desired field is found
		if !fieldFound {
			fmt.Println("Field not found in response")
			return
		}
		var rows [][]string
		// Use the dynamically fetched field value
		users, ok := usersInterface.([]interface{})
		if !ok {
			fmt.Println("Field is not an array")
			return
		}
		// Iterate over each user object
		for _, user := range users {
			userData, ok := user.(map[string]interface{})
			if !ok {
				fmt.Println("Invalid user data")
				continue
			}
			row := make([]string, len(columnNames))
			// Retrieve values for each column and add to row
			for i, columnName := range columnNames {
				value := getField(userData, columnName)
				// Check if the value contains a comma
				row[i] = fmt.Sprintf("%v", value)
			}
			rows = append(rows, row)
		}
		// rows = append(rows, row)
		if err := writeCSV2(columnNames, rows); err != nil {
			fmt.Println("Error writing CSV:", err)
			return
		}

	case []interface{}:
		// Handle if the root element is an array
		fmt.Println("This is an array")
		var rows [][]string
		for _, item := range data.([]interface{}) {
			switch item := item.(type) {
			case string:
				fmt.Println("type of data is string", item)
				// If the item is a string, create a row with just that string
				row := []string{item}
				rows = append(rows, row)
			default:
				userData, ok := item.(map[string]interface{})
				if !ok {
					fmt.Println("Invalid user data")
					continue
				}
				row := make([]string, len(columnNames))
				for i, columnName := range columnNames {
					value := getField(userData, columnName)
					row[i] = fmt.Sprintf("%v", value)
					fmt.Println("values of rows", value)
				}
				rows = append(rows, row)
			}

			// Write to CSV
			if err := writeCSV(columnNames, rows); err != nil {
				fmt.Println("Error writing CSV:", err)
				return
			}
		}
	default:
		fmt.Println("Unexpected type of data received from API")
	}

	fmt.Println("Code executed successfully")
}

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
)

// StringProperties holds the computed properties of a string....
type StringProperties struct {
	Length                int          `json:"length"`
	IsPalindrome          bool         `json:"is_palindrome"`
	UniqueCharacters      int          `json:"unique_characters"`
	WordCount             int          `json:"word_count"`
	SHA256Hash            string       `json:"sha256_hash"`
	CharacterFrequencyMap map[rune]int `json:"character_frequency_map"`
}

// AnalyzedString represents a stored string with its properties....
type AnalyzedString struct {
	ID         string           `json:"id"` // SHA-256 hash...
	Value      string           `json:"value"`
	Properties StringProperties `json:"properties"`
	CreatedAt  time.Time        `json:"created_at"`
}

// Store for analyzed strings (in-memory)....
var (
	stringStore = make(map[string]AnalyzedString)
	mu          sync.RWMutex // Mutex to protect stringStore...
)

// Request body for POST /strings....
type CreateStringRequest struct {
	Value string `json:"value" binding:"required"`
}

// Response body for GET /strings with filtering....
type FilteredStringsResponse struct {
	Data           []AnalyzedString       `json:"data"`
	Count          int                    `json:"count"`
	FiltersApplied map[string]interface{} `json:"filters_applied"`
}

// Response body for GET /strings/filter-by-natural-language....
type NaturalLanguageFilterResponse struct {
	Data             []AnalyzedString     `json:"data"`
	Count            int                  `json:"count"`
	InterpretedQuery NaturalLanguageQuery `json:"interpreted_query"`
}

type NaturalLanguageQuery struct {
	Original      string                 `json:"original"`
	ParsedFilters map[string]interface{} `json:"parsed_filters"`
}

// calculateLength returns the number of characters in a string....
func calculateLength(s string) int {
	return len([]rune(s))
}

// isPalindrome checks if a string is a palindrome (case-insensitive, ignoring non-alphanumeric characters)....\
func isPalindrome(s string) bool {
	var filteredRunes []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			filteredRunes = append(filteredRunes, r)
		}
	}

	for i, j := 0, len(filteredRunes)-1; i < j; i, j = i+1, j-1 {
		if filteredRunes[i] != filteredRunes[j] {
			return false
		}
	}
	return true
}

// countUniqueCharacters returns the count of distinct characters in a string....
func countUniqueCharacters(s string) int {
	seen := make(map[rune]bool)
	for _, char := range s {
		seen[char] = true
	}
	return len(seen)
}

// countWords returns the number of words separated by whitespace....
func countWords(s string) int {
	if s == "" {
		return 0
	}
	// Split by one or more whitespace characters....
	words := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(s), -1)
	count := 0
	for _, word := range words {
		if word != "" {
			count++
		}
	}
	return count
}

// generateSHA256Hash computes the SHA-256 hash of a string....
func generateSHA256Hash(s string) string {
	hasher := sha256.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

// getCharacterFrequencyMap returns a map of character counts....
func getCharacterFrequencyMap(s string) map[rune]int {
	freqMap := make(map[rune]int)
	for _, char := range s {
		freqMap[char]++
	}
	return freqMap
}

// analyzeString computes all properties for a given string....
func analyzeString(value string) StringProperties {
	hash := generateSHA256Hash(value)
	return StringProperties{
		Length:                calculateLength(value),
		IsPalindrome:          isPalindrome(value),
		UniqueCharacters:      countUniqueCharacters(value),
		WordCount:             countWords(value),
		SHA256Hash:            hash,
		CharacterFrequencyMap: getCharacterFrequencyMap(value),
	}
}

// CreateStringHandler handles POST /strings....
func CreateStringHandler(c *gin.Context) {
	var req CreateStringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Check for missing "value" field....
		if strings.Contains(err.Error(), "Field validation for 'Value' failed on the 'required' tag") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required field: 'value'"})
			return
		}
		// Check for invalid data type for "value" (e.g., if it's not a string)....\
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid data type for 'value' field, expected string"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mu.RLock()
	existingHash := generateSHA256Hash(req.Value)
	if _, exists := stringStore[existingHash]; exists {
		mu.RUnlock()
		c.JSON(http.StatusConflict, gin.H{"error": "String already exists in the system"})
		return
	}
	mu.RUnlock()

	properties := analyzeString(req.Value)

	newAnalyzedString := AnalyzedString{
		ID:         properties.SHA256Hash,
		Value:      req.Value,
		Properties: properties,
		CreatedAt:  time.Now().UTC(),
	}

	mu.Lock()
	stringStore[newAnalyzedString.ID] = newAnalyzedString
	mu.Unlock()

	c.JSON(http.StatusCreated, newAnalyzedString)
}

// GetSpecificStringHandler handles GET /strings/{string_value}....
func GetSpecificStringHandler(c *gin.Context) {
	stringValue := c.Param("string_value")

	mu.RLock()
	defer mu.RUnlock()

	// Try to find by direct hash....
	if analyzedStr, exists := stringStore[stringValue]; exists {
		c.JSON(http.StatusOK, analyzedStr)
		return
	}

	// If not found by direct hash, try to hash the param and find....
	hashedValue := generateSHA256Hash(stringValue)
	if analyzedStr, exists := stringStore[hashedValue]; exists {
		c.JSON(http.StatusOK, analyzedStr)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "String not found in the system"})
}

// GetFilteredStringsHandler handles GET /strings with filtering....
func GetFilteredStringsHandler(c *gin.Context) {
	filters := make(map[string]interface{})
	filteredStrings := make([]AnalyzedString, 0) // Initialize as empty slice, not nil

	mu.RLock()
	defer mu.RUnlock()

	for _, str := range stringStore {
		match := true

		// is_palindrome filter....
		if param := c.Query("is_palindrome"); param != "" {
			val, err := strconv.ParseBool(param)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid value for 'is_palindrome', must be boolean (true/false)"})
				return
			}
			filters["is_palindrome"] = val
			if str.Properties.IsPalindrome != val {
				match = false
			}
		}

		// min_length filter....
		if param := c.Query("min_length"); param != "" {
			val, err := strconv.Atoi(param)
			if err != nil || val < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid value for 'min_length', must be a non-negative integer"})
				return
			}
			filters["min_length"] = val
			if str.Properties.Length < val {
				match = false
			}
		}

		// max_length filter....
		if param := c.Query("max_length"); param != "" {
			val, err := strconv.Atoi(param)
			if err != nil || val < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid value for 'max_length', must be a non-negative integer"})
				return
			}
			filters["max_length"] = val
			if str.Properties.Length > val {
				match = false
			}
		}

		// word_count filter....
		if param := c.Query("word_count"); param != "" {
			val, err := strconv.Atoi(param)
			if err != nil || val < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid value for 'word_count', must be a non-negative integer"})
				return
			}
			filters["word_count"] = val
			if str.Properties.WordCount != val {
				match = false
			}
		}

		// contains_character filter....
		if param := c.Query("contains_character"); param != "" {
			if len([]rune(param)) != 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid value for 'contains_character', must be a single character"})
				return
			}
			charToFind := []rune(param)[0]
			filters["contains_character"] = param
			found := false
			for c := range str.Properties.CharacterFrequencyMap {
				if c == charToFind {
					found = true
					break
				}
			}
			if !found {
				match = false
			}
		}

		if match {
			filteredStrings = append(filteredStrings, str)
		}
	}

	c.JSON(http.StatusOK, FilteredStringsResponse{
		Data:           filteredStrings,
		Count:          len(filteredStrings),
		FiltersApplied: filters,
	})
}

// ParseNaturalLanguageQuery attempts to parse a natural language query into structured filters....
func ParseNaturalLanguageQuery(query string) (map[string]interface{}, error) {
	parsedFilters := make(map[string]interface{})
	lowerQuery := strings.ToLower(query)

	// Palindrome....
	if strings.Contains(lowerQuery, "palindrome") || strings.Contains(lowerQuery, "palindromic") {
		parsedFilters["is_palindrome"] = true
	}

	// Word Count....
	reWordCount := regexp.MustCompile(`(single|one|two|three|four|five|six|seven|eight|nine|ten) word`)
	if matches := reWordCount.FindStringSubmatch(lowerQuery); len(matches) > 1 {
		switch matches[1] {
		case "single", "one":
			parsedFilters["word_count"] = 1
		case "two":
			parsedFilters["word_count"] = 2
		case "three":
			parsedFilters["word_count"] = 3
		case "four":
			parsedFilters["word_count"] = 4
		case "five":
			parsedFilters["word_count"] = 5
		// Add more cases as needed....
		default:
			return nil, fmt.Errorf("unsupported word count '%s'", matches[1])
		}
	} else {
		reWordCountNum := regexp.MustCompile(`(\d+) words?`) // Corrected: single backslash for \d
		if matches := reWordCountNum.FindStringSubmatch(lowerQuery); len(matches) > 1 {
			num, err := strconv.Atoi(matches[1])
			if err == nil {
				parsedFilters["word_count"] = num
			}
		}
	}

	// Length filters....
	reLongerThan := regexp.MustCompile(`longer than (\d+)`)
	if matches := reLongerThan.FindStringSubmatch(lowerQuery); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			if existingMin, ok := parsedFilters["min_length"].(int); ok && existingMin > num+1 {
				// Conflict: already has a higher min_length....
				return nil, fmt.Errorf("conflicting length filters detected")
			}
			parsedFilters["min_length"] = num + 1 // "longer than X" means min_length = X + 1....
		}
	}

	reShorterThan := regexp.MustCompile(`shorter than (\d+)`)
	if matches := reShorterThan.FindStringSubmatch(lowerQuery); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			if existingMax, ok := parsedFilters["max_length"].(int); ok && existingMax < num-1 {
				// Conflict: already has a lower max_length....
				return nil, fmt.Errorf("conflicting length filters detected")
			}
			parsedFilters["max_length"] = num - 1 // "shorter than X" means max_length = X - 1....
		}
	}

	reExactlyLength := regexp.MustCompile(`exactly (\d+)`)
	if matches := reExactlyLength.FindStringSubmatch(lowerQuery); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			// Check for conflicts with min/max length....
			if existingMin, ok := parsedFilters["min_length"].(int); ok && existingMin > num {
				return nil, fmt.Errorf("conflicting length filters detected")
			}
			if existingMax, ok := parsedFilters["max_length"].(int); ok && existingMax < num {
				return nil, fmt.Errorf("conflicting length filters detected")
			}
			parsedFilters["min_length"] = num
			parsedFilters["max_length"] = num
		}
	}

	// Contains character....
	reContainsChar := regexp.MustCompile(`contains the letter ([a-z])`)
	if matches := reContainsChar.FindStringSubmatch(lowerQuery); len(matches) > 1 {
		parsedFilters["contains_character"] = matches[1]
	} else {
		if strings.Contains(lowerQuery, "contains the first vowel") {
			parsedFilters["contains_character"] = "a"
		}
	}

	// Check for overall min_length > max_length conflict....
	if minLen, okMin := parsedFilters["min_length"].(int); okMin {
		if maxLen, okMax := parsedFilters["max_length"].(int); okMax {
			if minLen > maxLen {
				return nil, fmt.Errorf("query resulted in conflicting length filters (min_length > max_length)")
			}
		}
	}

	if len(parsedFilters) == 0 {
		return nil, fmt.Errorf("unable to parse natural language query into filters")
	}

	return parsedFilters, nil
}

// NaturalLanguageFilterHandler handles GET /strings/filter-by-natural-language....
func NaturalLanguageFilterHandler(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'query' parameter"})
		return
	}

	parsedFilters, err := ParseNaturalLanguageQuery(query)
	if err != nil {
		if strings.Contains(err.Error(), "conflicting") {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Conflicting filters detected in query"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	filteredStrings := make([]AnalyzedString, 0) // Initialize as empty slice, not nil
	mu.RLock()
	defer mu.RUnlock()

	for _, str := range stringStore {
		match := true

		// Apply parsed filters....
		for filterKey, filterValue := range parsedFilters {
			switch filterKey {
			case "is_palindrome":
				if val, ok := filterValue.(bool); ok && str.Properties.IsPalindrome != val {
					match = false
				}
			case "min_length":
				if val, ok := filterValue.(int); ok && str.Properties.Length < val {
					match = false
				}
			case "max_length":
				if val, ok := filterValue.(int); ok && str.Properties.Length > val {
					match = false
				}
			case "word_count":
				if val, ok := filterValue.(int); ok && str.Properties.WordCount != val {
					match = false
				}
			case "contains_character":
				if val, ok := filterValue.(string); ok {
					charToFind := []rune(val)[0]
					found := false
					for c := range str.Properties.CharacterFrequencyMap {
						if c == charToFind {
							found = true
							break
						}
					}
					if !found {
						match = false
					}
				}
			}
			if !match {
				break
			}
		}

		if match {
			filteredStrings = append(filteredStrings, str)
		}
	}

	c.JSON(http.StatusOK, NaturalLanguageFilterResponse{
		Data:  filteredStrings,
		Count: len(filteredStrings),
		InterpretedQuery: NaturalLanguageQuery{
			Original:      query,
			ParsedFilters: parsedFilters,
		},
	})
}

// DeleteStringHandler handles DELETE /strings/{string_value}....
func DeleteStringHandler(c *gin.Context) {
	stringValue := c.Param("string_value")

	mu.Lock()
	defer mu.Unlock()

	deleted := false
	// Try to delete by direct hash....
	if _, exists := stringStore[stringValue]; exists {
		delete(stringStore, stringValue)
		deleted = true
	} else {
		// If not found by direct hash, try to hash the param and delete....
		hashedValue := generateSHA256Hash(stringValue)
		if _, exists := stringStore[hashedValue]; exists {
			delete(stringStore, hashedValue)
			deleted = true
		}
	}

	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "String not found in the system"})
		return
	}

	c.Status(http.StatusNoContent)
}

// HealthCheckHandler handles GET /health....
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "String Analyzer API",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// RootHandler handles GET /....
func RootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":     "Welcome to String Analyzer API",
		"description": "A REST API for analyzing string properties including palindrome detection, character frequency, and more",
		"version":     "1.0.0",
		"endpoints": gin.H{
			"POST /strings":                           "Create and analyze a new string",
			"GET /strings":                            "Get all strings with optional filtering",
			"GET /strings/:string_value":              "Get a specific string by value or hash",
			"GET /strings/filter-by-natural-language": "Filter strings using natural language queries",
			"DELETE /strings/:string_value":           "Delete a string by value or hash",
			"GET /health":                             "Health check endpoint",
		},
		"documentation": "See README.md for detailed API documentation",
	})
}

func main() {
	router := gin.Default()

	// Use logging middleware....
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next() // Process the request....
		duration := time.Since(start)
		log.Printf("Request - Method: %s, Path: %s, Status: %d, Duration: %s",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
	})

	// Register API endpoints....
	router.GET("/", RootHandler)
	router.GET("/health", HealthCheckHandler)
	router.POST("/strings", CreateStringHandler)
	router.GET("/strings", GetFilteredStringsHandler)
	router.GET("/strings/:string_value", GetSpecificStringHandler)
	router.GET("/strings/filter-by-natural-language", NaturalLanguageFilterHandler)
	router.DELETE("/strings/:string_value", DeleteStringHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified....
	}

	log.Printf("Server starting on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

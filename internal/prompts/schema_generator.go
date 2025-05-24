package prompts

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// JSONExampleGenerator generates JSON examples from Go structs using reflection
type JSONExampleGenerator struct{}

// GenerateJSONExample creates a JSON example string from a struct type
// with placeholder values that demonstrate the expected structure
func (g *JSONExampleGenerator) GenerateJSONExample(structType interface{}) (string, error) {
	example := g.createExampleStruct(structType)
	jsonBytes, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal example struct: %w", err)
	}
	return string(jsonBytes), nil
}

// GenerateJSONExampleCompact creates a compact JSON example (single line)
func (g *JSONExampleGenerator) GenerateJSONExampleCompact(structType interface{}) (string, error) {
	example := g.createExampleStruct(structType)
	jsonBytes, err := json.Marshal(example)
	if err != nil {
		return "", fmt.Errorf("failed to marshal example struct: %w", err)
	}
	return string(jsonBytes), nil
}

// createExampleStruct uses reflection to create an example struct with placeholder values
func (g *JSONExampleGenerator) createExampleStruct(structType interface{}) interface{} {
	t := reflect.TypeOf(structType)
	v := reflect.ValueOf(structType)

	// If it's a pointer, get the element
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Create a new instance of the struct
	newStruct := reflect.New(t).Elem()

	// Fill in the fields with example values
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := newStruct.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue // Skip fields marked with json:"-"
		}

		// Parse JSON tag to get the field name
		tagParts := strings.Split(jsonTag, ",")
		fieldName := tagParts[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		// Set example values based on field type and name
		g.setExampleValue(fieldValue, field, fieldName)
	}

	return newStruct.Interface()
}

// Add an allowlist to filter fields for JSON example generation
func (g *JSONExampleGenerator) createExampleStructWithAllowlist(structType interface{}, allowlist map[string]bool) interface{} {
	t := reflect.TypeOf(structType)

	// If it's a pointer, get the element
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Create a new instance of the struct
	newStruct := reflect.New(t).Elem()

	// Fill in the fields with example values
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := newStruct.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get the JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue // Skip fields marked with json:"-"
		}

		// Parse JSON tag to get the field name
		tagParts := strings.Split(jsonTag, ",")
		fieldName := tagParts[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		// Check if the field is in the allowlist
		if !allowlist[fieldName] {
			continue
		}

		// Set example values based on field type and name
		g.setExampleValue(fieldValue, field, fieldName)
	}

	return newStruct.Interface()
}

// setExampleValue sets appropriate example values based on field type and name
func (g *JSONExampleGenerator) setExampleValue(fieldValue reflect.Value, field reflect.StructField, jsonFieldName string) {
	switch fieldValue.Kind() {
	case reflect.String:
		example := g.getStringExample(jsonFieldName, field.Name)
		fieldValue.SetString(example)
	case reflect.Bool:
		example := g.getBoolExample(jsonFieldName, field.Name)
		fieldValue.SetBool(example)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.SetInt(g.getIntExample(jsonFieldName, field.Name))
	case reflect.Float32, reflect.Float64:
		fieldValue.SetFloat(g.getFloatExample(jsonFieldName, field.Name))
	case reflect.Slice:
		g.setSliceExample(fieldValue, field, jsonFieldName)
	case reflect.Struct:
		// For nested structs, recursively create examples
		if fieldValue.CanSet() {
			nestedExample := g.createExampleStruct(fieldValue.Interface())
			fieldValue.Set(reflect.ValueOf(nestedExample))
		}
	}
}

// getStringExample returns appropriate string examples based on field names
func (g *JSONExampleGenerator) getStringExample(jsonName, fieldName string) string {
	switch strings.ToLower(jsonName) {
	case "id", "itemid":
		return "t3_1keo3te"
	case "title":
		return "Example Article Title"
	case "summary":
		return "Brief summary of the content..."
	case "commentsummary":
		return "Community discussion highlights..."
	case "imagedescription", "imagesummary":
		return "Description of any images in the post..."
	case "webcontentsummary":
		return "Summary of linked web content..."
	case "link":
		return "https://example.com/article"
	case "thumbnailurl":
		return "https://example.com/thumbnail.jpg"
	case "text":
		return "Key development description..."
	default:
		return ""
	}
}

// getBoolExample returns appropriate boolean examples based on field names
func (g *JSONExampleGenerator) getBoolExample(jsonName, fieldName string) bool {
	switch strings.ToLower(jsonName) {
	case "isrelevant":
		return true
	default:
		return false
	}
}

// getIntExample returns appropriate integer examples
func (g *JSONExampleGenerator) getIntExample(jsonName, fieldName string) int64 {
	return 0
}

// getFloatExample returns appropriate float examples
func (g *JSONExampleGenerator) getFloatExample(jsonName, fieldName string) float64 {
	return 0.0
}

// setSliceExample sets example values for slice fields
func (g *JSONExampleGenerator) setSliceExample(fieldValue reflect.Value, field reflect.StructField, jsonFieldName string) {
	elementType := fieldValue.Type().Elem()

	// Create a slice with one example element
	slice := reflect.MakeSlice(fieldValue.Type(), 1, 1)
	element := slice.Index(0)

	if elementType.Kind() == reflect.Struct {
		// For struct slices, create an example struct
		exampleStruct := g.createExampleStruct(reflect.New(elementType).Interface())
		element.Set(reflect.ValueOf(exampleStruct))
	} else if elementType.Kind() == reflect.String {
		// For string slices, add an example string
		element.SetString("example item")
	}

	fieldValue.Set(slice)
}

// Update GetItemJSONExample to use the allowlist
func GetItemJSONExample() (string, error) {
	generator := &JSONExampleGenerator{}
	allowlist := map[string]bool{
		"id":             true,
		"title":          true,
		"summary":        true,
		"commentSummary": true,
		"isRelevant":     true,
	}
	return generator.GenerateJSONExampleCompactWithAllowlist(createItemExample(), allowlist)
}

// GenerateJSONExampleCompactWithAllowlist creates a compact JSON example with an allowlist
func (g *JSONExampleGenerator) GenerateJSONExampleCompactWithAllowlist(structType interface{}, allowlist map[string]bool) (string, error) {
	example := g.createExampleStructWithAllowlist(structType, allowlist)
	jsonBytes, err := json.Marshal(example)
	if err != nil {
		return "", fmt.Errorf("failed to marshal example struct: %w", err)
	}
	return string(jsonBytes), nil
}

// GetSummaryResponseJSONExample returns a formatted JSON example for the SummaryResponse struct
func GetSummaryResponseJSONExample() (string, error) {
	generator := &JSONExampleGenerator{}
	return generator.GenerateJSONExampleCompact(createSummaryResponseExample())
}

// Temporary example structs - these would be replaced with actual imports
// when integrating with the real models package
type itemExample struct {
	Title             string `json:"title"`
	ID                string `json:"id"`
	Summary           string `json:"summary"`
	CommentSummary    string `json:"commentSummary,omitempty"`
	ImageSummary      string `json:"imageDescription,omitempty"`
	WebContentSummary string `json:"webContentSummary,omitempty"`
	Link              string `json:"link,omitempty"`
	IsRelevant        bool   `json:"isRelevant"`
	ThumbnailURL      string `json:"thumbnailUrl,omitempty"`
}

type keyDevelopmentExample struct {
	Text   string `json:"text"`
	ItemID string `json:"itemID"`
}

type summaryResponseExample struct {
	KeyDevelopments []keyDevelopmentExample `json:"keyDevelopments"`
}

func createItemExample() itemExample {
	return itemExample{}
}

func createSummaryResponseExample() summaryResponseExample {
	return summaryResponseExample{}
}

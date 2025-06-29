package providers

import (
	"strings"
)

// this should just be a lower case string representing the persona name
func processPersonaName(personaName string) string {
	personaName = strings.ToLower(personaName)
	personaName = strings.ReplaceAll(personaName, "/", "")
	return personaName
}

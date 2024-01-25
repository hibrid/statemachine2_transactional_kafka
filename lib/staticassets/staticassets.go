package staticassets

import "embed"

// Get retuns bytes for a requested resource
func Get(resource string, dataBox embed.FS) ([]byte, error) {
	return dataBox.ReadFile(resource)
}

package env

import (
	"os"
	"strconv"
	"sync"

	"strings"
)

// IsDevLike holds whether the system thinks its in a dev like env or not
var IsDevLike bool

// IsProdLike holds whether the system thinks its in a prod like env or not
var IsProdLike bool
var oneTime sync.Once

// Has returns a bool if the environment has the given env var
func Has(envVar string) bool {
	return os.Getenv(envVar) != ""
}

// GetIfSet returns the value of an env var and the bool if it was found or not
func GetIfSet(envVar string) (string, bool) {
	return os.LookupEnv(envVar)
}

// GetLower returns the env var value in lowercase
func GetLower(envVar string) string {
	return strings.ToLower(Get(envVar))
}

// Get returns or panics if an env var is not found
func Get(envVar string) string {
	if v, ok := os.LookupEnv(envVar); !ok {
		panic("Environment variable " + envVar + " not found")
	} else {
		return v
	}
}

// GetOrDefault returns the env var or the default value if the env var is not set
func GetOrDefault(envVar string, defaultValue string) string {
	if v, ok := os.LookupEnv(envVar); !ok {
		return defaultValue
	} else {
		return v
	}
}

// GetBool converts an env var value to a bool or panics if not set
func GetBool(envVar string) bool {
	if raw, ok := os.LookupEnv(envVar); !ok {
		panic("Environment variable " + envVar + " not found")
	} else if parsed, err := strconv.ParseBool(raw); err != nil {
		panic(err)
	} else {
		return parsed
	}
}

// GetInt converts an env var value to an int64 or panics if not set
func GetInt(envVar string) int64 {
	if raw, ok := os.LookupEnv(envVar); !ok {
		panic("Environment variable " + envVar + " not found")
	} else if parsed, err := strconv.ParseInt(raw, 10, 64); err != nil {
		panic(err)
	} else {
		return parsed
	}
}

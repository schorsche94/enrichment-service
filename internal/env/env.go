package env

import (
	"fmt"
	"os"
	"strconv"
)

func GetString(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return val
}

func GetInt(key string, fallback int) (int, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	valAsInt, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("parse %s as int: %w", key, err)
	}

	return valAsInt, nil
}

func GetBool(key string, fallback bool) (bool, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("parse %s as bool: %w", key, err)
	}

	return boolVal, nil
}

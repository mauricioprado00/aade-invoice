package mydata

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Credentials for one myDATA environment.
type Credentials struct {
	UserID          string
	SubscriptionKey string
	VatNumber       string
	BaseURL         string
	Name            string
}

// LoadDotEnv reads KEY=VALUE lines into the process environment, without
// overwriting variables that are already set.
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, value)
		}
	}
	return s.Err()
}

// LoadCredentials picks the sandbox or production block out of the environment.
// Production has an extra /myDATA/ path segment that the dev host does not.
func LoadCredentials(production bool) (Credentials, error) {
	c := Credentials{Name: "sandbox", BaseURL: "https://mydataapidev.aade.gr"}
	prefix := "AADE_SANDBOX_"
	if production {
		c = Credentials{Name: "production", BaseURL: "https://mydatapi.aade.gr/myDATA"}
		prefix = "AADE_PROD_"
	}

	c.UserID = os.Getenv(prefix + "USER_ID")
	c.SubscriptionKey = os.Getenv(prefix + "SUBSCRIPTION_KEY")
	c.VatNumber = os.Getenv(prefix + "VAT_NUMBER")

	for name, value := range map[string]string{
		prefix + "USER_ID":          c.UserID,
		prefix + "SUBSCRIPTION_KEY": c.SubscriptionKey,
		prefix + "VAT_NUMBER":       c.VatNumber,
	} {
		if value == "" {
			return c, fmt.Errorf("%s is not set (see .env.example)", name)
		}
	}
	return c, nil
}

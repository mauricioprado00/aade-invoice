// Package cli holds the pieces every aade-* command shares: the same
// environment selection, the same credential loading, the same failure
// handling.
package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

// Common is the flag set every command carries. Sandbox is the default
// everywhere; --prod is always a deliberate act.
type Common struct {
	Production *bool
	EnvPath    *string
}

func Register(fs *flag.FlagSet) *Common {
	return &Common{
		Production: fs.Bool("prod", false, "talk to the production myDATA API instead of the sandbox"),
		EnvPath:    fs.String("env", ".env", "file holding the API credentials"),
	}
}

// Credentials loads the .env file and returns the credentials for the selected
// environment.
func (c *Common) Credentials() (mydata.Credentials, error) {
	if err := mydata.LoadDotEnv(*c.EnvPath); err != nil {
		return mydata.Credentials{}, err
	}
	return mydata.LoadCredentials(*c.Production)
}

// Client is Credentials plus a ready client.
func (c *Common) Client() (*mydata.Client, mydata.Credentials, error) {
	creds, err := c.Credentials()
	if err != nil {
		return nil, creds, err
	}
	return mydata.NewClient(creds), creds, nil
}

// Main runs f and turns an error into a message on stderr and exit status 1.
func Main(f func() error) {
	if err := f(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

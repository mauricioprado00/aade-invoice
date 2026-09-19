// Command aade-expenses shows the expense side of the myDATA books.
package main

import (
	"os"

	"github.com/mauricioprado00/aade-invoice/internal/cli"
	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func main() {
	books := cli.Books{
		Name:        "aade-expenses",
		Path:        mydata.PathMyExpenses,
		What:        "expenses",
		Counterpart: "supplier",
	}
	cli.Main(func() error { return books.Run(os.Args[1:]) })
}

// Command aade-income shows the income side of the myDATA books.
package main

import (
	"os"

	"github.com/mauricioprado00/aade-invoice/internal/cli"
	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func main() {
	books := cli.Books{
		Name:        "aade-income",
		Path:        mydata.PathMyIncome,
		What:        "income",
		Counterpart: "customer",
	}
	cli.Main(func() error { return books.Run(os.Args[1:]) })
}

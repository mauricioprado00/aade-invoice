// Command aade-vat shows the VAT inputs and outputs myDATA holds for a period.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/mauricioprado00/aade-invoice/internal/cli"
	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func main() { cli.Main(run) }

func run() error {
	fs := flag.NewFlagSet("aade-vat", flag.ExitOnError)
	common := cli.Register(fs)
	var (
		from     = fs.String("from", "", "start of the period, YYYY-MM-DD (required)")
		to       = fs.String("to", "", "end of the period, YYYY-MM-DD (required)")
		entity   = fs.String("entity", "", "VAT number to report on (default: our own)")
		perDay   = fs.Bool("per-day", false, "group the records per day instead of per invoice")
		totals   = fs.Bool("totals", false, "print only the totals per VAT box")
		maxPages = fs.Int("max-pages", 20, "safety limit on paged fetches")
		asJSON   = fs.Bool("json", false, "print the records as JSON instead of a table")
		dryRun   = fs.Bool("dry-run", false, "print the request URL that would be called and stop")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: aade-vat --from YYYY-MM-DD --to YYYY-MM-DD [flags]\n\nShows the VAT boxes (Vat301, Vat361, …) myDATA holds per invoice, or per day\nwith --per-day. Only the boxes a record actually carries are printed.\n\nflags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	query, err := cli.InfoQuery(*from, *to, *entity, *perDay)
	if err != nil {
		return err
	}

	client, _, err := common.Client()
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Println(client.URL(mydata.PathVatInfo, query.Values()))
		return nil
	}

	// Paging only applies when the results are not grouped per day; asking for
	// a second page in that mode returns the same one, so stop after the first.
	pages := *maxPages
	if *perDay {
		pages = 1
	}
	records, err := mydata.FetchAllPages(func(token *mydata.ContinuationToken) ([]mydata.VatInfo, *mydata.ContinuationToken, error) {
		q := query
		q.Continuation = token
		doc, err := client.RequestVatInfo(q)
		if err != nil {
			return nil, nil, err
		}
		return doc.Records, doc.Continuation, nil
	}, pages)
	if err != nil {
		return err
	}

	switch {
	case *asJSON:
		return cli.PrintJSON(records)
	case *totals:
		printTotals(records)
	default:
		printTable(records)
	}
	return nil
}

func printTable(records []mydata.VatInfo) {
	if len(records) == 0 {
		fmt.Println("no VAT records in that period")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MARK\tDATE\tCANCELLED\tBOXES")
	for _, r := range records {
		cancelled := ""
		if r.IsCancelled {
			cancelled = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", mark(r), date(r.IssueDate), cancelled, boxes(r.Amounts))
	}
	w.Flush()
	fmt.Printf("\n%d record(s)\n\n", len(records))
	printTotals(records)
}

// printTotals adds every box across the records, which is the number that goes
// on the VAT return.
func printTotals(records []mydata.VatInfo) {
	sums := map[string][]string{}
	for _, r := range records {
		if r.IsCancelled {
			continue
		}
		for _, a := range r.Amounts {
			sums[a.Name] = append(sums[a.Name], a.Value)
		}
	}
	if len(sums) == 0 {
		fmt.Println("no totals: every record in that period is cancelled or empty")
		return
	}

	names := make([]string, 0, len(sums))
	for name := range sums {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println("totals (cancelled records excluded)")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, name := range names {
		fmt.Fprintf(w, "  %s\t%s\n", name, mydata.Sum(sums[name]...))
	}
	w.Flush()
}

func boxes(amounts []mydata.VatAmount) string {
	out := ""
	for _, a := range amounts {
		if a.Value == "" || mydata.Sum(a.Value) == "0.00" {
			continue
		}
		if out != "" {
			out += " "
		}
		out += a.Name + "=" + a.Value
	}
	if out == "" {
		return "-"
	}
	return out
}

func mark(r mydata.VatInfo) string {
	if r.Mark == "" {
		return "-"
	}
	return r.Mark
}

// date trims the time off the dateTime myDATA returns here.
func date(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

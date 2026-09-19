// Command aade-e3 shows the E3 classification lines myDATA holds for a period.
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
	fs := flag.NewFlagSet("aade-e3", flag.ExitOnError)
	common := cli.Register(fs)
	var (
		from     = fs.String("from", "", "start of the period, YYYY-MM-DD (required)")
		to       = fs.String("to", "", "end of the period, YYYY-MM-DD (required)")
		entity   = fs.String("entity", "", "VAT number to report on (default: our own)")
		perDay   = fs.Bool("per-day", false, "group the records per day instead of per invoice")
		totals   = fs.Bool("totals", false, "print only the totals per E3 code")
		maxPages = fs.Int("max-pages", 20, "safety limit on paged fetches")
		asJSON   = fs.Bool("json", false, "print the records as JSON instead of a table")
		dryRun   = fs.Bool("dry-run", false, "print the request URL that would be called and stop")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: aade-e3 --from YYYY-MM-DD --to YYYY-MM-DD [flags]\n\nShows the E3 lines (code, category and amount) behind the income tax return.\n\nflags:\n")
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
		fmt.Println(client.URL(mydata.PathE3Info, query.Values()))
		return nil
	}

	// Paging only applies when the results are not grouped per day.
	pages := *maxPages
	if *perDay {
		pages = 1
	}
	records, err := mydata.FetchAllPages(func(token *mydata.ContinuationToken) ([]mydata.E3Info, *mydata.ContinuationToken, error) {
		q := query
		q.Continuation = token
		doc, err := client.RequestE3Info(q)
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

func printTable(records []mydata.E3Info) {
	if len(records) == 0 {
		fmt.Println("no E3 lines in that period")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MARK\tDATE\tBOOK\tCANCELLED\tCODE\tCATEGORY\tVALUE")
	for _, r := range records {
		cancelled := ""
		if r.IsCancelled {
			cancelled = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			or(r.Mark), date(r.IssueDate), or(r.Book), cancelled,
			or(r.ClassType), or(r.ClassCategory), r.ClassValue)
	}
	w.Flush()
	fmt.Printf("\n%d line(s)\n\n", len(records))
	printTotals(records)
}

func printTotals(records []mydata.E3Info) {
	sums := map[string][]string{}
	for _, r := range records {
		if r.IsCancelled {
			continue
		}
		code := r.ClassType
		if code == "" {
			code = or(r.ClassCategory)
		}
		sums[code] = append(sums[code], r.ClassValue)
	}
	if len(sums) == 0 {
		fmt.Println("no totals: every line in that period is cancelled")
		return
	}

	codes := make([]string, 0, len(sums))
	for code := range sums {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	fmt.Println("totals (cancelled lines excluded)")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, code := range codes {
		fmt.Fprintf(w, "  %s\t%s\t(%d line(s))\n", code, mydata.Sum(sums[code]...), len(sums[code]))
	}
	w.Flush()
}

func or(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// date trims the time off the dateTime myDATA returns here.
func date(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

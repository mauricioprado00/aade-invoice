// Command aade-list lists invoices held in myDATA.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mauricioprado00/aade-invoice/internal/cli"
	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func main() { cli.Main(run) }

func run() error {
	fs := flag.NewFlagSet("aade-list", flag.ExitOnError)
	common := cli.Register(fs)
	var (
		source    = fs.String("source", "transmitted", "which side to read: transmitted (what we submitted), received (what others sent us), or all")
		from      = fs.String("from", "", "earliest issue date, YYYY-MM-DD")
		to        = fs.String("to", "", "latest issue date, YYYY-MM-DD")
		invType   = fs.String("type", "", "invoice type, e.g. 2.3 (see appendix 8.1 of the spec)")
		counter   = fs.String("counterpart", "", "VAT number of the other party")
		afterMark = fs.Int64("after-mark", 0, "only documents with a MARK above this one")
		maxMark   = fs.Int64("max-mark", 0, "only documents with a MARK up to this one")
		series    = fs.String("series", "", "only this invoice series (filtered locally)")
		maxPages  = fs.Int("max-pages", 20, "safety limit on paged fetches")
		asXML     = fs.Bool("xml", false, "print the raw matching documents as XML instead of a table")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: aade-list [flags]\n\nLists invoices in myDATA. Dates filter on the issue date.\nThe DIR column shows out for documents we issued and in for the rest.\n\nflags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	dateFrom, err := mydata.NormalizeDate(*from)
	if err != nil {
		return err
	}
	dateTo, err := mydata.NormalizeDate(*to)
	if err != nil {
		return err
	}

	client, creds, err := common.Client()
	if err != nil {
		return err
	}

	query := mydata.Query{
		AfterMark:        *afterMark,
		MaxMark:          *maxMark,
		DateFrom:         dateFrom,
		DateTo:           dateTo,
		InvType:          *invType,
		CounterVatNumber: *counter,
	}

	// "transmitted" is not the same as "issued": expense documents we register
	// ourselves are transmitted by us but issued by someone else. The DIR
	// column below tells the two apart by comparing the issuer's VAT number.
	var docs []mydata.Doc
	switch *source {
	case "transmitted", "issued":
		docs, err = mydata.FetchAll(client.RequestTransmittedDocs, query, *maxPages)
	case "received":
		docs, err = mydata.FetchAll(client.RequestDocs, query, *maxPages)
	case "all":
		docs, err = mydata.FetchAll(client.RequestTransmittedDocs, query, *maxPages)
		if err == nil {
			var received []mydata.Doc
			received, err = mydata.FetchAll(client.RequestDocs, query, *maxPages)
			docs = append(docs, received...)
		}
	default:
		return fmt.Errorf("--source must be transmitted, received or all")
	}
	if err != nil {
		return err
	}

	if *series != "" {
		docs = filter(docs, func(d mydata.Doc) bool { return d.Header.Series == *series })
	}

	if *asXML {
		return printXML(docs)
	}
	printTable(docs, creds.VatNumber)
	return nil
}

func filter(docs []mydata.Doc, keep func(mydata.Doc) bool) []mydata.Doc {
	var out []mydata.Doc
	for _, d := range docs {
		if keep(d) {
			out = append(out, d)
		}
	}
	return out
}

func printTable(docs []mydata.Doc, ownVat string) {
	if len(docs) == 0 {
		fmt.Println("no matching documents")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MARK\tDATE\tTYPE\tSERIES/AA\tDIR\tCOUNTERPART\tNET\tVAT\tGROSS")
	for _, d := range docs {
		direction, counterpart := "in", describe(d.Issuer)
		if d.IssuedBy(ownVat) {
			direction, counterpart = "out", describe(d.Counterpart)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			d.Mark, d.Header.IssueDate, d.Header.InvoiceType,
			d.Header.Series+"/"+d.Header.Aa, direction, counterpart,
			d.Summary.TotalNetValue, d.Summary.TotalVatAmount, d.Summary.TotalGrossValue)
	}
	w.Flush()
	fmt.Printf("\n%d document(s)\n", len(docs))
}

func describe(p mydata.DocParty) string {
	if p.Name != "" {
		return truncate(p.Name, 24)
	}
	if p.VatNumber == "" {
		return "-"
	}
	return p.VatNumber
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func printXML(docs []mydata.Doc) error {
	out, err := mydata.MarshalDocs(docs)
	if err != nil {
		return err
	}
	fmt.Println(strings.TrimRight(string(out), "\n"))
	return nil
}

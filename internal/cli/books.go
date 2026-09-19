package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

// Books runs aade-income and aade-expenses, which differ only in the method
// they call and in the word they use for the other party.
type Books struct {
	Name        string // command name, e.g. aade-income
	Path        string // mydata.PathMyIncome or mydata.PathMyExpenses
	What        string // "income" or "expenses", for the usage text
	Counterpart string // what the other party is called on this side
}

func (b Books) Run(args []string) error {
	fs := flag.NewFlagSet(b.Name, flag.ExitOnError)
	common := Register(fs)
	var (
		from     = fs.String("from", "", "start of the period, YYYY-MM-DD (required)")
		to       = fs.String("to", "", "end of the period, YYYY-MM-DD (required)")
		counter  = fs.String("counterpart", "", "VAT number of the other party")
		entity   = fs.String("entity", "", "VAT number to report on (default: our own)")
		invType  = fs.String("type", "", "invoice type, e.g. 2.3 (see appendix 8.1 of the spec)")
		maxPages = fs.Int("max-pages", 20, "safety limit on paged fetches")
		asJSON   = fs.Bool("json", false, "print the records as JSON instead of a table")
		dryRun   = fs.Bool("dry-run", false, "print the request URL that would be called and stop")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s --from YYYY-MM-DD --to YYYY-MM-DD [flags]\n\nShows the %s myDATA holds for our VAT number, aggregated per\n%s, issue date and invoice type.\n\nflags:\n", b.Name, b.What, b.Counterpart)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	query, err := bookQuery(*from, *to)
	if err != nil {
		return err
	}
	query.CounterVatNumber = *counter
	query.EntityVatNumber = *entity
	query.InvType = *invType

	client, _, err := common.Client()
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Println(client.URL(b.Path, query.Values()))
		return nil
	}

	fetch := client.RequestMyIncome
	if b.Path == mydata.PathMyExpenses {
		fetch = client.RequestMyExpenses
	}
	entries, err := mydata.FetchAllPages(func(token *mydata.ContinuationToken) ([]mydata.BookEntry, *mydata.ContinuationToken, error) {
		q := query
		q.Continuation = token
		doc, err := fetch(q)
		if err != nil {
			return nil, nil, err
		}
		return doc.Entries, doc.Continuation, nil
	}, *maxPages)
	if err != nil {
		return err
	}

	if *asJSON {
		return PrintJSON(entries)
	}
	b.printTable(entries)
	return nil
}

func (b Books) printTable(entries []mydata.BookEntry) {
	if len(entries) == 0 {
		fmt.Printf("no %s in that period\n", b.What)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "DATE\tTYPE\t%s\tDOCS\tNET\tVAT\tWITHHELD\tGROSS\tMARKS\n", upper(b.Counterpart))
	var net, vat, withheld, gross []string
	docs := 0
	for _, e := range entries {
		marks := e.MinMark
		if e.MaxMark != "" && e.MaxMark != e.MinMark {
			marks += "-" + e.MaxMark
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
			e.IssueDate, e.InvType, dash(e.CounterVatNumber), e.Count,
			e.NetValue, e.VatAmount, e.WithheldAmount, e.GrossValue, marks)
		net = append(net, e.NetValue)
		vat = append(vat, e.VatAmount)
		withheld = append(withheld, e.WithheldAmount)
		gross = append(gross, e.GrossValue)
		docs += e.Count
	}
	fmt.Fprintf(w, "\t\tTOTAL\t%d\t%s\t%s\t%s\t%s\t\n", docs,
		mydata.Sum(net...), mydata.Sum(vat...), mydata.Sum(withheld...), mydata.Sum(gross...))
	w.Flush()
	fmt.Printf("\n%d line(s), %d document(s)\n", len(entries), docs)
}

// bookQuery turns the two date flags into the dd/MM/yyyy pair myDATA requires
// here; unlike RequestDocs, neither is optional.
func bookQuery(from, to string) (mydata.BookQuery, error) {
	var q mydata.BookQuery
	if from == "" || to == "" {
		return q, fmt.Errorf("--from and --to are both required")
	}
	dateFrom, err := mydata.NormalizeDate(from)
	if err != nil {
		return q, err
	}
	dateTo, err := mydata.NormalizeDate(to)
	if err != nil {
		return q, err
	}
	q.DateFrom, q.DateTo = dateFrom, dateTo
	return q, nil
}

// InfoQuery is bookQuery for the two methods that take a period and nothing
// about the counterpart.
func InfoQuery(from, to, entity string, groupedPerDay bool) (mydata.InfoQuery, error) {
	b, err := bookQuery(from, to)
	if err != nil {
		return mydata.InfoQuery{}, err
	}
	return mydata.InfoQuery{
		DateFrom:        b.DateFrom,
		DateTo:          b.DateTo,
		EntityVatNumber: entity,
		GroupedPerDay:   groupedPerDay,
	}, nil
}

func PrintJSON(v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func upper(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'a' && r <= 'z' {
			out[i] = r - 32
		}
	}
	return string(out)
}

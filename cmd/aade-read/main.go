// Command aade-read shows one invoice, found by its MARK.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mauricioprado00/aade-invoice/internal/cli"
	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func main() { cli.Main(run) }

func run() error {
	fs := flag.NewFlagSet("aade-read", flag.ExitOnError)
	common := cli.Register(fs)
	var (
		asJSON = fs.Bool("json", false, "print the document as JSON")
		asXML  = fs.Bool("xml", false, "print the document as XML")
		pdfDir = fs.String("pdf", "", "also download AADE's PDF into this directory")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: aade-read [flags] <mark>\n\nShows the invoice registered under a MARK, whether we issued it or received it.\n\nflags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one MARK")
	}

	var mark int64
	if _, err := fmt.Sscan(fs.Arg(0), &mark); err != nil || mark <= 0 {
		return fmt.Errorf("%q is not a MARK", fs.Arg(0))
	}

	client, creds, err := common.Client()
	if err != nil {
		return err
	}
	doc, err := client.FindMark(mark)
	if err != nil {
		return err
	}

	switch {
	case *asJSON:
		out, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	case *asXML:
		out, err := mydata.MarshalDocs([]mydata.Doc{*doc})
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	default:
		print(doc, creds.VatNumber)
	}

	if *pdfDir != "" {
		path, err := client.DownloadInvoicePDF(mark, *pdfDir)
		if err != nil {
			return err
		}
		fmt.Printf("\nPDF: %s\n", path)
	}
	return nil
}

func print(d *mydata.Doc, ownVat string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	row := func(label, value string) {
		if value != "" {
			fmt.Fprintf(w, "%s\t%s\n", label, value)
		}
	}

	direction := "received"
	if d.IssuedBy(ownVat) {
		direction = "issued"
	}

	row("MARK", fmt.Sprint(d.Mark))
	row("uid", d.Uid)
	row("direction", direction)
	row("date", d.Header.IssueDate)
	row("number", d.Header.Series+"/"+d.Header.Aa)
	row("type", d.Header.InvoiceType)
	row("currency", d.Header.Currency)
	row("issuer", party(d.Issuer))
	row("counterpart", party(d.Counterpart))
	w.Flush()

	fmt.Println("\nlines")
	lines := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(lines, "  #\tNET\tVAT CAT\tVAT\tCLASSIFICATION")
	for _, l := range d.Details {
		vatCategory := fmt.Sprint(l.VatCategory)
		if l.VatExemptionCategory != 0 {
			vatCategory += fmt.Sprintf(" (exempt %d)", l.VatExemptionCategory)
		}
		fmt.Fprintf(lines, "  %d\t%s\t%s\t%s\t%s\n",
			l.LineNumber, l.NetValue, vatCategory, l.VatAmount,
			classifications(append(append([]mydata.Classification{}, l.Income...), l.Expenses...)))
	}
	lines.Flush()

	fmt.Println("\ntotals")
	totals := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(totals, "  net\t%s\n  vat\t%s\n  withheld\t%s\n  gross\t%s\n",
		d.Summary.TotalNetValue, d.Summary.TotalVatAmount,
		d.Summary.TotalWithheld, d.Summary.TotalGrossValue)
	totals.Flush()

	if d.QrCodeURL != "" || d.DownloadingInvoiceURL != "" {
		fmt.Println()
		links := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if d.QrCodeURL != "" {
			fmt.Fprintf(links, "QR\t%s\n", d.QrCodeURL)
		}
		if d.DownloadingInvoiceURL != "" {
			fmt.Fprintf(links, "PDF\t%s\n", d.DownloadingInvoiceURL)
		}
		links.Flush()
	}
}

func party(p mydata.DocParty) string {
	s := p.VatNumber
	if s == "" {
		s = "(not stated)"
	}
	if p.Name != "" {
		s += " " + p.Name
	}
	if p.Country != "" {
		s += " [" + p.Country + "]"
	}
	if p.City != "" {
		s += " " + p.City
	}
	return s
}

func classifications(cs []mydata.Classification) string {
	out := ""
	for i, c := range cs {
		if i > 0 {
			out += ", "
		}
		out += c.Type
		if c.Category != "" {
			out += "/" + c.Category
		}
	}
	return out
}

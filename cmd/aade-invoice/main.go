// Command aade-invoice registers an invoice in myDATA.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mauricioprado00/aade-invoice/internal/cli"
	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func main() { cli.Main(run) }

func run() error {
	fs := flag.NewFlagSet("aade-invoice", flag.ExitOnError)
	common := cli.Register(fs)
	var (
		dryRun       = fs.Bool("dry-run", false, "print the XML that would be submitted and stop")
		templatePath = fs.String("template", "", "invoice template to fill in")
		date         = fs.String("date", "", "issue date as YYYY-MM-DD (default: today)")
		aa           = fs.Int("aa", 0, "invoice number within the series (default: nextAa from the template)")
		pdf          = fs.Bool("pdf", false, "after registering, download AADE's PDF of the invoice")
		pdfMark      = fs.Int64("pdf-mark", 0, "download the PDF for an already registered MARK and exit")
		pdfDir       = fs.String("pdf-dir", ".", "directory to write downloaded PDFs into")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: aade-invoice [flags] <amount>\n\nRegisters one invoice in myDATA. The amount is the net value in euros.\n\nflags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if *pdfMark != 0 {
		client, _, err := common.Client()
		if err != nil {
			return err
		}
		return downloadPDF(client, *pdfMark, *pdfDir)
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one amount")
	}
	netCents, err := mydata.ParseAmount(fs.Arg(0))
	if err != nil {
		return err
	}

	resolvedTemplatePath, err := resolveTemplatePath(*templatePath, "templates")
	if err != nil {
		return err
	}
	tmpl, err := mydata.LoadTemplate(resolvedTemplatePath)
	if err != nil {
		return err
	}
	creds, err := common.Credentials()
	if err != nil {
		return err
	}

	issueDate := *date
	if issueDate == "" {
		issueDate = time.Now().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", issueDate); err != nil {
		return fmt.Errorf("--date must be YYYY-MM-DD")
	}

	number := *aa
	if number == 0 {
		number = tmpl.NextAa
	}

	invoice := tmpl.Build(netCents, issueDate, number, creds.VatNumber)
	body, err := invoice.RenderXML()
	if err != nil {
		return err
	}

	if *dryRun {
		fmt.Print(string(body))
		return nil
	}

	fmt.Printf("submitting %s %d for %s %s to %s\n",
		invoice.Series, invoice.Aa, invoice.Gross(), invoice.Currency, creds.Name)

	client := mydata.NewClient(creds)
	doc, err := client.SendInvoices(body)
	if err != nil {
		return err
	}

	failed := false
	var marks []int64
	for _, r := range doc.Responses {
		if len(r.Errors) > 0 {
			failed = true
			fmt.Fprintf(os.Stderr, "rejected (status %s):\n", r.StatusCode)
			for _, e := range r.Errors {
				fmt.Fprintf(os.Stderr, "  [%s] %s\n", e.Code, e.Message)
			}
			continue
		}
		marks = append(marks, r.InvoiceMark)
		fmt.Printf("registered\n  MARK: %d\n  uid:  %s\n", r.InvoiceMark, r.InvoiceUid)
		if r.QrURL != "" {
			fmt.Printf("  QR:   %s\n", r.QrURL)
		}
	}
	if failed {
		return fmt.Errorf("submission rejected; the template was not advanced")
	}

	// Only bump the local counter once AADE has actually accepted the number.
	if *aa == 0 {
		tmpl.NextAa = number + 1
		if err := tmpl.Save(resolvedTemplatePath); err != nil {
			return fmt.Errorf("invoice registered but the counter could not be saved: %w", err)
		}
	}

	if *pdf {
		for _, mark := range marks {
			if err := downloadPDF(client, mark, *pdfDir); err != nil {
				// The invoice is registered either way, so this is not fatal.
				fmt.Fprintln(os.Stderr, "warning:", err)
			}
		}
	}
	return nil
}

func resolveTemplatePath(path, templatesDir string) (string, error) {
	if path != "" {
		return path, nil
	}

	matches, err := filepath.Glob(filepath.Join(templatesDir, "*.json"))
	if err != nil {
		return "", fmt.Errorf("find invoice templates: %w", err)
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("--template is required unless %s contains exactly one JSON file (found %d)", templatesDir, len(matches))
	}
	return matches[0], nil
}

func downloadPDF(client *mydata.Client, mark int64, dir string) error {
	path, err := client.DownloadInvoicePDF(mark, dir)
	if err != nil {
		return err
	}
	fmt.Printf("  PDF:  %s\n", path)
	return nil
}

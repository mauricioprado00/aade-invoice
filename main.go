package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		production   = flag.Bool("prod", false, "submit to the production myDATA API instead of the sandbox")
		dryRun       = flag.Bool("dry-run", false, "print the XML that would be submitted and stop")
		templatePath = flag.String("template", "invoice-template.json", "invoice template to fill in")
		envPath      = flag.String("env", ".env", "file holding the API credentials")
		date         = flag.String("date", "", "issue date as YYYY-MM-DD (default: today)")
		aa           = flag.Int("aa", 0, "invoice number within the series (default: nextAa from the template)")
		pdf          = flag.Bool("pdf", false, "after registering, download AADE's PDF of the invoice")
		pdfMark      = flag.Int64("pdf-mark", 0, "download the PDF for an already registered MARK and exit")
		pdfDir       = flag.String("pdf-dir", ".", "directory to write downloaded PDFs into")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: aade-invoice [flags] <amount>\n\nRegisters one invoice in myDATA. The amount is the net value in euros.\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *pdfMark != 0 {
		if err := loadDotEnv(*envPath); err != nil {
			return err
		}
		creds, err := credentials(*production)
		if err != nil {
			return err
		}
		return downloadPDF(NewClient(creds), *pdfMark, *pdfDir)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		return fmt.Errorf("expected exactly one amount")
	}
	netCents, err := parseAmount(flag.Arg(0))
	if err != nil {
		return err
	}

	if err := loadDotEnv(*envPath); err != nil {
		return err
	}
	creds, err := credentials(*production)
	if err != nil {
		return err
	}
	tmpl, err := loadTemplate(*templatePath)
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

	invoice := tmpl.build(netCents, issueDate, number, creds.VatNumber)
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

	doc, err := NewClient(creds).SendInvoices(body)
	if err != nil {
		return err
	}

	client := NewClient(creds)
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
		if err := tmpl.save(*templatePath); err != nil {
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

func downloadPDF(client *Client, mark int64, dir string) error {
	path, err := client.DownloadInvoicePDF(mark, dir)
	if err != nil {
		return err
	}
	fmt.Printf("  PDF:  %s\n", path)
	return nil
}

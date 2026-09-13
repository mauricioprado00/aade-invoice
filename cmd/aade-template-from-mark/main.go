// Command aade-template-from-mark builds an invoice template from an invoice
// already registered in myDATA, identified by its MARK. It is meant to seed
// templates/ for a new counterpart from a real invoice, without having to
// hand-type the party details and classifications.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mauricioprado00/aade-invoice/internal/cli"
	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func main() { cli.Main(run) }

func run() error {
	fs := flag.NewFlagSet("aade-template-from-mark", flag.ExitOnError)
	common := cli.Register(fs)
	var (
		templatesDir = fs.String("templates-dir", "templates", "directory to write the template into")
		force        = fs.Bool("force", false, "overwrite the template file if it already exists")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: aade-template-from-mark [flags] <mark>\n\nFetches the invoice registered under a MARK and writes a reusable\ninvoice template under templates/, named after the counterpart.\n\nflags:\n")
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

	tmpl, name, err := templateFromDoc(doc, creds.VatNumber)
	if err != nil {
		return err
	}

	path := filepath.Join(*templatesDir, slugify(name)+".json")
	if _, err := os.Stat(path); err == nil && !*force {
		return fmt.Errorf("%s already exists; pass --force to overwrite", path)
	}

	if err := os.MkdirAll(*templatesDir, 0o755); err != nil {
		return err
	}
	if err := tmpl.Save(path); err != nil {
		return err
	}

	fmt.Printf("wrote %s\n", path)
	return nil
}

// templateFromDoc builds a Template out of a myDATA document, plus the name
// of the counterpart the template is for. issuerVat identifies which side of
// the document is "us"; the other side becomes the template's counterpart.
func templateFromDoc(d *mydata.Doc, issuerVat string) (*mydata.Template, string, error) {
	if len(d.Details) == 0 {
		return nil, "", fmt.Errorf("MARK %d has no invoice lines to build a template from", d.Mark)
	}
	line := d.Details[0]

	counterpart := d.Counterpart
	if !d.IssuedBy(issuerVat) {
		counterpart = d.Issuer
	}
	if counterpart.Name == "" && counterpart.VatNumber == "" {
		return nil, "", fmt.Errorf("MARK %d carries no counterpart name or VAT number to name the template after", d.Mark)
	}

	nextAa, err := strconv.Atoi(strings.TrimSpace(d.Header.Aa))
	if err != nil {
		return nil, "", fmt.Errorf("invoice number %q is not numeric: %w", d.Header.Aa, err)
	}

	vatRate, err := vatRatePercent(line)
	if err != nil {
		return nil, "", err
	}

	incomeType, incomeCategory := "", ""
	if len(line.Income) > 0 {
		incomeType, incomeCategory = line.Income[0].Type, line.Income[0].Category
	}

	paymentMethodType := 0
	if len(d.PaymentMethods) > 0 {
		paymentMethodType = d.PaymentMethods[0].Type
	}

	tmpl := &mydata.Template{
		Issuer: mydata.Party{
			Country: "GR",
			Branch:  0,
		},
		Counterpart: mydata.Party{
			VatNumber:  counterpart.VatNumber,
			Country:    counterpart.Country,
			Branch:     counterpart.Branch,
			Name:       counterpart.Name,
			Number:     counterpart.Number,
			PostalCode: counterpart.PostCode,
			City:       counterpart.City,
		},
		Series:                       d.Header.Series,
		InvoiceType:                  d.Header.InvoiceType,
		Currency:                     d.Header.Currency,
		VatCategory:                  line.VatCategory,
		VatExemptionCategory:         line.VatExemptionCategory,
		VatRatePercent:               vatRate,
		IncomeClassificationType:     incomeType,
		IncomeClassificationCategory: incomeCategory,
		PaymentMethodType:            paymentMethodType,
		NextAa:                       nextAa + 1,
	}

	name := counterpart.Name
	if name == "" {
		name = counterpart.VatNumber
	}
	return tmpl, name, nil
}

// vatRatePercent recovers the rate from the ratio of vatAmount to netValue.
// An exemption means no rate applies at all.
func vatRatePercent(line mydata.DocDetail) (float64, error) {
	if line.VatExemptionCategory != 0 {
		return 0, nil
	}
	net, err := strconv.ParseFloat(line.NetValue, 64)
	if err != nil || net == 0 {
		return 0, nil
	}
	vat, err := strconv.ParseFloat(line.VatAmount, 64)
	if err != nil {
		return 0, nil
	}
	return math.Round(vat/net*10000) / 100, nil
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// slugify turns a company name into a filesystem- and shell-friendly slug,
// e.g. "TEEL TECHNOLOGIES" -> "teel-technologies".
func slugify(name string) string {
	s := nonAlnum.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "template"
	}
	return s
}

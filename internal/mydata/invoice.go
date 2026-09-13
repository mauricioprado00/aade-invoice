package mydata

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/template"
)

// Invoice is one outgoing invoice, ready to be rendered as InvoicesDoc XML.
type Invoice struct {
	Issuer      Party
	Counterpart Party

	Series      string
	Aa          int
	IssueDate   string // YYYY-MM-DD
	InvoiceType string
	Currency    string

	VatCategory          int
	VatExemptionCategory int
	PaymentMethodType    int

	IncomeClassificationType     string
	IncomeClassificationCategory string

	NetCents int64
	VatCents int64
}

// Build fills an Invoice from the template plus the values that change per
// invoice. issuerVat is the VAT number tied to the API credentials, used when
// the template does not spell one out.
func (t *Template) Build(netCents int64, issueDate string, aa int, issuerVat string) *Invoice {
	issuer := t.Issuer
	if issuer.VatNumber == "" {
		issuer.VatNumber = issuerVat
	}

	vatCents := int64(0)
	if t.VatExemptionCategory == 0 && t.VatRatePercent > 0 {
		vatCents = int64(math.Round(float64(netCents) * t.VatRatePercent / 100))
	}

	return &Invoice{
		Issuer:                       issuer,
		Counterpart:                  t.Counterpart,
		Series:                       t.Series,
		Aa:                           aa,
		IssueDate:                    issueDate,
		InvoiceType:                  t.InvoiceType,
		Currency:                     t.Currency,
		VatCategory:                  t.VatCategory,
		VatExemptionCategory:         t.VatExemptionCategory,
		PaymentMethodType:            t.PaymentMethodType,
		IncomeClassificationType:     t.IncomeClassificationType,
		IncomeClassificationCategory: t.IncomeClassificationCategory,
		NetCents:                     netCents,
		VatCents:                     vatCents,
	}
}

func (i *Invoice) Net() string   { return formatCents(i.NetCents) }
func (i *Invoice) Vat() string   { return formatCents(i.VatCents) }
func (i *Invoice) Gross() string { return formatCents(i.NetCents + i.VatCents) }

// The element order below follows AadeBookInvoiceType in
// docs/aade/xsd/InvoicesDoc-v2.0.2.xsd; the schema is a sequence, so reordering
// anything here makes the submission invalid.
var invoiceDocTemplate = template.Must(template.New("InvoicesDoc").Funcs(template.FuncMap{
	"xml": escapeXML,
}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<InvoicesDoc xmlns="http://www.aade.gr/myDATA/invoice/v1.0"
             xmlns:icls="https://www.aade.gr/myDATA/incomeClassificaton/v1.0"
             xmlns:ecls="https://www.aade.gr/myDATA/expensesClassificaton/v1.0">
  <invoice>
    <issuer>
      <vatNumber>{{xml .Issuer.VatNumber}}</vatNumber>
      <country>{{xml .Issuer.Country}}</country>
      <branch>{{.Issuer.Branch}}</branch>
    </issuer>
    <counterpart>
      <vatNumber>{{xml .Counterpart.VatNumber}}</vatNumber>
      <country>{{xml .Counterpart.Country}}</country>
      <branch>{{.Counterpart.Branch}}</branch>
{{- if .Counterpart.Name}}
      <name>{{xml .Counterpart.Name}}</name>
{{- end}}
{{- if .Counterpart.HasAddress}}
      <address>
{{- if .Counterpart.Number}}
        <number>{{xml .Counterpart.Number}}</number>
{{- end}}
{{- if .Counterpart.PostalCode}}
        <postalCode>{{xml .Counterpart.PostalCode}}</postalCode>
{{- end}}
{{- if .Counterpart.City}}
        <city>{{xml .Counterpart.City}}</city>
{{- end}}
      </address>
{{- end}}
    </counterpart>
    <invoiceHeader>
      <series>{{xml .Series}}</series>
      <aa>{{.Aa}}</aa>
      <issueDate>{{.IssueDate}}</issueDate>
      <invoiceType>{{xml .InvoiceType}}</invoiceType>
      <vatPaymentSuspension>false</vatPaymentSuspension>
      <currency>{{xml .Currency}}</currency>
    </invoiceHeader>
{{- if .PaymentMethodType}}
    <paymentMethods>
      <paymentMethodDetails>
        <type>{{.PaymentMethodType}}</type>
        <amount>0</amount>
      </paymentMethodDetails>
    </paymentMethods>
{{- end}}
    <invoiceDetails>
      <lineNumber>1</lineNumber>
      <netValue>{{.Net}}</netValue>
      <vatCategory>{{.VatCategory}}</vatCategory>
      <vatAmount>{{.Vat}}</vatAmount>
{{- if .VatExemptionCategory}}
      <vatExemptionCategory>{{.VatExemptionCategory}}</vatExemptionCategory>
{{- end}}
      <incomeClassification>
        <icls:classificationType>{{xml .IncomeClassificationType}}</icls:classificationType>
        <icls:classificationCategory>{{xml .IncomeClassificationCategory}}</icls:classificationCategory>
        <icls:amount>{{.Net}}</icls:amount>
      </incomeClassification>
    </invoiceDetails>
    <invoiceSummary>
      <totalNetValue>{{.Net}}</totalNetValue>
      <totalVatAmount>{{.Vat}}</totalVatAmount>
      <totalWithheldAmount>0.00</totalWithheldAmount>
      <totalFeesAmount>0.00</totalFeesAmount>
      <totalStampDutyAmount>0.00</totalStampDutyAmount>
      <totalOtherTaxesAmount>0.00</totalOtherTaxesAmount>
      <totalDeductionsAmount>0.00</totalDeductionsAmount>
      <totalGrossValue>{{.Gross}}</totalGrossValue>
      <incomeClassification>
        <icls:classificationType>{{xml .IncomeClassificationType}}</icls:classificationType>
        <icls:classificationCategory>{{xml .IncomeClassificationCategory}}</icls:classificationCategory>
        <icls:amount>{{.Net}}</icls:amount>
      </incomeClassification>
    </invoiceSummary>
  </invoice>
</InvoicesDoc>
`))

// HasAddress is exported for the template.
func (p Party) HasAddress() bool { return p.hasAddress() }

func (i *Invoice) RenderXML() ([]byte, error) {
	var buf bytes.Buffer
	if err := invoiceDocTemplate.Execute(&buf, i); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func escapeXML(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func formatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// ParseAmount turns "1234.56" into 123456 cents. It refuses anything with more
// than two decimals rather than silently rounding money.
func ParseAmount(s string) (int64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	whole, frac, _ := strings.Cut(s, ".")
	if len(frac) > 2 {
		return 0, fmt.Errorf("amount %q has more than two decimals", s)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	cents, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q", s)
	}
	if cents <= 0 {
		return 0, fmt.Errorf("amount must be positive")
	}
	return cents, nil
}

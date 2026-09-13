package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Template holds the parts of an invoice that are the same every time. Only the
// amount, the issue date and the sequence number change per invoice.
type Template struct {
	Issuer      Party  `json:"issuer"`
	Counterpart Party  `json:"counterpart"`
	Series      string `json:"series"`
	InvoiceType string `json:"invoiceType"`
	Currency    string `json:"currency"`

	// VAT category 7 with an exemption category means no VAT is charged, which
	// is the usual case for services billed outside the EU. VatRatePercent is
	// only consulted for the other categories.
	VatCategory          int     `json:"vatCategory"`
	VatExemptionCategory int     `json:"vatExemptionCategory,omitempty"`
	VatRatePercent       float64 `json:"vatRatePercent"`

	IncomeClassificationType     string `json:"incomeClassificationType"`
	IncomeClassificationCategory string `json:"incomeClassificationCategory"`

	PaymentMethodType int `json:"paymentMethodType"`

	// NextAa is the sequence number the next invoice will carry. It is bumped
	// locally after a successful submission; --aa overrides it.
	NextAa int `json:"nextAa"`
}

// Party is an issuer or a counterpart.
type Party struct {
	// VatNumber may be left empty on the issuer, in which case the VAT number
	// registered with the API credentials is used.
	VatNumber  string `json:"vatNumber,omitempty"`
	Country    string `json:"country"`
	Branch     int    `json:"branch"`
	Name       string `json:"name,omitempty"`
	Number     string `json:"number,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	City       string `json:"city,omitempty"`
}

func (p Party) hasAddress() bool {
	return p.Number != "" || p.PostalCode != "" || p.City != ""
}

func loadTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Template
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if t.NextAa < 1 {
		return nil, fmt.Errorf("%s: nextAa must be 1 or greater", path)
	}
	return &t, nil
}

func (t *Template) save(path string) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

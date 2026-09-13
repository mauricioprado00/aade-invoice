package main

import (
	"testing"

	"github.com/mauricioprado00/aade-invoice/internal/mydata"
)

func TestSlugify(t *testing.T) {
	tests := []struct{ in, want string }{
		{"TEEL TECHNOLOGIES", "teel-technologies"},
		{"  Acme, Inc.  ", "acme-inc"},
		{"", "template"},
		{"---", "template"},
	}
	for _, tt := range tests {
		if got := slugify(tt.in); got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTemplateFromDocIssuedByUs(t *testing.T) {
	doc := &mydata.Doc{
		Mark:   1,
		Issuer: mydata.DocParty{VatNumber: "177077414", Country: "GR", Branch: 0},
		Counterpart: mydata.DocParty{
			VatNumber: "000000000", Country: "US", Branch: 0,
			Name: "EXAMPLE CUSTOMER", Number: "0", PostCode: "06851", City: "Norwalk",
		},
		Header: mydata.DocHeader{
			Series: "A", Aa: "84", InvoiceType: "2.3", Currency: "EUR",
		},
		PaymentMethods: []mydata.DocPaymentMethod{{Type: 2, Amount: "0"}},
		Details: []mydata.DocDetail{{
			LineNumber: 1, NetValue: "3221.27", VatCategory: 7, VatAmount: "0",
			VatExemptionCategory: 6,
			Income: []mydata.Classification{{
				Type: "E3_561_006", Category: "category1_3", Amount: "3221.27",
			}},
		}},
	}

	tmpl, name, err := templateFromDoc(doc, "177077414")
	if err != nil {
		t.Fatal(err)
	}
	if name != "EXAMPLE CUSTOMER" {
		t.Fatalf("name = %q, want EXAMPLE CUSTOMER", name)
	}
	if tmpl.Issuer.VatNumber != "" || tmpl.Issuer.Country != "GR" {
		t.Fatalf("issuer = %+v", tmpl.Issuer)
	}
	if tmpl.Counterpart.Name != "EXAMPLE CUSTOMER" || tmpl.Counterpart.VatNumber != "000000000" {
		t.Fatalf("counterpart = %+v", tmpl.Counterpart)
	}
	if tmpl.Series != "A" || tmpl.InvoiceType != "2.3" || tmpl.Currency != "EUR" {
		t.Fatalf("header fields = %+v", tmpl)
	}
	if tmpl.VatCategory != 7 || tmpl.VatExemptionCategory != 6 || tmpl.VatRatePercent != 0 {
		t.Fatalf("vat fields = %+v", tmpl)
	}
	if tmpl.IncomeClassificationType != "E3_561_006" || tmpl.IncomeClassificationCategory != "category1_3" {
		t.Fatalf("income classification = %+v", tmpl)
	}
	if tmpl.PaymentMethodType != 2 {
		t.Fatalf("paymentMethodType = %d, want 2", tmpl.PaymentMethodType)
	}
	if tmpl.NextAa != 85 {
		t.Fatalf("nextAa = %d, want 85", tmpl.NextAa)
	}
}

func TestTemplateFromDocReceivedByUs(t *testing.T) {
	doc := &mydata.Doc{
		Mark: 2,
		Issuer: mydata.DocParty{
			VatNumber: "999999999", Country: "GR", Branch: 0, Name: "SOME SUPPLIER",
		},
		Counterpart: mydata.DocParty{VatNumber: "177077414", Country: "GR", Branch: 0},
		Header:      mydata.DocHeader{Series: "0", Aa: "1", InvoiceType: "14.5", Currency: "EUR"},
		Details: []mydata.DocDetail{{
			LineNumber: 1, NetValue: "100.00", VatCategory: 1, VatAmount: "24.00",
			Expenses: []mydata.Classification{{Type: "E3_102", Category: "category1_1", Amount: "100.00"}},
		}},
	}

	tmpl, name, err := templateFromDoc(doc, "177077414")
	if err != nil {
		t.Fatal(err)
	}
	if name != "SOME SUPPLIER" {
		t.Fatalf("name = %q, want SOME SUPPLIER", name)
	}
	if tmpl.Counterpart.VatNumber != "999999999" {
		t.Fatalf("counterpart = %+v, want the issuer (999999999)", tmpl.Counterpart)
	}
	if tmpl.VatRatePercent != 24 {
		t.Fatalf("vatRatePercent = %v, want 24", tmpl.VatRatePercent)
	}
	if tmpl.NextAa != 2 {
		t.Fatalf("nextAa = %d, want 2", tmpl.NextAa)
	}
}

func TestTemplateFromDocRequiresLinesAndCounterpartIdentity(t *testing.T) {
	if _, _, err := templateFromDoc(&mydata.Doc{}, "177077414"); err == nil {
		t.Fatal("expected an error for a document with no invoice lines")
	}

	doc := &mydata.Doc{
		Details: []mydata.DocDetail{{LineNumber: 1, NetValue: "1.00"}},
	}
	if _, _, err := templateFromDoc(doc, "177077414"); err == nil {
		t.Fatal("expected an error when the counterpart has neither name nor VAT number")
	}
}

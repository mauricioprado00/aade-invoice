package mydata

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

// RequestedDoc is the reply to RequestDocs and RequestTransmittedDocs; see
// docs/aade/xsd/requestedInvoicesDoc-v2.0.2.xsd.
type RequestedDoc struct {
	Continuation *ContinuationToken `xml:"continuationToken"`
	Invoices     []Doc              `xml:"invoicesDoc>invoice"`
}

// ContinuationToken carries the keys that fetch the next page of a result set
// too large to return in one call.
type ContinuationToken struct {
	NextPartitionKey string `xml:"nextPartitionKey"`
	NextRowKey       string `xml:"nextRowKey"`
}

// Doc is one invoice as myDATA holds it. Only the fields worth showing are
// mapped; the full type is AadeBookInvoiceType in the XSD.
type Doc struct {
	Uid                   string             `xml:"uid"`
	Mark                  int64              `xml:"mark"`
	Issuer                DocParty           `xml:"issuer"`
	Counterpart           DocParty           `xml:"counterpart"`
	Header                DocHeader          `xml:"invoiceHeader"`
	PaymentMethods        []DocPaymentMethod `xml:"paymentMethods>paymentMethodDetails"`
	Details               []DocDetail        `xml:"invoiceDetails"`
	Summary               DocSummary         `xml:"invoiceSummary"`
	InvoiceFormat         string             `xml:"invoiceFormat"`
	QrCodeURL             string             `xml:"qrCodeUrl"`
	DownloadingInvoiceURL string             `xml:"downloadingInvoiceUrl"`
}

type DocParty struct {
	VatNumber string `xml:"vatNumber"`
	Country   string `xml:"country"`
	Branch    int    `xml:"branch"`
	Name      string `xml:"name"`
	Number    string `xml:"address>number"`
	City      string `xml:"address>city"`
	PostCode  string `xml:"address>postalCode"`
}

// DocPaymentMethod is one entry of <paymentMethods><paymentMethodDetails>.
type DocPaymentMethod struct {
	Type   int    `xml:"type"`
	Amount string `xml:"amount"`
}

type DocHeader struct {
	Series      string `xml:"series"`
	Aa          string `xml:"aa"`
	IssueDate   string `xml:"issueDate"`
	InvoiceType string `xml:"invoiceType"`
	Currency    string `xml:"currency"`
}

type DocDetail struct {
	LineNumber           int              `xml:"lineNumber"`
	NetValue             string           `xml:"netValue"`
	VatCategory          int              `xml:"vatCategory"`
	VatAmount            string           `xml:"vatAmount"`
	VatExemptionCategory int              `xml:"vatExemptionCategory"`
	Income               []Classification `xml:"incomeClassification"`
	Expenses             []Classification `xml:"expensesClassification"`
}

type Classification struct {
	Type     string `xml:"classificationType"`
	Category string `xml:"classificationCategory"`
	Amount   string `xml:"amount"`
}

type DocSummary struct {
	TotalNetValue   string `xml:"totalNetValue"`
	TotalVatAmount  string `xml:"totalVatAmount"`
	TotalWithheld   string `xml:"totalWithheldAmount"`
	TotalGrossValue string `xml:"totalGrossValue"`
}

// IssuedBy reports whether this entity issued the document, as opposed to
// having received it.
func (d Doc) IssuedBy(vatNumber string) bool { return d.Issuer.VatNumber == vatNumber }

// Query is the set of filters RequestDocs and RequestTransmittedDocs accept.
// Every field is optional except AfterMark, which myDATA requires; zero means
// "from the beginning".
type Query struct {
	AfterMark int64
	MaxMark   int64

	// DateFrom and DateTo filter on issue date and must be dd/MM/yyyy, which is
	// what myDATA expects here even though invoices carry ISO dates. Giving
	// only one of the two searches that single date rather than an open range.
	DateFrom string
	DateTo   string

	CounterVatNumber string
	EntityVatNumber  string
	InvType          string

	Continuation *ContinuationToken
}

func (q Query) values() url.Values {
	v := url.Values{}
	v.Set("mark", fmt.Sprint(q.AfterMark))
	if q.MaxMark > 0 {
		v.Set("maxMark", fmt.Sprint(q.MaxMark))
	}
	for name, value := range map[string]string{
		"dateFrom":         q.DateFrom,
		"dateTo":           q.DateTo,
		"counterVatNumber": q.CounterVatNumber,
		"entityVatNumber":  q.EntityVatNumber,
		"invType":          q.InvType,
	} {
		if value != "" {
			v.Set(name, value)
		}
	}
	if q.Continuation != nil {
		v.Set("nextPartitionKey", q.Continuation.NextPartitionKey)
		v.Set("nextRowKey", q.Continuation.NextRowKey)
	}
	return v
}

// RequestTransmittedDocs returns documents this entity transmitted itself.
func (c *Client) RequestTransmittedDocs(q Query) (*RequestedDoc, error) {
	var doc RequestedDoc
	if err := c.get("/RequestTransmittedDocs", q.values(), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// RequestDocs returns documents that concern this entity, which is mostly the
// ones other people issued to it.
func (c *Client) RequestDocs(q Query) (*RequestedDoc, error) {
	var doc RequestedDoc
	if err := c.get("/RequestDocs", q.values(), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// FetchAll walks the continuation tokens and returns every matching document.
// fetch is RequestDocs or RequestTransmittedDocs.
func FetchAll(fetch func(Query) (*RequestedDoc, error), q Query, maxPages int) ([]Doc, error) {
	var all []Doc
	for page := 0; page < maxPages; page++ {
		doc, err := fetch(q)
		if err != nil {
			return all, err
		}
		all = append(all, doc.Invoices...)
		if doc.Continuation == nil || doc.Continuation.NextRowKey == "" {
			return all, nil
		}
		q.Continuation = doc.Continuation
	}
	return all, fmt.Errorf("stopped after %d pages; narrow the filters", maxPages)
}

// FindMark returns the single document carrying this MARK, from either
// direction: documents we transmitted, then documents addressed to us.
func (c *Client) FindMark(mark int64) (*Doc, error) {
	for _, fetch := range []func(Query) (*RequestedDoc, error){
		c.RequestTransmittedDocs,
		c.RequestDocs,
	} {
		doc, err := fetch(Query{AfterMark: mark - 1, MaxMark: mark})
		if err != nil {
			return nil, err
		}
		for i := range doc.Invoices {
			if doc.Invoices[i].Mark == mark {
				return &doc.Invoices[i], nil
			}
		}
	}
	return nil, fmt.Errorf("MARK %d was not found", mark)
}

// downloadURLForMark returns AADE's own PDF link for an invoice. The link is
// only present on invoices AADE renders itself; invoices merely transmitted by
// an ERP carry none.
func (c *Client) downloadURLForMark(mark int64) (string, error) {
	doc, err := c.FindMark(mark)
	if err != nil {
		return "", err
	}
	if doc.DownloadingInvoiceURL == "" {
		return "", fmt.Errorf("MARK %d carries no downloadingInvoiceUrl: AADE only renders a PDF for invoices issued through its own timologio application", mark)
	}
	return doc.DownloadingInvoiceURL, nil
}

// NormalizeDate accepts either an ISO date or the dd/MM/yyyy myDATA wants, and
// returns the latter.
func NormalizeDate(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if parts := strings.Split(s, "-"); len(parts) == 3 {
		return fmt.Sprintf("%s/%s/%s", parts[2], parts[1], parts[0]), nil
	}
	if len(strings.Split(s, "/")) == 3 {
		return s, nil
	}
	return "", fmt.Errorf("date %q must be YYYY-MM-DD", s)
}

// MarshalDocs renders documents back to XML, for piping into other tools.
func MarshalDocs(docs []Doc) ([]byte, error) {
	type wrapper struct {
		XMLName xml.Name `xml:"invoicesDoc"`
		Invoice []Doc    `xml:"invoice"`
	}
	return xml.MarshalIndent(wrapper{Invoice: docs}, "", "  ")
}

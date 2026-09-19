package mydata

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Paths of the four read-only "book" methods, spec §4.2.8–4.2.11. They are
// exported so a command can print the URL it would call without calling it.
const (
	PathMyIncome   = "/RequestMyIncome"
	PathMyExpenses = "/RequestMyExpenses"
	PathVatInfo    = "/RequestVatInfo"
	PathE3Info     = "/RequestE3Info"
)

// BookQuery is what RequestMyIncome and RequestMyExpenses accept. Unlike
// RequestDocs these two require both dates, in dd/MM/yyyy.
type BookQuery struct {
	DateFrom string
	DateTo   string

	CounterVatNumber string
	EntityVatNumber  string
	InvType          string

	Continuation *ContinuationToken
}

func (q BookQuery) Values() url.Values {
	v := url.Values{}
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
	addContinuation(v, q.Continuation)
	return v
}

// InfoQuery is what RequestVatInfo and RequestE3Info accept. Paging only
// applies when the results are not grouped per day.
type InfoQuery struct {
	DateFrom string
	DateTo   string

	EntityVatNumber string
	GroupedPerDay   bool

	Continuation *ContinuationToken
}

func (q InfoQuery) Values() url.Values {
	v := url.Values{}
	for name, value := range map[string]string{
		"dateFrom":        q.DateFrom,
		"dateTo":          q.DateTo,
		"entityVatNumber": q.EntityVatNumber,
	} {
		if value != "" {
			v.Set(name, value)
		}
	}
	if q.GroupedPerDay {
		v.Set("GroupedPerDay", "true")
	}
	addContinuation(v, q.Continuation)
	return v
}

// UnmarshalXML matches the two keys case-insensitively: the book methods have
// no published XSD, and their token is the same shape under either casing.
func (t *ContinuationToken) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			var text string
			if err := d.DecodeElement(&text, &el); err != nil {
				return err
			}
			switch strings.ToLower(el.Name.Local) {
			case "nextpartitionkey":
				t.NextPartitionKey = strings.TrimSpace(text)
			case "nextrowkey":
				t.NextRowKey = strings.TrimSpace(text)
			}
		case xml.EndElement:
			return nil
		}
	}
}

func addContinuation(v url.Values, t *ContinuationToken) {
	if t == nil {
		return
	}
	v.Set("nextPartitionKey", t.NextPartitionKey)
	v.Set("nextRowKey", t.NextRowKey)
}

// RequestedBookInfo is the reply to RequestMyIncome and RequestMyExpenses,
// described in spec §6.3. AADE publishes no XSD for it, so the element names
// are matched case-insensitively rather than trusted to be exactly what the
// table in the spec prints.
type RequestedBookInfo struct {
	Continuation *ContinuationToken
	Entries      []BookEntry
}

// BookEntry is one aggregated line: one combination of counterpart, issue date
// and invoice type, with the totals and the number of documents behind it.
type BookEntry struct {
	CounterVatNumber  string
	IssueDate         string
	InvType           string
	SelfPricing       bool
	InvoiceDetailType string
	NetValue          string
	VatAmount         string
	WithheldAmount    string
	OtherTaxesAmount  string
	StampDutyAmount   string
	FeesAmount        string
	DeductionsAmount  string
	ThirdPartyAmount  string
	GrossValue        string
	Count             int
	MinMark           string
	MaxMark           string
}

func (r *RequestedBookInfo) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if strings.EqualFold(el.Name.Local, "continuationToken") {
				var t ContinuationToken
				if err := d.DecodeElement(&t, &el); err != nil {
					return err
				}
				r.Continuation = &t
				continue
			}
			var entry BookEntry
			if err := d.DecodeElement(&entry, &el); err != nil {
				return err
			}
			r.Entries = append(r.Entries, entry)
		case xml.EndElement:
			return nil
		}
	}
}

func (b *BookEntry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	fields := map[string]*string{
		"countervatnumber":  &b.CounterVatNumber,
		"issuedate":         &b.IssueDate,
		"invtype":           &b.InvType,
		"invoicedetailtype": &b.InvoiceDetailType,
		"netvalue":          &b.NetValue,
		"vatamount":         &b.VatAmount,
		"withheldamount":    &b.WithheldAmount,
		"othertaxesamount":  &b.OtherTaxesAmount,
		"stampdutyamount":   &b.StampDutyAmount,
		"feesamount":        &b.FeesAmount,
		"deductionsamount":  &b.DeductionsAmount,
		"thirdpartyamount":  &b.ThirdPartyAmount,
		"grossvalue":        &b.GrossValue,
		"minmark":           &b.MinMark,
		"maxmark":           &b.MaxMark,
	}

	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			var text string
			if err := d.DecodeElement(&text, &el); err != nil {
				return err
			}
			text = strings.TrimSpace(text)
			name := strings.ToLower(el.Name.Local)
			switch {
			case fields[name] != nil:
				*fields[name] = text
			case name == "selfpricing":
				b.SelfPricing = strings.EqualFold(text, "true") || text == "1"
			case name == "count":
				b.Count, _ = strconv.Atoi(text)
			}
		case xml.EndElement:
			return nil
		}
	}
}

// RequestedVatInfo is the reply to RequestVatInfo; see
// docs/aade/xsd/RequestVatInfoResponse-v2.0.2.xsd.
type RequestedVatInfo struct {
	XMLName      xml.Name           `xml:"RequestedVatInfo"`
	Continuation *ContinuationToken `xml:"continuationToken"`
	Records      []VatInfo          `xml:"VatInfo"`
}

// VatInfo is one record of the VAT book. The thirty-odd Vat### boxes are kept
// as an ordered list rather than as fields: every one of them is optional, a
// record carries a handful, and a list prints and totals without naming them
// one by one.
type VatInfo struct {
	Mark        string
	IsCancelled bool
	IssueDate   string
	Amounts     []VatAmount
}

type VatAmount struct {
	Name  string
	Value string
}

func (v *VatInfo) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			var text string
			if err := d.DecodeElement(&text, &el); err != nil {
				return err
			}
			text = strings.TrimSpace(text)
			switch name := el.Name.Local; strings.ToLower(name) {
			case "mark":
				v.Mark = text
			case "iscancelled":
				v.IsCancelled = strings.EqualFold(text, "true")
			case "issuedate":
				v.IssueDate = text
			default:
				v.Amounts = append(v.Amounts, VatAmount{Name: name, Value: text})
			}
		case xml.EndElement:
			return nil
		}
	}
}

// RequestedE3Info is the reply to RequestE3Info; see
// docs/aade/xsd/RequestE3InfoResponse-v2.0.2.xsd.
type RequestedE3Info struct {
	XMLName      xml.Name           `xml:"RequestedE3Info"`
	Continuation *ContinuationToken `xml:"continuationToken"`
	Records      []E3Info           `xml:"E3Info"`
}

type E3Info struct {
	VatNumber     string `xml:"V_Afm"`
	Mark          string `xml:"V_Mark"`
	Book          string `xml:"vBook"`
	IsCancelled   bool   `xml:"IsCancelled"`
	IssueDate     string `xml:"IssueDate"`
	ClassCategory string `xml:"V_Class_Category"`
	ClassType     string `xml:"V_Class_Type"`
	ClassValue    string `xml:"V_Class_Value"`
}

// RequestMyIncome returns the income side of the books, aggregated per
// counterpart, issue date and invoice type.
func (c *Client) RequestMyIncome(q BookQuery) (*RequestedBookInfo, error) {
	return c.requestBook(PathMyIncome, q)
}

// RequestMyExpenses is RequestMyIncome for the expense side.
func (c *Client) RequestMyExpenses(q BookQuery) (*RequestedBookInfo, error) {
	return c.requestBook(PathMyExpenses, q)
}

func (c *Client) requestBook(path string, q BookQuery) (*RequestedBookInfo, error) {
	var doc RequestedBookInfo
	if err := c.get(path, q.Values(), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// RequestVatInfo returns the VAT inputs and outputs, per invoice or, with
// GroupedPerDay, one record per day.
func (c *Client) RequestVatInfo(q InfoQuery) (*RequestedVatInfo, error) {
	var doc RequestedVatInfo
	if err := c.get(PathVatInfo, q.Values(), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// RequestE3Info returns the E3 classification lines.
func (c *Client) RequestE3Info(q InfoQuery) (*RequestedE3Info, error) {
	var doc RequestedE3Info
	if err := c.get(PathE3Info, q.Values(), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// FetchAllPages walks the continuation tokens of any of the book methods and
// returns every record. fetch takes the token of the previous page, nil for
// the first one.
func FetchAllPages[T any](fetch func(*ContinuationToken) ([]T, *ContinuationToken, error), maxPages int) ([]T, error) {
	var (
		all   []T
		token *ContinuationToken
	)
	for page := 0; page < maxPages; page++ {
		records, next, err := fetch(token)
		all = append(all, records...)
		if err != nil {
			return all, err
		}
		if next == nil || next.NextRowKey == "" {
			return all, nil
		}
		token = next
	}
	return all, fmt.Errorf("stopped after %d pages; narrow the filters", maxPages)
}

// Sum adds up amounts as myDATA formats them, and renders the result the same
// way. An unparseable amount is skipped rather than failing a whole report.
func Sum(amounts ...string) string {
	var total float64
	for _, a := range amounts {
		if a == "" {
			continue
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(a), 64); err == nil {
			total += f
		}
	}
	return strconv.FormatFloat(total, 'f', 2, 64)
}

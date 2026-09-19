package mydata

import (
	"encoding/xml"
	"testing"
)

// AADE publishes no XSD for RequestedBookInfo, so the parser has to survive
// either casing of the element names.
func TestRequestedBookInfoAcceptsEitherCasing(t *testing.T) {
	for name, payload := range map[string]string{
		"lowercase": `<RequestedBookInfo>
			<continuationToken><nextPartitionKey>p</nextPartitionKey><nextRowKey>r</nextRowKey></continuationToken>
			<bookInfo><counterVatNumber>123456789</counterVatNumber><issueDate>2026-09-01</issueDate>
				<invType>2.1</invType><selfpricing>false</selfpricing><netValue>100.00</netValue>
				<vatAmount>24.00</vatAmount><grossValue>124.00</grossValue><count>2</count>
				<minMark>400000001</minMark><maxMark>400000002</maxMark></bookInfo>
		</RequestedBookInfo>`,
		"uppercase": `<RequestedBookInfo>
			<ContinuationToken><NextPartitionKey>p</NextPartitionKey><NextRowKey>r</NextRowKey></ContinuationToken>
			<BookInfo><CounterVatNumber>123456789</CounterVatNumber><IssueDate>2026-09-01</IssueDate>
				<InvType>2.1</InvType><SelfPricing>false</SelfPricing><NetValue>100.00</NetValue>
				<VatAmount>24.00</VatAmount><GrossValue>124.00</GrossValue><Count>2</Count>
				<MinMark>400000001</MinMark><MaxMark>400000002</MaxMark></BookInfo>
		</RequestedBookInfo>`,
	} {
		t.Run(name, func(t *testing.T) {
			var doc RequestedBookInfo
			if err := xml.Unmarshal([]byte(payload), &doc); err != nil {
				t.Fatal(err)
			}
			if doc.Continuation == nil || doc.Continuation.NextRowKey != "r" {
				t.Errorf("continuation token not read: %+v", doc.Continuation)
			}
			if len(doc.Entries) != 1 {
				t.Fatalf("expected one entry, got %d", len(doc.Entries))
			}
			e := doc.Entries[0]
			if e.CounterVatNumber != "123456789" || e.InvType != "2.1" || e.NetValue != "100.00" ||
				e.GrossValue != "124.00" || e.Count != 2 || e.MaxMark != "400000002" || e.SelfPricing {
				t.Errorf("entry read wrong: %+v", e)
			}
		})
	}
}

func TestVatInfoKeepsUnknownBoxes(t *testing.T) {
	payload := `<RequestedVatInfo><VatInfo>
		<Mark>400000001</Mark><IsCancelled>false</IsCancelled><IssueDate>2026-09-01T00:00:00</IssueDate>
		<Vat301>100.00</Vat301><Vat361>24.00</Vat361>
	</VatInfo></RequestedVatInfo>`

	var doc RequestedVatInfo
	if err := xml.Unmarshal([]byte(payload), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Records) != 1 {
		t.Fatalf("expected one record, got %d", len(doc.Records))
	}
	r := doc.Records[0]
	if r.Mark != "400000001" || r.IsCancelled {
		t.Errorf("header read wrong: %+v", r)
	}
	if len(r.Amounts) != 2 || r.Amounts[0].Name != "Vat301" || r.Amounts[1].Value != "24.00" {
		t.Errorf("boxes read wrong: %+v", r.Amounts)
	}
}

func TestSumIgnoresJunk(t *testing.T) {
	if got := Sum("10.50", "", "4.50", "not a number"); got != "15.00" {
		t.Errorf("Sum = %q, want 15.00", got)
	}
}

func TestBookQueryValuesSkipsEmptyFilters(t *testing.T) {
	v := BookQuery{DateFrom: "01/09/2026", DateTo: "30/09/2026"}.Values()
	if v.Get("dateFrom") != "01/09/2026" || v.Get("dateTo") != "30/09/2026" {
		t.Errorf("dates missing: %v", v)
	}
	if _, ok := v["invType"]; ok {
		t.Errorf("empty filters should not be sent: %v", v)
	}
}

func TestInfoQueryOnlySendsGroupedPerDayWhenSet(t *testing.T) {
	q := InfoQuery{DateFrom: "01/09/2026", DateTo: "30/09/2026"}
	if _, ok := q.Values()["GroupedPerDay"]; ok {
		t.Error("GroupedPerDay should be omitted when false")
	}
	q.GroupedPerDay = true
	if q.Values().Get("GroupedPerDay") != "true" {
		t.Error("GroupedPerDay=true should be sent")
	}
}

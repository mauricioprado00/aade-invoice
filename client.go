package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	creds Credentials
	http  *http.Client
}

func NewClient(creds Credentials) *Client {
	return &Client{creds: creds, http: &http.Client{Timeout: 60 * time.Second}}
}

// ResponseDoc is the reply to every submission call; see
// docs/aade/xsd/response-v2.0.2.xsd.
type ResponseDoc struct {
	XMLName   xml.Name   `xml:"ResponseDoc"`
	Responses []Response `xml:"response"`
}

type Response struct {
	Index       int     `xml:"index"`
	InvoiceUid  string  `xml:"invoiceUid"`
	InvoiceMark int64   `xml:"invoiceMark"`
	QrURL       string  `xml:"qrUrl"`
	StatusCode  string  `xml:"statusCode"`
	Errors      []Error `xml:"errors>error"`
}

type Error struct {
	Message string `xml:"message"`
	Code    string `xml:"code"`
}

func (c *Client) SendInvoices(body []byte) (*ResponseDoc, error) {
	url := c.creds.BaseURL + "/SendInvoices"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("aade-user-id", c.creds.UserID)
	req.Header.Set("ocp-apim-subscription-key", c.creds.SubscriptionKey)
	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s:\n%s", url, resp.Status, payload)
	}

	var doc ResponseDoc
	if err := xml.Unmarshal(payload, &doc); err != nil {
		return nil, fmt.Errorf("could not parse response:\n%s", payload)
	}
	return &doc, nil
}

// RequestedDoc is the reply to RequestTransmittedDocs; only the fields this
// tool needs are mapped. See docs/aade/xsd/requestedInvoicesDoc-v2.0.2.xsd.
type RequestedDoc struct {
	XMLName  xml.Name         `xml:"RequestedDoc"`
	Invoices []TransmittedDoc `xml:"invoicesDoc>invoice"`
}

type TransmittedDoc struct {
	Uid                   string `xml:"uid"`
	Mark                  int64  `xml:"mark"`
	DownloadingInvoiceURL string `xml:"downloadingInvoiceUrl"`
}

// RequestTransmittedDocs returns the documents this entity transmitted with a
// MARK greater than the one given.
func (c *Client) RequestTransmittedDocs(afterMark int64) (*RequestedDoc, error) {
	url := fmt.Sprintf("%s/RequestTransmittedDocs?mark=%d", c.creds.BaseURL, afterMark)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("aade-user-id", c.creds.UserID)
	req.Header.Set("ocp-apim-subscription-key", c.creds.SubscriptionKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s:\n%s", url, resp.Status, payload)
	}

	var doc RequestedDoc
	if err := xml.Unmarshal(payload, &doc); err != nil {
		return nil, fmt.Errorf("could not parse response:\n%s", payload)
	}
	return &doc, nil
}

// downloadURLForMark reads the invoice back and returns AADE's own PDF link for
// it. The link is only present on invoices AADE renders itself; invoices merely
// transmitted by an ERP may carry none.
func (c *Client) downloadURLForMark(mark int64) (string, error) {
	doc, err := c.RequestTransmittedDocs(mark - 1)
	if err != nil {
		return "", err
	}
	for _, inv := range doc.Invoices {
		if inv.Mark != mark {
			continue
		}
		if inv.DownloadingInvoiceURL == "" {
			return "", fmt.Errorf("MARK %d carries no downloadingInvoiceUrl: AADE only renders a PDF for invoices issued through its own timologio application", mark)
		}
		return inv.DownloadingInvoiceURL, nil
	}
	return "", fmt.Errorf("MARK %d was not found in the transmitted documents", mark)
}

// DownloadInvoicePDF saves AADE's rendering of an invoice into dir and returns
// the path written.
func (c *Client) DownloadInvoicePDF(mark int64, dir string) (string, error) {
	url, err := c.downloadURLForMark(mark)
	if err != nil {
		return "", err
	}

	// The link 302s to the actual PDF; the default client follows that for us.
	resp, err := c.http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading the invoice returned %s", resp.Status)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/pdf") {
		return "", fmt.Errorf("expected a PDF, got %q", ct)
	}

	path := filepath.Join(dir, fmt.Sprintf("invoice-%d.pdf", mark))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return path, nil
}

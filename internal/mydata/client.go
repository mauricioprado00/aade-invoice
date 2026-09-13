package mydata

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// DownloadInvoicePDF saves AADE's rendering of an invoice into dir and returns
// the path written.
func (c *Client) DownloadInvoicePDF(mark int64, dir string) (string, error) {
	link, err := c.downloadURLForMark(mark)
	if err != nil {
		return "", err
	}

	// The link 302s to the actual PDF; the default client follows that for us.
	resp, err := c.http.Get(link)
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

// get performs an authenticated GET and unmarshals the XML reply into out.
func (c *Client) get(path string, query url.Values, out any) error {
	endpoint := c.creds.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("aade-user-id", c.creds.UserID)
	req.Header.Set("ocp-apim-subscription-key", c.creds.SubscriptionKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s:\n%s", endpoint, resp.Status, payload)
	}
	if err := xml.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("could not parse the response:\n%s", payload)
	}
	return nil
}

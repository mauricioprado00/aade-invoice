# aade-invoice

Go command line that registers an invoice in the AADE myDATA system.

```
go build -o bin/aade-invoice .
./bin/aade-invoice <amount>          # sandbox
./bin/aade-invoice --prod <amount>   # the real thing
```

The amount is the net value in euros. Everything else comes from
`invoice-template.json`, which is modelled on the invoices already issued under
this VAT number: a 2.3 service invoice, series A, VAT category 7 with exemption
category 6, income classification `E3_561_006` / `category1_3`.

Useful flags:

| Flag | Meaning |
|---|---|
| `--dry-run` | print the XML that would be sent and stop |
| `--prod` | production instead of sandbox |
| `--date 2026-09-13` | issue date (default: today) |
| `--aa 85` | invoice number, overriding the template counter |
| `--pdf` | after registering, download AADE's PDF of the invoice |
| `--pdf-mark 400000000000000` | download the PDF for an already registered MARK and exit |
| `--pdf-dir` | where to write PDFs (default: the current directory) |
| `--template`, `--env` | alternative file locations |

`nextAa` in the template is the local invoice counter. It is incremented only
after AADE accepts a submission, and is skipped entirely when `--aa` is given.
It can drift from what AADE holds if invoices are also issued elsewhere — pass
`--aa` when in doubt.

## Credentials

Copy `.env.example` to `.env` and fill it in. `.env` is gitignored.

Registration: sandbox at <https://mydata-dev-register.azurewebsites.net>,
production at <https://www1.aade.gr/saadeapps2/bookkeeper-web> (TAXISnet login →
«Φόρμα εγγραφής στο myDATA REST API»). The two are separate accounts with
separate keys.

## Documentation

`docs/NOTES.md` summarises the API; `docs/aade/` holds the official v2.0.2 spec,
XSDs and sample payloads, including a real production response in
`docs/aade/samples/`.

## The PDF

Submission itself returns no PDF, only MARK, uid and a QR URL. The PDF comes
from a second step: the invoice is read back with `RequestTransmittedDocs`, and
the `downloadingInvoiceUrl` it carries redirects to an `application/pdf` that
AADE renders. That is what `--pdf` and `--pdf-mark` do.

**That link only exists on invoices AADE renders itself** — the ones carrying
`<invoiceFormat>1</invoiceFormat>`, i.e. issued through its timologio
application. Invoices merely transmitted by an ERP, including everything this
tool sends, come back without it, and `--pdf` then reports that rather than
writing a file. Verified both ways: the sandbox invoice registered by this tool
has no link; production MARK 400000000000000, issued through timologio,
downloads a one-page PDF.

So getting a PDF for our own submissions means rendering it here from the same
data. Not done yet.

# aade-invoice

Go command line tools for the AADE myDATA system.

```
make
./bin/aade-invoice <amount>   # register an invoice
./bin/aade-list               # list invoices
./bin/aade-read <mark>        # show one invoice
```

All three behave the same way about environments and credentials: they read
`.env`, they talk to the **sandbox by default**, and `--prod` is what switches
them to production. `--env` points at a different credentials file.

## aade-invoice

The amount is the net value in euros. Everything else comes from an invoice
template. Copy `templates/invoice-template.json.sample` to a `.json` file in
`templates/` and fill in the issuer and counterpart details. When that directory
contains exactly one `.json` file it is selected automatically; otherwise,
provide the file explicitly with `--template <file>`.

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
| `--template <file>`, `--env <file>` | explicit template and credentials file locations |

`nextAa` is the ΑΑ (αύξων αριθμός) — the invoice number inside the series, the
`<aa>` element of the header. AADE does not allocate it; the issuer owns the
counter, so the template keeps the next one to use. It is incremented only after AADE accepts a
submission, and is skipped entirely when `--aa` is given. It can drift from what
AADE holds if invoices are also issued elsewhere — `./bin/aade-list --prod
--type 2.3 --series A` shows the real last number.

## aade-list

```
./bin/aade-list --from 2026-01-01 --to 2026-12-31
./bin/aade-list --prod --type 2.3 --series A
./bin/aade-list --prod --source received
```

| Flag | Meaning |
|---|---|
| `--from`, `--to` | issue date range, `YYYY-MM-DD` (translated to the dd/MM/yyyy myDATA wants) |
| `--type` | invoice type, e.g. `2.3` (appendix 8.1 of the spec) |
| `--counterpart` | VAT number of the other party |
| `--series` | invoice series, filtered locally |
| `--after-mark`, `--max-mark` | MARK range |
| `--source` | `transmitted` (default), `received`, or `all` |
| `--xml` | raw XML instead of the table |

`--source` picks which endpoint is read, and it is not the same distinction as
who issued the document: expense documents we register ourselves are
*transmitted* by us but *issued* by someone else. The DIR column says which,
comparing the issuer against our own VAT number.

Results are paged by myDATA through continuation tokens; the tool follows them
up to `--max-pages`.

## aade-read

```
./bin/aade-read 400001970914584
./bin/aade-read --prod 400000000000000 --json
./bin/aade-read --prod 400000000000000 --pdf .
```

Looks the MARK up in both directions — what we transmitted first, then what was
addressed to us — and prints the header, the lines with their classifications,
the totals and any QR/PDF links. `--json` and `--xml` give the raw document;
`--pdf <dir>` also downloads AADE's rendering, subject to the limitation below.

## Credentials

Copy `.env.example` to `.env` and fill it in. `.env` is gitignored.

Registration: sandbox at <https://mydata-dev-register.azurewebsites.net>,
production at <https://www1.aade.gr/saadeapps2/bookkeeper-web> (TAXISnet login →
«Φόρμα εγγραφής στο myDATA REST API»). The two are separate accounts with
separate keys.

## Layout

`cmd/` holds one directory per command, `internal/mydata` the API client,
invoice rendering and the template, `internal/cli` the flags and credential
handling all three commands share.

## Documentation

`docs/NOTES.md` summarises the API; `docs/aade/` holds the official v2.0.2 spec,
XSDs and sample payloads.

## The PDF

Submission itself returns no PDF, only MARK, uid and a QR URL. The PDF comes
from a second step: the invoice is read back with `RequestTransmittedDocs`, and
the `downloadingInvoiceUrl` it carries redirects to an `application/pdf` that
AADE renders. That is what `--pdf` and `--pdf-mark` do.

**That link only exists on invoices AADE renders itself** — the ones carrying
`<invoiceFormat>1</invoiceFormat>`, i.e. issued through its timologio
application. Invoices merely transmitted by an ERP, including everything this
tool sends, come back without it, and `--pdf` then reports that rather than
writing a file.

So getting a PDF for our own submissions means rendering it here from the same
data. Not done yet.

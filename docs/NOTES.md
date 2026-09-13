# myDATA (AADE) REST API — working notes

Everything downloaded lives in `docs/aade/`. Nothing here needs to be in context;
grep the extracted text instead.

| File | What it is |
|---|---|
| `aade/myDATA_API_Documentation_v2.0.2_erp.pdf` | Official ERP-user REST API spec, v2.0.2 (Sept 2026), 121 pp, Greek |
| `aade/myDATA_API_Documentation_v2.0.2_erp.txt` | Same, `pdftotext -layout` — **grep this** |
| `aade/myDATA_API_Documentation_DeliveryNote_v2.0.2.pdf` | Digital delivery notes (not needed for plain invoicing) |
| `aade/xsd/` | Official XSDs v2.0.2 (`InvoicesDoc-v2.0.2.xsd`, `SimpleTypes-v2.0.2.xsd`, `incomeClassification-*.xsd`, `response-*.xsd`, …) |
| `aade/syndiasmoi_xaraktirismwn_v2.0.2.xlsx` | Allowed invoiceType × income/expense classification combinations |
| `aade/samples/` | Official example request XMLs (2.1 service invoice, 1.1 goods sale, 5.1 credit note) |

Handy line numbers in the extracted text: registration §4.1.1 (l.237), headers
§4.1.2 (l.340), SendInvoices §4.2.1 (l.360), CancelInvoice (l.557),
RequestDocs (l.595), invoice schema ch.5 (l.~960), error codes ch.7 (l.~3400),
appendix code tables ch.8 (invoice types, VAT categories, payment methods,
classification codes) from l.~4300.

## Endpoints

| | Production | Dev / sandbox |
|---|---|---|
| Base | `https://mydatapi.aade.gr/myDATA/` | `https://mydataapidev.aade.gr/` |
| Send | `POST {base}SendInvoices` | `POST https://mydataapidev.aade.gr/SendInvoices` |
| Cancel | `GET {base}CancelInvoice?mark={mark}` | same path on dev host |
| Read own | `GET {base}RequestTransmittedDocs?mark={mark}` | |

Note the production base has the extra `/myDATA/` path segment; dev does not.

## Auth

Two headers on every call — no OAuth, no token refresh:

```
aade-user-id: <username>
ocp-apim-subscription-key: <subscription key>
```

The VAT number (ΑΦΜ) is bound to the account at registration, so it is not sent
per call (it is still written into `<issuer><vatNumber>`).

## Getting credentials

- **Production:** log in at <https://www1.aade.gr/saadeapps2/bookkeeper-web>
  with TAXISnet codes → «Φόρμα εγγραφής στο myDATA REST API» → «Νέα εγγραφή
  χρήστη» → pick a username (Latin letters + digits only) → «Προσθήκη». The key
  is the «Κωδικός API» column in the resulting list; it can be regenerated there.
- **Dev/sandbox:** separate registration at
  <https://mydata-dev-register.azurewebsites.net> — keys are not shared with
  production.

## Request shape

POST body is XML: `<InvoicesDoc>` containing one or more `<invoice>` of type
`AadeBookInvoiceType`. Response is XML (`response-v2.0.2.xsd`) with, per
invoice, either a success carrying **MARK** (the unique AADE registration
number), `uid`, `qrUrl`, or one/more business error codes.

Minimal service invoice (type 2.1), from `samples/sample_2.1_TPY_taxes_per_invoice.xml`:
issuer → counterpart → invoiceHeader (series, aa, issueDate, invoiceType,
currency) → paymentMethods → invoiceDetails (per line: netValue, vatCategory,
vatAmount, incomeClassification) → taxesTotals (e.g. withholding) →
invoiceSummary (totals + aggregated classification). Summary totals must equal
the sum of the lines or the submission is rejected.

Re-submitting an invoice with the same identifiers supersedes the previous one
(the old becomes cancelled).

## Open questions for the CLI

- Which `invoiceType` we issue (2.1 ΤΠΥ is the usual freelancer case).
- `incomeClassification` type/category pair (e.g. `E3_561_001` /
  `category1_3`) — must match the xlsx combination table.
- Whether withholding tax (παρακράτηση 20%) applies.
- Series (`series`) and numbering (`aa`) — who owns the counter.
- PDF: myDATA itself returns no PDF, only MARK + QR URL. Printing a human
  invoice means either rendering it ourselves from the same data, or using the
  AADE «timologio» app. Needs a decision.

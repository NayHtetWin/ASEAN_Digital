# `config.yaml` Source & Parser Specification

All public funding and tender sources are declared in `config.yaml` at the repository root.

## Source Entry Fields

| Field | Required | Allowed Values / Type | Notes |
| :--- | :--- | :--- | :--- |
| `name` | Yes | `string` | Human-readable source name written to `Opportunity.source`. |
| `url` | Yes | `string` (HTTP/HTTPS URI) | Feed or listing page URL. |
| `frequency` | Yes | `"daily"` \| `"weekly"` | Controls GitHub Actions / CLI schedule filtering. |
| `type` | Yes | `"rss"` \| `"html"` \| `"chromedp"` | Parser strategy (`gofeed`, `goquery`, or headless Chrome). |
| `item_selector` | Conditional | CSS selector `string` | Required for `"html"` and `"chromedp"` sources to select each listing container. |
| `link_selector` | Conditional | CSS selector `string` | Optional sub-selector inside `item_selector` to extract the canonical `href`. |
| `force_region` | No | `string` (e.g. `"ASEAN"`) | Forces regional relevance tag when the portal omits explicit ASEAN keywords. |

## Example

```yaml
sources:
  - name: "ASEAN Foundation"
    url: "https://www.aseanfoundation.org/grants"
    frequency: "daily"
    type: "html"
    item_selector: ".grant-card"
    link_selector: "a.read-more"
    force_region: "ASEAN"

  - name: "DFAT AusConnect"
    url: "https://ausconnect.dfat.gov.au/"
    frequency: "weekly"
    type: "chromedp"
    item_selector: ".opportunity-row"
    link_selector: "a.opportunity-title"
    force_region: "ASEAN"
```

## Go Date Parsing Layouts

Use a multi-layout slice with `time.Parse` after regex extraction:
- `"2006-01-02"`
- `"02 Jan 2006"` / `"2 Jan 2006"`
- `"02 January 2006"` / `"2 January 2006"`
- `"January 2, 2006"` / `"Jan 2, 2006"`
- `"02/01/2006"`

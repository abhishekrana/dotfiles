# Edge cases

## Lists

3. Ordered lists may start anywhere
4. And keep counting
   - A bullet nested in a number
     1. A number nested in a bullet
5. An item with a code block inside:

   ```rust
   fn main() {
       println!("nested");
   }
   ```

- A loose list

- keeps a blank row between its items

## Breaks and inline

A hard break follows this line  
and this one continues after a backslash\
then the paragraph ends. ~~Struck~~ text, <https://example.com/autolink>, a [titled link](https://example.com "Example"),
and math left as written: $E = mc^2$. Escaped \*asterisks\* and an inline <kbd>html</kbd> tag stay readable.

An inline image ![alt text](diagram.png) sits in a sentence.

![A block image](figure.png)

Averyveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylongword
breaks by grapheme, and 日本語のテキスト and emoji 🎉 take two cells each.

## Wide table

| Component | Purpose | Notes | Owner | Status |
| --- | --- | --- | --- | --- |
| Reverse proxy that terminates TLS for every service | Routes requests by host name to the right container | Certificates renew nightly through the ACME client | infra | live |
| Object storage gateway | Exposes the local disk as an S3-compatible endpoint | Used only by the backup jobs | infra | live |

| Left | Center | Right |
| :--- | :----: | ----: |
| a | b | c |
| longer | mid | 1 |

## Every callout

> [!NOTE]
> Information the reader should notice.

> [!TIP]
> Something that helps.

> [!IMPORTANT]
> Something the reader must know.

> [!WARNING]
> Something that needs care.

> [!CAUTION]
> Something with consequences.

> A quote holding a list:
>
> - one
> - two
>
> > and a nested quote.

### Level three

#### Level four

##### Level five

###### Level six

<div class="note">
A raw HTML block is shown as source.
</div>

```
```

Footnotes can have two paragraphs[^long].

[^long]: The first paragraph of the note.

    The second paragraph, indented under it.

package email

import (
	"fmt"
	"io"
	"mime"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/emersion/go-message/mail"
	"github.com/k3a/html2text"
	"github.com/muesli/reflow/wrap"
	"golang.org/x/net/html/charset"
)

// ExtractedBody holds the result of MIME parsing
type ExtractedBody struct {
	Text   string
	HTML   string
	Params map[string]string // Content-Type params like format=flowed
}

// ExtractBody parses a MIME message and extracts the best content
func ExtractBody(mr *mail.Reader) (*ExtractedBody, error) {
	eb := &ExtractedBody{
		Params: make(map[string]string),
	}

	textBody, htmlBody, params, err := walkMIMEPartsWithParams(mr)
	if err != nil {
		return nil, err
	}

	eb.Text = textBody
	eb.HTML = htmlBody
	eb.Params = params

	// Strict preference: Only use HTML-to-text if we have NO text at all
	// or if the text is extremely short/useless (like "View in HTML")
	if strings.TrimSpace(eb.Text) == "" && eb.HTML != "" {
		eb.Text = html2text.HTML2Text(eb.HTML)
	}

	return eb, nil
}

// walkMIMEPartsWithParams recursively walks through MIME parts and returns content + params
func walkMIMEPartsWithParams(mr *mail.Reader) (string, string, map[string]string, error) {
	var textBody, htmlBody strings.Builder
	var textParams map[string]string

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		contentType := p.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			mediaType = contentType
		}

		if strings.HasPrefix(mediaType, "multipart/") {
			subMr, err := mail.CreateReader(p.Body)
			if err == nil {
				subText, subHtml, subParams, _ := walkMIMEPartsWithParams(subMr)

				if strings.HasPrefix(mediaType, "multipart/alternative") {
					// For alternative, we want the best representation.
					// If we found text in the alternative, use it and its params.
					if subText != "" {
						textParams = subParams
					}
				}

				textBody.WriteString(subText)
				htmlBody.WriteString(subHtml)
			}
		} else {
			// Read body content with charset decoding
			var reader io.Reader = p.Body
			if cs, ok := params["charset"]; ok {
				r, err := charset.NewReaderLabel(cs, p.Body)
				if err == nil {
					reader = r
				}
			}

			b, err := io.ReadAll(reader)
			if err != nil {
				continue
			}

			if mediaType == "text/plain" {
				textBody.Write(b)
				// Capture params for the text part (e.g. format=flowed)
				if textParams == nil {
					textParams = params
				}
			} else if mediaType == "text/html" {
				htmlBody.Write(b)
			}
		}
	}

	if textParams == nil {
		textParams = make(map[string]string)
	}

	return textBody.String(), htmlBody.String(), textParams, nil
}

// RenderBody renders the email body with proper wrapping and styling
// Returns the rendered body and a list of extracted links
func RenderBody(text string, width int, params map[string]string) (string, []string) {
	if width <= 0 {
		width = 80 // Fallback
	}

	// 1. Handle format=flowed
	if params["format"] == "flowed" {
		// Reflow the text (join lines that end with space)
		text = reflowText(text)
	}

	// 2. Extract and numbering URLs
	var links []string
	text, links = extractAndNumberURLs(text)

	// 3. Wrap text
	// We use muesli/reflow/wrap or wordwrap
	wrapped := wrap.String(text, width)

	// 4. Style quoted text
	styled := styleQuotedText(wrapped)

	// 5. Append references if links exist
	if len(links) > 0 {
		styled += "\n\n" + renderReferences(links, width)
	}

	return styled, links
}

// extractAndNumberURLs finds URLs, replaces them with [N], and returns the modified text and link list
func extractAndNumberURLs(text string) (string, []string) {
	// Simple regex for URLs (http/https)
	// Note: This is a basic regex, might need refinement for complex URLs
	re := regexp.MustCompile(`https?://[^\s\)\}\]<>,]+`)

	var links []string
	linkMap := make(map[string]int) // Map URL to index (1-based)

	// ReplaceAllStringFunc doesn't support state easily, so we use FindAllStringIndex
	// But simpler to just iterate matches

	// We want to avoid replacing inside existing brackets if possible, but for plain text it's usually fine.

	replaced := re.ReplaceAllStringFunc(text, func(url string) string {
		// Clean trailing punctuation often picked up by regex
		url = strings.TrimRight(url, ".,;:?!")

		idx, exists := linkMap[url]
		if !exists {
			idx = len(links) + 1
			links = append(links, url)
			linkMap[url] = idx
		}

		// OSC-8 clickable link: \x1b]8;;URL\x1b\LINK_TEXT\x1b]8;;\x1b\
		// We display [N] as the link text
		return fmt.Sprintf("\x1b]8;;%s\x1b\\[%d]\x1b]8;;\x1b\\", url, idx)
	})

	return replaced, links
}

func renderReferences(links []string, width int) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Links:"))
	sb.WriteString("\n")

	for i, link := range links {
		// Format: [1] https://example.com
		// We can truncate long URLs if needed, but usually references are meant to be full.
		// Let's just print them.
		line := fmt.Sprintf("[%d] %s", i+1, link)

		// Wrap the line if it's too long?
		// URLs shouldn't be wrapped ideally, but for TUI we might have to.
		// For now, let's just let them be long or truncate if really needed.
		// We'll just print them.
		sb.WriteString(line)
		if i < len(links)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// reflowText handles basic format=flowed decoding (joining lines ending in space)
func reflowText(text string) string {
	var sb strings.Builder
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// If line ends with space and is not a signature separator "-- "
		if strings.HasSuffix(line, " ") && line != "-- " {
			// Remove the trailing space and join with next line
			sb.WriteString(strings.TrimSuffix(line, " "))
			sb.WriteString(" ") // Add a single space separator
		} else {
			sb.WriteString(line)
			if i < len(lines)-1 {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// styleQuotedText adds ANSI colors to quoted lines based on depth
func styleQuotedText(text string) string {
	var sb strings.Builder
	lines := strings.Split(text, "\n")

	// Colors for nested quotes (Aerc-like cycling)
	// 1: Green, 2: Yellow, 3: Red, 4: Cyan, 5: Magenta
	quoteColors := []string{
		"\033[32m", // Green
		"\033[33m", // Yellow
		"\033[31m", // Red
		"\033[36m", // Cyan
		"\033[35m", // Magenta
	}

	inSignature := false
	sigColor := "\033[38;5;240m" // Dark grey for signature
	reset := "\033[0m"

	for i, line := range lines {
		// Check for signature separator
		if line == "-- " || line == "--" {
			inSignature = true
		}

		if inSignature {
			sb.WriteString(sigColor + line + reset)
		} else {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, ">") {
				// Count depth
				depth := 0
				for _, r := range trimmed {
					if r == '>' {
						depth++
					} else {
						break
					}
				}

				// Cycle colors (0-indexed for array)
				colorIdx := (depth - 1) % len(quoteColors)
				color := quoteColors[colorIdx]

				sb.WriteString(color + line + reset)
			} else if strings.HasPrefix(line, "On ") && strings.HasSuffix(trimmed, "wrote:") {
				// Attribution line
				sb.WriteString("\033[38;5;245m" + line + reset)
			} else {
				sb.WriteString(line)
			}
		}

		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// RenderHeaders renders the email headers nicely (Aerc style)
func RenderHeaders(from, to, subject, date string, width int) string {
	var sb strings.Builder

	// Aerc style:
	// From: Name <email>
	// To:   Name <email>
	// Date: ...
	// Subject: ...
	// (Divider)

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")) // Blue
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))            // White

	// Helper to render a row
	renderRow := func(label, value string) {
		// Ensure alignment
		padding := 8 - len(label)
		if padding < 0 {
			padding = 0
		}

		sb.WriteString(labelStyle.Render(label + ":"))
		sb.WriteString(strings.Repeat(" ", padding))
		sb.WriteString(valueStyle.Render(value))
		sb.WriteString("\n")
	}

	renderRow("From", from)
	renderRow("To", to)
	renderRow("Date", date)
	renderRow("Subject", subject)

	// Simple divider
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("─", width)))

	return sb.String()
}

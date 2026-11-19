// Package testfixture builds synthetic LinkedIn exports for tests. All names
// and URLs are fictional.
package testfixture

import (
	"strings"
	"testing/fstest"
)

const msgHeader = `"CONVERSATION ID","CONVERSATION TITLE","FROM","SENDER PROFILE URL","TO","RECIPIENT PROFILE URLS","DATE","SUBJECT","CONTENT","FOLDER","ATTACHMENTS"`

// Messages in the first export.
var baseMessages = []string{
	`"conv-1","","Equity Buyer","https://www.linkedin.com/in/equity-buyer","Pat Owner","https://www.linkedin.com/in/pat-owner","2026-03-01 10:00:00 UTC","Your shares","<p>We are buying shares in Example Corp from employees. Interested in a secondary sale?</p>","INBOX",""`,
	`"conv-1","","Pat Owner","https://www.linkedin.com/in/pat-owner","Equity Buyer","https://www.linkedin.com/in/equity-buyer","2026-03-01 11:00:00 UTC","","Not right now, thanks.","INBOX",""`,
	`"conv-2","","Liquidity Fund","","Pat Owner","https://www.linkedin.com/in/pat-owner","2026-04-01 09:00:00 UTC","","Get cash for your stock options at Example Corp without selling.","ARCHIVE",""`,
	`"conv-3","","Recruiter Person","https://www.linkedin.com/in/recruiter-person","Pat Owner","https://www.linkedin.com/in/pat-owner","2025-05-01 09:00:00 UTC","Role","We are hiring a VP of Product at a pre-IPO company.","INBOX",""`,
}

const connections = `Notes:
"Email addresses may be missing."

First Name,Last Name,URL,Email Address,Company,Position,Connected On
Equity,Buyer,https://www.linkedin.com/in/equity-buyer,,Buyer Capital,Partner,01 Feb 2026
,,,,,,10 Jun 2024
`

const invitations = `From,To,Sent At,Message,Direction,inviterProfileUrl,inviteeProfileUrl
Equity Buyer,Pat Owner,"1/30/26, 5:04 PM",Let's connect,INCOMING,https://www.linkedin.com/in/equity-buyer,https://www.linkedin.com/in/pat-owner
`

// Basic returns the first synthetic Basic export.
func Basic() fstest.MapFS {
	return fstest.MapFS{
		"messages.csv":    {Data: []byte(msgHeader + "\n" + strings.Join(baseMessages, "\n") + "\n")},
		"Connections.csv": {Data: []byte(connections)},
		"Invitations.csv": {Data: []byte(invitations)},
		"Skills.csv":      {Data: []byte("Name\nProduct Management\nProduct Management\n")},
	}
}

// Newer returns a later Complete export: one message moved to ARCHIVE, one
// new message, and a saved items file.
func Newer() fstest.MapFS {
	msgs := append([]string(nil), baseMessages...)
	msgs[1] = strings.Replace(msgs[1], `"INBOX"`, `"ARCHIVE"`, 1)
	msgs = append(msgs, `"conv-4","","Tender Desk","https://www.linkedin.com/in/tender-desk","Pat Owner","https://www.linkedin.com/in/pat-owner","2026-10-01 09:00:00 UTC","","A tender offer for Example Corp shares is open.","INBOX",""`)
	fs := Basic()
	fs["messages.csv"] = &fstest.MapFile{Data: []byte(msgHeader + "\n" + strings.Join(msgs, "\n") + "\n")}
	fs["Saved_Items.csv"] = &fstest.MapFile{Data: []byte("Saved Item,Saved At\nhttps://www.linkedin.com/feed/update/urn:li:activity:1,2026-09-01 10:00:00 UTC\n")}
	fs["Shares.csv"] = &fstest.MapFile{Data: []byte("Date,ShareLink,ShareCommentary\n2026-09-02 10:00:00,https://www.linkedin.com/feed/update/urn:li:activity:2,Hello world\n")}
	return fs
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/tealeg/xlsx"
)

type TrelloName struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TrelloNameList []*TrelloName

type RuntimeConfig struct {
	ShowUsage          bool
	Command            string
	SearchResultFields string
	ColSep             string
	RowSep             string
	QuoteChar          string
	CardLimit          int
	NumberOutput       bool
	Format             string
	BoardName          string
	ListName           string
}

type TrelloClient struct {
	TrelloKey    string
	TrelloToken  string
	HTTPClient   *http.Client
	TrelloBoards TrelloNameList
	TrelloLists  map[string]TrelloNameList
	config       *RuntimeConfig
}

type TrelloBadges struct {
	Attachments        int    `json:"attachments"`
	CheckItems         int    `json:"checkItems"`
	CheckItemsChecked  int    `json:"checkItemsChecked"`
	Comments           int    `json:"comments"`
	Description        bool   `json:"description"`
	Due                string `json:"due"`
	Fogbugz            string `json:"fogbugz"`
	Subscribed         bool   `json:"subscribed"`
	ViewingMemberVoted bool   `json:"viewingMemberVoted"`
	Votes              int    `json:"votes"`
}

type TrelloLabel struct {
	Color   string `json:"color"`
	ID      string `json:"id"`
	IDBoard string `json:"idBoard"`
	Name    string `json:"name"`
	Uses    int    `json:"uses"`
}

type TrelloCardSearchResult struct {
	ID              string        `json:"id"`
	Badges          *TrelloBadges `json:"badges"`
	CheckItemStates []*struct {
		IDCheckItem string `json:"idCheckItem"`
		State       string `json:"state"`
	} `json:"checkItemStates"`
	Closed                bool           `json:"closed"`
	DateLastActivity      string         `json:"dateLastActivity"`
	Desc                  string         `json:"desc"`
	DescData              interface{}    `json:"descData"`
	Due                   string         `json:"due"`
	Email                 string         `json:"email"`
	IDAttachmentCover     string         `json:"idAttachmentCover"`
	IDBoard               string         `json:"idBoard"`
	IDChecklists          []string       `json:"idChecklists"`
	IDLabels              []string       `json:"idLabels"`
	IDList                string         `json:"idList"`
	IDMembers             []string       `json:"idMembers"`
	IDMembersVoted        []string       `json:"idMembersVoted"`
	IDShort               int            `json:"idShort"`
	Labels                []*TrelloLabel `json:"labels"`
	ManualCoverAttachment bool           `json:"manualCoverAttachment"`
	Name                  string         `json:"name"`
	Pos                   float64        `json:"pos"`
	ShortLink             string         `json:"shortLink"`
	ShortURL              string         `json:"shortUrl"`
	Subscribed            bool           `json:"subscribed"`
	URL                   string         `json:"url	"`
}

type TrelloSearchResult struct {
	Cards   []*TrelloCardSearchResult `json:"cards"`
	Options interface{}
}

type TrelloCardComment struct {
	Type            string `json:"type"`
	Date            string `json:"date"`
	IDComment       string `json:"id"`
	IDMemberCreator string `json:"idMemberCreator"`
	Data            struct {
		Board struct {
			IDBoard   string `json:"id"`
			BoardName string `json:"name"`
			ShortLink string `json:"shortLink"`
		} `json:"board"`
		Card struct {
			IDCard    string `json:"id"`
			CardName  string `json:"name"`
			IDShort   int64  `json:"idShort"`
			ShortLink string `json:"shortLink"`
		} `json:"card"`
		List struct {
			IDList   string `json:"id"`
			ListName string `json:"name"`
		} `json:"list"`
		Text string `json:"text"`
	} `json:"data"`
	MemberCreator struct {
		AvatarHash string `json:"avatarHash"`
		FullName   string `json:"fullName"`
		IDMember   string `json:"id"`
		Initials   string `json:"initials"`
		UserName   string `json:"username"`
	} `json:"memberCreator"`
}

type TrelloChecklist struct {
	IDChecklist string  `json:"id"`
	IDBoard     string  `json:"idBoard"`
	IDCard      string  `json:"idCard"`
	Name        string  `json:"name"`
	Position    float64 `json:"pos"`
	CheckItems  []struct {
		ID       string      `json:"id"`
		Name     string      `json:"name"`
		NameData interface{} `json:"nameData"`
		Pos      float64     `json:"pos"`
		State    string      `json:"state"`
	} `json:"checkItems"`
}

type TrelloMember struct {
	AvatarHash      string      `json:"avatarHash"`
	Bio             string      `json:"bio"`
	BioData         interface{} `json:"bioData"`
	Confirmed       bool        `json:"confirmed"`
	FullName        string      `json:"fullName"`
	IDMember        string      `json:"id"`
	IDPremOrgsAdmin interface{} `json:"idPremOrgsAdmin"`
	Initials        string      `json:"initials"`
	MemberType      string      `json:"memberType"`
	Products        []int       `json:"products"`
	Status          string      `json:"status"`
	URL             string      `json:"url"`
	UserName        string      `json:"username"`
}

type TrelloList struct {
	IDList   string  `json:"id"`
	ListName string  `json:"name"`
	Closed   bool    `json:"closed"`
	IDBoard  string  `json:"idBoard"`
	Position float64 `json:"pos"`
}

func newTrelloClient() *TrelloClient {
	client := &TrelloClient{}
	client.HTTPClient = &http.Client{}
	client.config = &RuntimeConfig{}
	client.TrelloLists = make(map[string]TrelloNameList)

	key := os.ExpandEnv("$TRELLO_KEY")
	if key == "" {
		fmt.Println("TRELLO_KEY environment variable not set, exiting.")
		os.Exit(1)
	} else {
		client.TrelloKey = key
	}
	tok := os.ExpandEnv("$TRELLO_TOKEN")
	if key == "" {
		fmt.Println("TRELLO_TOKEN environment variable not set, exiting.")
		os.Exit(1)
	} else {
		client.TrelloToken = tok
	}

	return client
}

func IDFromName(name string, nameList TrelloNameList) string {
	for _, v := range nameList {
		if strings.ToUpper(v.Name) == strings.ToUpper(name) {
			return v.ID
		}
	}
	return ""
}

func NameFromID(theID string, nameList TrelloNameList) string {
	for _, v := range nameList {
		if v.ID == theID {
			return v.Name
		}
	}
	return ""
}

func (client *TrelloClient) prepareQuery(path string, query map[string]string) *url.URL {
	theURL := &url.URL{}
	theURL.Scheme = "https"
	theURL.Host = "api.trello.com"
	theURL.Path = path
	if query != nil {
		q := theURL.Query()
		for key, value := range query {
			q.Set(key, value)
		}
		q.Set("key", client.TrelloKey)
		q.Set("token", client.TrelloToken)
		theURL.RawQuery = q.Encode()
	}
	return theURL
}

func processResponse(resp *http.Response, err error, result interface{}) error {

	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("HTTP Status " + resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &result)

	return err
}

func (client *TrelloClient) TrelloNamesFromURL(theURL string) (TrelloNameList, error) {
	result := TrelloNameList{}
	resp, err := client.HTTPClient.Get(theURL)
	err = processResponse(resp, err, &result)
	return result, err
}

func (client *TrelloClient) ListNames(boardID string) (TrelloNameList, error) {
	q := map[string]string{
		"fields": "name",
	}
	theURL := client.prepareQuery("/1/boards/"+strings.TrimSpace(boardID)+"/lists", q)
	return client.TrelloNamesFromURL(theURL.String())
}

func (client *TrelloClient) BoardNames(memberID string) (TrelloNameList, error) {
	q := map[string]string{
		"fields": "name",
	}
	theURL := client.prepareQuery("/1/members/"+strings.TrimSpace(memberID)+"/boards", q)
	return client.TrelloNamesFromURL(theURL.String())
}

func (client *TrelloClient) LabelNames(boardID string) (TrelloNameList, error) {
	q := map[string]string{
		"fields": "name",
	}
	theURL := client.prepareQuery("/1/boards/"+strings.TrimSpace(boardID)+"/labels", q)
	return client.TrelloNamesFromURL(theURL.String())
}

func (client *TrelloClient) CreateList(boardID, listName, position string) (*TrelloList, error) {
	theURL := client.prepareQuery("/1/board/"+strings.TrimSpace(boardID)+"/lists", nil)
	postData := fmt.Sprintf(`{"key": "%s", "token": "%s", "name": "%s", "pos": "%s"}`, client.TrelloKey, client.TrelloToken, listName, position)
	req, err := http.NewRequest("POST", theURL.String(), bytes.NewReader([]byte(postData)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.HTTPClient.Do(req)
	result := &TrelloList{}
	err = processResponse(resp, err, &result)
	return result, err
}

func (client *TrelloClient) BoardMembers(boardID string) ([]*TrelloMember, error) {
	q := map[string]string{
		"fields": "all",
	}
	theURL := client.prepareQuery("/1/board/"+strings.TrimSpace(boardID)+"/members", q)
	result := []*TrelloMember{}
	resp, err := client.HTTPClient.Get(theURL.String())
	err = processResponse(resp, err, &result)
	return result, err
}

func (client *TrelloClient) CardComments(cardID string) ([]*TrelloCardComment, error) {
	q := map[string]string{
		"filter": "commentCard",
	}
	theURL := client.prepareQuery("/1/card/"+strings.TrimSpace(cardID)+"/actions", q)
	result := []*TrelloCardComment{}
	resp, err := client.HTTPClient.Get(theURL.String())
	err = processResponse(resp, err, &result)
	return result, err
}

func (client *TrelloClient) CardChecklists(cardID string) ([]*TrelloChecklist, error) {
	q := map[string]string{
		"fields": "name,idBoard,idCard",
	}
	theURL := client.prepareQuery("/1/card/"+strings.TrimSpace(cardID)+"/checklists", q)
	result := []*TrelloChecklist{}
	resp, err := client.HTTPClient.Get(theURL.String())
	err = processResponse(resp, err, &result)
	return result, err
}

func (client *TrelloClient) SearchCards(query string, limit int) ([]*TrelloCardSearchResult, error) {
	q := map[string]string{
		"modelTypes":  "cards",
		"card_fields": "all",
		"cards_limit": strconv.Itoa(limit),
		"query":       query,
	}
	theURL := client.prepareQuery("/1/search", q)
	fmt.Println(theURL.String())
	result := TrelloSearchResult{}
	resp, err := client.HTTPClient.Get(theURL.String())
	err = processResponse(resp, err, &result)
	return result.Cards, err
}

func (client *TrelloClient) loadBoardInfo() error {
	var err error

	trelloUser := os.ExpandEnv("$TRELLO_USER")
	if trelloUser == "" {
		trelloUser = "me" // default to "me" then
	}

	client.TrelloBoards, err = client.BoardNames(trelloUser)
	if err != nil {
		return err
	}
	for _, v := range client.TrelloBoards {
		boardName := strings.ToLower(v.Name)
		client.TrelloLists[boardName] = TrelloNameList{}
		tmp, err := client.ListNames(v.ID)
		if err != nil {
			return err
		}
		for _, v := range tmp {
			client.TrelloLists[boardName] = append(client.TrelloLists[boardName], v)
		}
	}
	return nil
}

func (client *TrelloClient) buildOutputLine(card *TrelloCardSearchResult) []string {
	list := strings.Split(client.config.SearchResultFields, ",")
	result := []string{}
	for _, v := range list {
		var item string
		v = strings.TrimSpace(strings.ToLower(v))
		switch v {
		case "id":
			item = card.ID
		case "attachmentcount":
			item = strconv.Itoa(card.Badges.Attachments)
		case "checked":
			item = strconv.Itoa(card.Badges.CheckItemsChecked) + "/" + strconv.Itoa(card.Badges.CheckItems)
		case "commentcount":
			item = strconv.Itoa(card.Badges.Comments)
		case "hasdesc":
			item = strconv.FormatBool(card.Badges.Description)
		case "closed":
			item = strconv.FormatBool(card.Closed)
		case "datelastactivity":
			item = card.DateLastActivity
		case "desc":
			item = card.Desc
		case "due":
			item = card.Due
		case "email":
			item = card.Email
		case "idattachmentcover":
			item = card.IDAttachmentCover
		case "idboard":
			item = card.IDBoard
		case "labels":
			labels := []string{}
			for _, v := range card.Labels {
				s := v.Name
				if s == "" {
					s = strings.ToUpper(v.Color)
				}
				labels = append(labels, "["+s+"]")
			}
			item = strings.Join(labels, " ")
		case "labelcolors":
			labels := []string{}
			for _, v := range card.Labels {
				s := v.Color
				s = strings.ToUpper(v.Color)
				labels = append(labels, "["+s+"]")
			}
			item = strings.Join(labels, " ")
		case "idlist":
			item = card.IDList
		case "listname":
			s := NameFromID(card.IDBoard, client.TrelloBoards)
			if s != "" {
				s = strings.ToLower(s)
				s = NameFromID(card.IDList, client.TrelloLists[s])
			}
			item = s
		case "boardname":
			item = NameFromID(card.IDBoard, client.TrelloBoards)
		case "idshort":
			item = strconv.Itoa(card.IDShort)
		case "name":
			item = card.Name
		case "pos":
			item = strconv.FormatFloat(card.Pos, 'g', 2, 64)
		case "shortlink":
			item = card.ShortLink
		case "shorturl":
			item = card.ShortURL
		case "subscribed":
			item = strconv.FormatBool(card.Subscribed)
		case "comments":
			item = ""
			if card.Badges.Comments > 0 {
				comments, err := client.CardComments(card.ID)
				if err == nil {
					for _, comment := range comments {
						item += "@" + comment.MemberCreator.UserName + " on " + comment.Date + ": " + strings.Replace(comment.Data.Text, "\n", "\\n", -1) + "\n"
					}
				} else {
					item = "[Could not read comments for card] " + err.Error()
				}
			}
		case "url":
			item = card.URL
		}
		if client.config.QuoteChar != "" {
			item = client.config.QuoteChar + item + client.config.QuoteChar
		}
		result = append(result, item)
	}
	return result
}

func (client *TrelloClient) formatterText(cards []*TrelloCardSearchResult) error {
	var err error
	fmt.Printf("Found %d cards\n", len(cards))
	fmt.Println()
	header := strings.Split(client.config.SearchResultFields, ",")
	for i, card := range cards {
		if client.config.NumberOutput {
			fmt.Printf("%4d ", i)
		}
		cols := client.buildOutputLine(card)
		for i := range header {
			fmt.Printf("%-25s: ", strings.Title(header[i]))
			if strings.ToLower(header[i]) == "comments" { // do a break before comments
				fmt.Println()
			}
			fmt.Println(cols[i])
		}

		if card.Badges.CheckItems > 0 {
			chklists, err := client.CardChecklists(card.ID)
			if err != nil {
				fmt.Println("[Could not read checklist items for card] ", err.Error())
			} else {
				fmt.Println("Checklists")

				for _, chklist := range chklists {
					fmt.Println(chklist.Name)
					for i, v := range chklist.CheckItems {
						s := fmt.Sprintf("%2d: %s ", i+1, v.Name)
						if v.State == "complete" {
							s += " ✅ (done)"
						}
						fmt.Println(s)
					}
				}
				fmt.Println()
			}
		}

		fmt.Println("--------")
	}
	return err
}

func (client *TrelloClient) formatterCsv(cards []*TrelloCardSearchResult) error {
	var err error
	header := strings.Join(strings.Split(client.config.SearchResultFields, ","), client.config.ColSep)
	fmt.Println(header)
	for _, card := range cards {
		card.Desc = strings.Replace(card.Desc, "\n", "\\n", -1)
		colbuf := client.buildOutputLine(card)
		fmt.Print(strings.Join(colbuf, client.config.ColSep))
		fmt.Print(client.config.RowSep)
	}
	return err
}

func (client *TrelloClient) formatterJSON(cards []*TrelloCardSearchResult) error {
	var err error
	doc := []byte{}
	doc, err = json.Marshal(cards)
	if err == nil {
		fmt.Print(string(doc))
		fmt.Print(client.config.RowSep)
	}
	return err
}

func (client *TrelloClient) formatterExcel(cards []*TrelloCardSearchResult) error {
	var err error

	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell

	client.config.QuoteChar = "" // we do not need quoting in excel
	file = xlsx.NewFile()
	sheet = file.AddSheet("Sheet1")
	for _, card := range cards {
		row = sheet.AddRow()
		for _, column := range client.buildOutputLine(card) {
			cell = row.AddCell()
			cell.Value = column
		}
	}
	if err != nil {
		fmt.Printf(err.Error())
	}

	err = file.Write(os.Stdout)
	if err != nil {
		fmt.Printf(err.Error())
	}

	return err
}

func (client *TrelloClient) formatterMarkdown(cards []*TrelloCardSearchResult) error {
	var err error
	for _, card := range cards {
		linebuf := []string{}
		linebuf = append(linebuf, "# "+strings.TrimSpace(card.Name))
		s := ""
		for _, label := range card.Labels {
			if label.Name == "" {
				label.Name = "[" + strings.ToUpper(label.Color) + "]"
			}
			s += "<span style=\"background-color: " + label.Color + ";\">" + label.Name + "</span> "
		}
		linebuf = append(linebuf, s)
		linebuf = append(linebuf, "")
		linebuf = append(linebuf, "## Description")
		linebuf = append(linebuf, card.Desc)

		if card.Badges.Comments > 0 {
			comments, err := client.CardComments(card.ID)
			if err == nil {
				linebuf = append(linebuf, "")
				linebuf = append(linebuf, "## Card Comments")
				for _, comment := range comments {
					linebuf = append(linebuf, "")
					linebuf = append(linebuf, "### "+comment.Date+" from @"+comment.MemberCreator.UserName)
					linebuf = append(linebuf, "")
					linebuf = append(linebuf, comment.Data.Text)
					linebuf = append(linebuf, "")
				}
			} else {
				linebuf = append(linebuf, "[Could not read comments for card] ", err.Error())
			}
		}

		if card.Badges.CheckItems > 0 {
			chklists, err := client.CardChecklists(card.ID)
			if err != nil {
				linebuf = append(linebuf, "[Could not read checklist items for card] ", err.Error())
			} else {
				linebuf = append(linebuf, "")
				linebuf = append(linebuf, "## Checklists")

				for _, chklist := range chklists {
					linebuf = append(linebuf, "### "+chklist.Name)
					for _, v := range chklist.CheckItems {
						s := " 1. " + v.Name
						if v.State == "complete" {
							s += " &#x2705; (done)"
						}
						linebuf = append(linebuf, s)
					}
				}
				linebuf = append(linebuf, "")
			}
		}

		linebuf = append(linebuf, "## Card Info")
		linebuf = append(linebuf, " * last activity on "+card.DateLastActivity)
		if card.Due != "" {
			linebuf = append(linebuf, " * due on "+card.Due)
		}
		linebuf = append(linebuf, " * card shortUrl ["+card.ShortURL+"]("+card.ShortURL+")")
		boardName := NameFromID(card.IDBoard, client.TrelloBoards)
		linebuf = append(linebuf, " * board "+boardName)
		linebuf = append(linebuf, " * list "+NameFromID(card.IDList, client.TrelloLists[strings.ToLower(boardName)]))
		linebuf = append(linebuf, "")
		linebuf = append(linebuf, "")
		fmt.Print(strings.Join(linebuf, "\n"))
		fmt.Print(client.config.RowSep)
	}
	return err
}

func (client *TrelloClient) buildMemberSlice(member *TrelloMember) []string {
	list := strings.Split(client.config.SearchResultFields, ",")
	result := []string{}
	for _, v := range list {
		var item string
		v = strings.TrimSpace(strings.ToLower(v))
		switch v {
		case "id":
			item = member.IDMember
		case "url":
			item = member.URL
		case "avatarhash":
			item = member.AvatarHash
		case "bio":
			item = strings.Replace(member.Bio, "\n", "\\n", -1)
		case "confirmed":
			item = strconv.FormatBool(member.Confirmed)
		case "fullname":
			item = member.FullName
		case "idmember":
			item = member.IDMember
		case "initials":
			item = member.Initials
		case "membertype":
			item = member.MemberType
		case "status":
			item = member.Status
		case "name":
			item = member.UserName
		}
		if client.config.QuoteChar != "" {
			item = client.config.QuoteChar + item + client.config.QuoteChar
		}
		result = append(result, item)
	}
	return result
}

func (client *TrelloClient) memberFormatterJSON(members []*TrelloMember) error {
	var err error
	doc := []byte{}
	doc, err = json.Marshal(members)
	if err == nil {
		fmt.Print(string(doc))
		fmt.Print(client.config.RowSep)
	}
	return err
}

func (client *TrelloClient) memberFormatterText(members []*TrelloMember) error {
	var err error
	header := strings.Split(client.config.SearchResultFields, ",")
	fmt.Println(header)
	for _, member := range members {
		colbuf := client.buildMemberSlice(member)
		for i := 0; i < len(header); i++ {
			fmt.Printf("%-20s : %s\n", header[i], colbuf[i])
		}
		fmt.Print("--------")
		fmt.Print(client.config.RowSep)
	}
	return err
}

func (client *TrelloClient) memberFormatterCsv(members []*TrelloMember) error {
	var err error
	header := strings.Join(strings.Split(client.config.SearchResultFields, ","), client.config.ColSep)
	fmt.Println(header)
	for _, member := range members {
		member.Bio = strings.Replace(member.Bio, "\n", "\\n", -1)
		colbuf := client.buildMemberSlice(member)
		fmt.Print(strings.Join(colbuf, client.config.ColSep))
		fmt.Print(client.config.RowSep)
	}
	return err
}

func (client *TrelloClient) memberFormatterExcel(members []*TrelloMember) error {
	var err error
	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	client.config.QuoteChar = "" // we do not need quoting in excel
	file = xlsx.NewFile()
	sheet = file.AddSheet("Sheet1")
	header := strings.Split(client.config.SearchResultFields, ",")
	row = sheet.AddRow()
	for _, column := range header {
		cell = row.AddCell()
		cell.Value = strings.Title(column)
	}

	for _, member := range members {
		row = sheet.AddRow()
		for _, column := range client.buildMemberSlice(member) {
			cell = row.AddCell()
			cell.Value = column
		}
	}
	if err != nil {
		fmt.Printf(err.Error())
	}

	err = file.Write(os.Stdout)
	if err != nil {
		fmt.Printf(err.Error())
	}

	return err
}

func (client *TrelloClient) memberFormatterMarkdown(members []*TrelloMember) error {
	var err error
	header := strings.Split(client.config.SearchResultFields, ",")
	for _, member := range members {
		colbuf := client.buildMemberSlice(member)
		fmt.Println("## " + member.FullName)
		fmt.Println()
		for i, headercol := range header {
			fmt.Printf(" * " + strings.Title(headercol) + ": " + colbuf[i] + "\n")
		}
		fmt.Print(client.config.RowSep)
	}
	return err
}

func (client *TrelloClient) outputCards(cards []*TrelloCardSearchResult, format string) error {
	var err error
	switch strings.ToLower(format) {
	case "text":
		err = client.formatterText(cards)
	case "csv":
		err = client.formatterCsv(cards)
	case "json":
		err = client.formatterJSON(cards)
	case "excel":
		err = client.formatterExcel(cards)
	case "markdown":
		err = client.formatterMarkdown(cards)
	default:
		err = errors.New("INVALID_OUTPUT_FORMAT")
	}
	return err
}

func (client *TrelloClient) outputMembers(cards []*TrelloMember, format string) error {
	var err error
	switch strings.ToLower(format) {
	case "text":
		err = client.memberFormatterText(cards)
	case "csv":
		err = client.memberFormatterCsv(cards)
	case "json":
		err = client.memberFormatterJSON(cards)
	case "excel":
		err = client.memberFormatterExcel(cards)
	case "markdown":
		err = client.memberFormatterMarkdown(cards)
	default:
		err = errors.New("INVALID_OUTPUT_FORMAT")
	}
	return err
}

func (client *TrelloClient) handleAtCommand(command string) error {
	var err error
	command = strings.ToLower(command)
	cmd := strings.Split(command, " ")[0]
	parms := strings.TrimSpace(strings.Replace(command, cmd, "", -1))

	switch cmd {
	case "@fields":
		client.config.SearchResultFields = parms
	case "@format":
		client.config.Format = parms
	case "@colsep":
		client.config.ColSep = parms
	case "@rowsep":
		client.config.RowSep = parms
	case "@limit":
		client.config.CardLimit, err = strconv.Atoi(parms)
	}
	return err
}

func (client *TrelloClient) parseQuery(queryString string) (string, error) {
	var err error
	lines := strings.Split(queryString, "\n")
	query := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Split(line, "//")[0] // dirty trick to get rid of the comments, if any
		if line != "" {
			if line[0] == '@' {
				err = client.handleAtCommand(line)
				if err != nil {
					return "", err
				}
			} else {
				query += line + " "
			}
		}
	}

	return query, err
}

func isFile(filename string) bool {
	var err error
	var fi os.FileInfo
	fi, err = os.Stat(filename)
	if err != nil || (err == nil && fi.IsDir()) {
		return false
	}
	return true
}

func (client *TrelloClient) loadQuery(filename string) (string, error) {
	var err error
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return client.parseQuery(string(data))
}

func (client *TrelloClient) cmdSearch() error {
	var err error
	query := flag.Arg(flag.NArg() - 1)
	limit := client.config.CardLimit

	if isFile(query) {
		query, err = client.loadQuery(query)
		if err != nil {
			fmt.Println("Could not load query:", err.Error())
			return err
		}
	}

	cards, err := client.SearchCards(query, limit)
	if err != nil {
		fmt.Println("Error searching for cards:", err.Error())
	} else {
		err = client.outputCards(cards, client.config.Format)
		if err != nil {
			fmt.Println("Error writing search result:", err.Error())
		}
	}
	return err
}

func (client *TrelloClient) cmdMembers() error {
	var err error
	board := flag.Arg(flag.NArg() - 1)
	boardID := IDFromName(board, client.TrelloBoards)
	members, err := client.BoardMembers(boardID)
	if err != nil {
	} else {
		err = client.outputMembers(members, client.config.Format)
	}
	return err
}

func (client *TrelloClient) cmdBoards() error {
	var err error
	format := strings.ToLower(client.config.Format)
	if format == "excel" || format == "markdown" || format == "json" {
		err = errors.New("Format not supported for this operation.")
	}
	for _, board := range client.TrelloBoards {
		fmt.Println("Board" + client.config.ColSep + board.Name + client.config.ColSep + board.ID)
		for _, list := range client.TrelloLists[strings.ToLower(board.Name)] {
			fmt.Println("List" + client.config.ColSep + list.Name + client.config.ColSep + list.ID)
		}
		fmt.Println()
	}
	return err
}

func showUsage() {
	fmt.Println(`
tres -- Trello Search for the command line

Usage: tres [options] [command]

Commands:
    search              'trello_search_query' | <filename>
                        see http://help.trello.com/article/808-searching-for-cards-all-boards
                        literal search query must be enclosed in single quotes
                        filename must contain a literal query without single quotes
    members "<name>"    retrieve members of the specified board
    boards              retrieve board name/id and and list name/id for each board

Options:  
    --colsep <string>   set column separator for result columns
    --rowsep <string>   set row separator for result lines
    --fields <string>   a comma-separated list of result field names for a search 
    --format <string>   specify output format (one of: text|excel|csv|json|markdown)
    --limit <n>         limit number of resulting cards (default 200)

List of field names:
    attachmentcount     hasdesc             labelcolors
    boardname           id                  labels
    checked             idattachmentcover   listname
    closed              idboard             name
    commentcount        idchecklists        pos
    comments            idlabels            shortlink
    datelastactivity    idlist              shorturl
    desc                idmembers           subscribed
    due                 idmembersvoted      url
    email               idshort

Environment vars used:
    TRELLO_KEY          your Trello API key
    TRELLO_TOKEN        your Trello API token
    TRELLO_USER         optional (defaults to "me"), you Trello API user name

If anything goes wrong, the tool exits with a return code of 1.

	`)
}

func main() {
	var err error
	trello := newTrelloClient()

	flag.BoolVar(&trello.config.ShowUsage, "help", false, "Display help message")
	flag.StringVar(&trello.config.ColSep, "colsep", "\t", "column separator for search fields")
	flag.StringVar(&trello.config.RowSep, "rowsep", "\n", "row separator for result lines")
	flag.StringVar(&trello.config.QuoteChar, "quotechar", "", "quote string for columns")
	flag.StringVar(&trello.config.SearchResultFields, "fields", "name", "list of result field names")
	flag.StringVar(&trello.config.Format, "format", "text", "output format (text|excel|csv|json|markdown)")
	flag.IntVar(&trello.config.CardLimit, "limit", 200, "limit of cards to retrieve")
	flag.BoolVar(&trello.config.NumberOutput, "number", false, "display row numbers for output lines")
	flag.StringVar(&trello.config.BoardName, "board", "", "")
	flag.StringVar(&trello.config.ListName, "list", "", "")
	flag.Parse()

	if trello.config.ShowUsage || len(flag.Args()) == 0 {
		showUsage()
		os.Exit(1)
	}

	trello.config.Command = strings.ToLower(strings.TrimSpace(flag.Args()[0]))
	type errFunc func() error
	cmds := map[string]errFunc{
		"search":  trello.cmdSearch,
		"members": trello.cmdMembers,
		"boards":  trello.cmdBoards,
	}

	f, present := cmds[trello.config.Command]
	if present {
		err = trello.loadBoardInfo()
		if err == nil {
			err = f()
		}
	} else {
		err = errors.New("Unknown command " + trello.config.Command)
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

//

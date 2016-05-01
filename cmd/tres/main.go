package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/derlinkshaender/tres"
)

var config = &tres.Config{}

func init() {
	flag.BoolVar(&config.ShowUsage, "help", false, "Display help message")
	flag.StringVar(&config.ColSep, "colsep", "\t", "column separator for search fields")
	flag.StringVar(&config.RowSep, "rowsep", "\n", "row separator for result lines")
	flag.StringVar(&config.QuoteChar, "quotechar", "", "quote string for columns")
	flag.StringVar(&config.SearchResultFields, "fields", "name", "list of result field names")
	flag.StringVar(&config.Format, "format", "text", "output format (text|excel|csv|json|markdown)")
	flag.IntVar(&config.CardLimit, "limit", 200, "limit of cards to retrieve")
	flag.BoolVar(&config.NumberOutput, "number", false, "display row numbers for output lines")
	flag.StringVar(&config.BoardName, "board", "", "")
	flag.StringVar(&config.ListName, "list", "", "")
}

func main() {
	flag.Parse()
	if config.ShowUsage || len(flag.Args()) == 0 {
		showUsage()
		os.Exit(1)
	}

	var err error
	trello := tres.NewTrelloClient(config)
	config.Command = strings.ToLower(strings.TrimSpace(flag.Args()[0]))
	type errFunc func() error
	cmds := map[string]errFunc{
		"search":  trello.Search,
		"members": trello.FetchAllMembers,
		"boards":  trello.FetchAllBoards,
	}

	f, present := cmds[config.Command]
	if present {
		err = trello.FetchBoardInfo()
		if err == nil {
			err = f()
		}
	} else {
		err = errors.New("Unknown command " + config.Command)
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
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

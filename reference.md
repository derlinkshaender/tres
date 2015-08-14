
# tres &mdash; search Trello from the shell

## Design considerations

The main reason for `tres` was to be able to integrate Trello in some of my shell scripts of have
a quick way to find these dozen cards that are hidden in a 400 card board with 8 lists that have
a certain label assigned. In my daily work I'm a Head of Product Data Management and ETL. `tres`
and othere tools I have written allow more automation and reduce the risk of error prone and boring manual work.

I decided to give options in a query file a higher priority than the command line options. Though this
may sound counter-intuitive at first, I learned that this way users cannot accidentally create the wrong
output if you provided a query file. So even non-experienced users can get their feet wet on the command
line and be safer when using this tool.

**Important!** `tres` always writes to the standard output to allow easy piping and output redirection.
This means that also the "excel" output format will be written to StdOut! You have to redirect this
to a file or you will just see garbled stuff on the console. Do it like so:

    ./tres --format excel --fields "name,shorturl"  search 'board:"Welcome Board" list:"Basic Stuff" '  > file.xlsx

An important aspect is "usability" from a shell user's point of view. Keep it simple, do one thing well,
use the standards and provide a nice usage screen. Be pragmatic and do not introduce unnecessary complexity because YAGNI.


## tres commands

### search

This will be the most-used command and it excepts a query string enclosed in single quotes or the name of
a file containng a saved query (see below).

### boards

This command displays a list of all board and list you have access to. The output contains the object type,
the name of the board/list and the 24char hex id for Trello.

### members

Display all members of a specific board. A quick way to see who can access this board.


## Output formats

Here is the list of possible output formats

 * csv
 * excel
 * text
 * markdown
 * json

These formats should be fairly self-explanatory. If you use "excel", the file is written as _.xlsx_ format
(hat tip to Geoffrey J. Teale for his great Go package!).
The "markdown" format is a dirty hack to suits my special and personal markdown needs. If you do not like
the output, this is open source! Go ahead and fork it. ;-)
JSON output is always the complete card response from the Trello API. If figured if you need JSON, you are going
to process it anyway, so I might as well stick with the well-documented JSON format of a Trello card and always
ignore the fields option and yield the complete JSON.


## Saved queries

This feature allows you to store your search query in a text file (UTF8, LF line ends)
and even specify output format or fields to display. All you then have to do is provide
the file name as the parameter of the `seach` command instead of a single-quoted query string.
As our friends in France would say: _tres chic!_ ;-)

The text file may contain comments (C-style with a double slash).
Each line starting with an at-sign ("@") is a processing instruction (see below).
Here is an example of a query file.

    // display a list of cards with a cover

    @fields listname, name, labels
    @format csv
    @colsep ;  // you can use line comments

    board:"Welcome Board" AND has:cover

Possible at-commands are

 * @fields
 * @limit
 * @colsep
 * @rowsep
 * @format

These commands work just as the command line options for `tres`. There is, however, a little difference:
command line options do **NOT** override the @-commands in the query file. This is by design and prevents
users to accidentally overwrite important options you provided in the qery file (think "user first").

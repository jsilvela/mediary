package main

import (
	"bufio"
	"fmt"
	"github.com/jsilvela/diary"
	"github.com/jsilvela/diary/filters"
	"github.com/jsilvela/diary/reports"
	"log"
	"os"
	"strings"
	"time"
)

func main() {

	var reqs diary.Diary
	var filename string

	if len(os.Args) < 2 {
		filename = "./mediary.txt"
	} else {
		filename = os.Args[1]
		var err error
		reqs, err = diary.Read(filename)
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(reqs) > 0 {
		w, h := reqs.LatestWritten(), reqs.LatestHappened()
		if w.WrittenTime != h.WrittenTime {
			fmt.Printf("\n**** Latest written:\n%s\n\n**** Latest happened:\n%s\n",
				w, h)
		} else {
			fmt.Printf("\n**** Latest written:\n%s\n", h)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	inText := false
	var record, nullrec diary.Record
	dirty := false
	inrec := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		} else if line == "}" && inrec {
			reqs.AddEntry(record)
			record = nullrec
			inText = false
			dirty = true
			inrec = false
		} else if line == "}" {
			fmt.Printf("Closing brace closes nothing\n")
			continue
		} else if line == "new {" || line == "new{" {
			if inrec {
				fmt.Println("Can't open a new record while we're" +
					" in a record")
				continue
			}
			inrec = true
		} else if line == "exit" {
			if inrec {
				fmt.Println("Can't exit while writing a record")
				continue
			}
			if dirty {
				err := diary.Write(filename, reqs)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Wrote records")
				}
			} else {
				fmt.Println("Unmodified, not saving")
			}
			return
		} else if inrec {
			frag := strings.SplitN(line, ":", 2)
			if len(frag) == 2 {
				sel, content := frag[0], frag[1]
				switch {
				case sel == "time":
					if !processTime(content, &record) {
						fmt.Printf("Date not understood: %s\n", content)
					}
				case sel == "tags":
					processTags(content, &record)
				case sel == "text":
					inText = processText(content, &record)
				default:
					if inText {
						inText = processText(line, &record)
					} else {
						fmt.Printf("Invalid tag: %s\n", sel)
						continue
					}
				}
			} else {
				if inText {
					inText = processText(line, &record)
				} else {
					fmt.Println("Invalid. Key must be supplied")
					continue
				}
			}
		} else {
			diaryCopy := reqs
			frags := strings.Split(line, " ")
			fs, rep := parseScript(frags)
			evalScript(diaryCopy, fs, rep)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

type funcCells struct {
	f    func(diary.Diary) *diary.Diary
	next *funcCells
}

type reportCell struct {
	f   func(diary.Diary) []string
	arg string
}

func parseScript(frags []string) (*funcCells, *reportCell) {
	var funcs *funcCells
	var rep *reportCell

	for len(frags) > 0 {
		switch frags[0] {
		case "week":
			funcs = &funcCells{filters.ByWeek, funcs}
			frags = frags[1:]
		case "month":
			funcs = &funcCells{filters.ByMonth, funcs}
			frags = frags[1:]
		case "by-tag":
			tag := frags[1]
			byTag := func(d diary.Diary) *diary.Diary {
				return filters.ByTag(d, tag)
			}
			funcs = &funcCells{byTag, funcs}
			frags = frags[2:]
		case "tags":
			rep = &reportCell{reports.Tags, ""}
			frags = frags[1:]
		case "series":
			rep = &reportCell{reports.TimeSeries, ""}
			frags = frags[1:]
		case "latest":
			latest := func(d diary.Diary) []string {
				rep := reports.Latest(d)
				out := make([]string, len(rep))
				i := 0
				for tag, t := range rep {
					out[i] = fmt.Sprintf("%s\t=> %v\n",
						tag, t.Format("Mon 2 Jan 2006"))
					i++
				}
				return out
			}
			rep = &reportCell{latest, ""}
			frags = frags[1:]
		default:
			fmt.Printf("Unrecognized command: %s\n", frags[0])
			return nil, nil
		}
	}

	return funcs, rep
}

func evalScript(d diary.Diary, fs *funcCells, rep *reportCell) {
	if fs != nil {
		evalScript(*fs.f(d), fs.next, rep)
	} else if rep != nil {
		fmt.Println(rep.f(d))
	}
}

// Parse text declaration when writing new entry
func processText(line string, record *diary.Record) bool {
	if line == "===" {
		return false
	}
	if record.Text == "" {
		record.Text = strings.TrimSpace(line)
	} else {
		record.Text = strings.Join([]string{record.Text, "\n", line}, "")
	}
	return true
}

func processTime(line string, record *diary.Record) bool {
	const shortForm = "2006-01-02"
	if strings.TrimSpace(line) == "today" {
		record.EventTime = time.Now()
		return true
	} else {
		t, err := time.Parse(shortForm, strings.TrimSpace(line))
		if err != nil {
			return false
		}
		record.EventTime = t
		return true
	}
}

func processTags(line string, record *diary.Record) {
	frags := strings.Split(line, ",")
	tags := make([]string, len(frags))
	for i := 0; i < len(frags); i++ {
		tags[i] = strings.TrimSpace(frags[i])
	}
	record.Tags = tags
}

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

type status struct {
	text   string
	time   time.Time
	tags   []string
	diar   diary.Diary
	dirty  bool
}

type state func(s *status, line string) state

func parseTop(s *status, line string) state {
	log.Println("TOP")
	if line == "" {
		return parseTop
	} else if line == "new {" || line == "new{" {
		return parseRecord
	} else if line == "exit" || line == "quit" {
		return nil
	}
	diaryCopy := s.diar
	frags := strings.Split(line, " ")
	fs, rep := parseScript(frags)
	evalScript(diaryCopy, fs, rep)
	return parseTop
}

func parseText(s *status, line string) state {
	log.Println("TEXT")
	switch {
	case line == "===":
		return parseRecord
	case line == "}":
		newrec := diary.Record{EventTime: s.time, Tags: s.tags, Text: s.text}
		(&(s.diar)).AddEntry(newrec)
		s.dirty = true
		return parseTop
	case strings.HasPrefix(line, "tags:"):
		return parseRecord(s, line)
	case strings.HasPrefix(line, "time:"):
		return parseRecord(s, line)
	case s.text == "":
		s.text = strings.TrimSpace(line)
	default:
		s.text = s.text + "\n" + line
	}
	return parseText
}

func parseRecord(s *status, line string) state {
	log.Println("RECORD")
	frag := strings.SplitN(line, ":", 2)
	if len(frag) == 2 {
		sel, content := frag[0], strings.TrimSpace(frag[1])
		switch {
		case sel == "time":
			t, err := processTime(content)
			if err != nil {
				log.Printf("Date not understood: %s\n%s\n", content, err)
				return parseRecord
			}
			s.time = t
		case sel == "tags":
			frags := strings.Split(content, ",")
			tags := make([]string, len(frags))
			for i := 0; i < len(frags); i++ {
				tags[i] = strings.TrimSpace(frags[i])
			}
			s.tags = tags
		case sel == "text":
			return parseText(s, content)
		default:
			log.Printf("Invalid tag: %s\n", sel)
		}
		return parseRecord
	} else if line == "}" {
		newrec := diary.Record{EventTime: s.time, Tags: s.tags, Text: s.text}
		(&(s.diar)).AddEntry(newrec)
		s.dirty = true
		return parseTop
	} else {
		log.Printf("Unexpected input while buildging record: %s\n", line)
		return parseTop
	}
}

func parseLines(scanner *bufio.Scanner, reqs diary.Diary) (*status, error) {
	parse := parseTop
	s := status{diar: reqs}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parse = parse(&s, line)
		if parse == nil {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &s, nil
}

func main() {
	var reqs diary.Diary
	var filename string

	if len(os.Args) < 2 {
		filename = "./mediary.txt"
	} else {
		filename = os.Args[1]
		var err error
		_, err = os.Stat(filename)
		if err != nil {
			log.Fatal(err)
		}
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		reqs, err = diary.Read(f)
	}

	if len(reqs) > 0 {
		w, h := reqs.LatestWritten(), reqs.LatestHappened()
		if w.WrittenTime != h.WrittenTime {
			log.Printf("\n**** Latest written:\n%s\n\n**** Latest happened:\n%s\n",
				w, h)
		} else {
			log.Printf("\n**** Latest written:\n%s\n", h)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	status, err := parseLines(scanner, reqs)
	if err != nil {
		log.Fatalf("Error reading input lines: %s", err.Error())
	}
	if status.dirty {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Error opening file for output: %s", err.Error())
		}
		err = diary.Write(f, status.diar)
		if err != nil {
			log.Fatalf("Error writing diary to file: %s", err.Error())
		}
		err = f.Close()
		if err != nil {
			log.Fatalf("Error closing output file: %s", err.Error())
		}
	} else {
		log.Println("Diary not modified, nothing to write out")
	}
	os.Exit(0)
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
			log.Printf("Unrecognized command: %s\n", frags[0])
			return nil, nil
		}
	}

	return funcs, rep
}

func evalScript(d diary.Diary, fs *funcCells, rep *reportCell) {
	if fs != nil {
		evalScript(*fs.f(d), fs.next, rep)
	} else if rep != nil {
		log.Println(rep.f(d))
	}
}

func processTime(line string) (time.Time, error) {
	const shortForm = "2006-01-02"
	if strings.TrimSpace(line) == "today" {
		return time.Now(), nil
	}
	t, err := time.Parse(shortForm, strings.TrimSpace(line))
	if err != nil {
		return t, err
	}
	return t, nil
}

package main

import (
	"bufio"
	"github.com/jsilvela/diary"
	"strings"
	"testing"
)

func TestParseInput(t *testing.T) {
	var d diary.Diary
	str := `new {
time: 2001-01-20
tags: myTag
text: hello, world!
}
exit
`
	reader := strings.NewReader(str)
	scanner := bufio.NewScanner(reader)
	status, err := parseLines(scanner, d)
	if err != nil {
		t.Error(err)
	}
	if len(status.diar) != 1 {
		t.Errorf("Unexpected diary %v", status.diar)
	}
	if status.diar[0].Text != "hello, world!" {
		t.Errorf("Unexpected text %v", status.diar[0].Text)
	}
	if status.diar[0].Tags[0] != "myTag" {
		t.Errorf("Unexpected tags %v", status.diar[0].Tags[0])
	}
}

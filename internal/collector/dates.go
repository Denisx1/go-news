package collector

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reDMY    = regexp.MustCompile(`(\d{2})\.(\d{2})\.(\d{4})(?:\D+(\d{1,2}):(\d{2}))?`)
	reUADate = regexp.MustCompile(`(\d{1,2})\s+([а-яіїєґ]+)\s+(\d{4})`)

	uaMonths = map[string]int{
		"січня": 1, "лютого": 2, "березня": 3, "квітня": 4,
		"травня": 5, "червня": 6, "липня": 7, "серпня": 8,
		"вересня": 9, "жовтня": 10, "листопада": 11, "грудня": 12,
	}
)

// parseDMY понимает «13 жовтня 2026» и «13.10.2026 12:30».
func parseDMY(s string) time.Time {
	if m := reUADate.FindStringSubmatch(strings.ToLower(s)); m != nil {
		if mo, ok := uaMonths[m[2]]; ok {
			d, _ := strconv.Atoi(m[1])
			y, _ := strconv.Atoi(m[3])
			return time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.Local)
		}
	}
	m := reDMY.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}
	}
	h, min := 0, 0
	if m[4] != "" {
		h, _ = strconv.Atoi(m[4])
		min, _ = strconv.Atoi(m[5])
	}
	d, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	y, _ := strconv.Atoi(m[3])
	return time.Date(y, time.Month(mo), d, h, min, 0, 0, time.Local)
}

// IsToday — время приходится на сегодняшний день (в time.Local).
func IsToday(t time.Time) bool {
	y, m, d := time.Now().Date()
	ty, tm, td := t.In(time.Local).Date()
	return y == ty && m == tm && d == td
}

package i18n

import "time"

var months = map[Locale][12]string{
	EN: {"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"},
	UK: {"Січень", "Лютий", "Березень", "Квітень", "Травень", "Червень",
		"Липень", "Серпень", "Вересень", "Жовтень", "Листопад", "Грудень"},
}

var shortMonths = map[Locale][12]string{
	EN: {"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	UK: {"січ", "лют", "бер", "кві", "тра", "чер", "лип", "сер", "вер", "жов", "лис", "гру"},
}

var weekdays = map[Locale][7]string{
	EN: {"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
	UK: {"нд", "пн", "вт", "ср", "чт", "пт", "сб"},
}

func (l Locale) Month(m time.Month) string {
	table, ok := months[l]
	if !ok {
		table = months[Default]
	}
	return table[int(m)-1]
}

func (l Locale) ShortMonth(m time.Month) string {
	table, ok := shortMonths[l]
	if !ok {
		table = shortMonths[Default]
	}
	return table[int(m)-1]
}

func (l Locale) Weekday(d time.Weekday) string {
	table, ok := weekdays[l]
	if !ok {
		table = weekdays[Default]
	}
	return table[int(d)]
}

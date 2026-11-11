package reports_handlers

import "strconv"

func itoa(n int) string { return strconv.Itoa(n) }

// doctorLabel is the doctor's name, or the "no doctor" label.
func doctorLabel(tr translate, name string) string {
	if name == "" {
		return tr("reports.no_doctor")
	}
	return name
}

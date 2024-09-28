package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// gobal variables
const serverAddress = ":3030"

var templates *template.Template

var veX_nRegex *regexp.Regexp

var veX_uRegex *regexp.Regexp

// dummy datas
var volunteerDatabase1Q1 = map[string][]string{
	"Andy": {"2024-01-21", "2024-01-28"},
	"Jack": {},
	"Tim":  {"2024-02-11", "2024-01-18", "2024-01-25"},
	"Bill": {"3/24/2024"},
}

var volunteerDatabase1Q2 = map[string][]string{
	"Andy":  {"2024-04-07", "2024-04-14"},
	"Jack":  {"2024-05-05", "2024-05-12"},
	"Tim":   {},
	"Roger": {"2024-06-30", "2024-06-23"},
}

var volunteerDatabase2Q1 = map[string][]string{
	"Andy": {"2024-01-21", "2024-01-28"},
	"Jack": {},
	"Tim":  {"2024-02-11", "2024-01-18", "2024-01-25"},
	"Bob":  {"2024-02-28"},
}

var volunteerDatabase2Q2 = map[string][]string{
	"Andy":   {"2024-04-07", "2024-04-14"},
	"Jack":   {"2024-05-05", "2024-05-12"},
	"Tim":    {},
	"George": {"2024-05-19"},
}

var scheduleDatabase = []scheduleStruct{ // this simulates the database just dumping a list of organized data at the program. Program needs to sort it and then put it into frontend
	{
		"First Volunteers 2024 Q1",
		volunteerDatabase1Q1,
		"2024-01-01",
		"2024-04-01",
		[]string{"Su"},
		3,
		3,
	},
	{
		"First Volunteers 2024 Q2",
		volunteerDatabase1Q2,
		"2024-04-01",
		"2024-07-01",
		[]string{"Su", "We"},
		1,
		3,
	},
	{
		"Second Volunteers 2024 Q1",
		volunteerDatabase2Q1,
		"2024-01-01",
		"2024-04-01",
		[]string{"Su"},
		3,
		3,
	},
	{
		"Second Volunteers 2024 Q2",
		volunteerDatabase2Q2,
		"2024-04-01",
		"2024-07-01",
		[]string{"Su", "Th"},
		2,
		3,
	},
}

var sortedSchedules map[string]*scheduleStruct

// useful structs

type weekdaysStruct struct {
	Sunday    bool
	Monday    bool
	Tuesday   bool
	Wednesday bool
	Thursday  bool
	Friday    bool
	Saturday  bool
}

type scheduleStruct struct {
	ScheduleName       string
	VolunteerEntries   map[string][]string // need to change this to a []map[string][]string
	MinDate            string
	MaxDate            string
	VolunteerDays      []string
	ShiftsOff          int
	VolunteersPerShift int
}

type handlerInfoStruct struct {
	address  string
	funcName string
	method   string
}

// have to uppercase struct members to make them available (public) to html/template ParseFiles function
type base_pageStruct struct {
	Top_bar      top_barStruct
	Left_column  left_columnStruct
	Right_column right_columnStruct
}

type top_barStruct struct {
	User                 string         // Seth
	Saved_schedules      []string       // First Volunteers 2024 Q1, First Volunteers 2024 Q2, Second Volunteers 2024 Q1, or Second Volunteers 2024 Q2
	Current_schedule     string         // First Volunteers 2024 Q1
	Min_date             string         // 5/19/24
	Max_date             string         // 6/9/24
	Volunteer_days       weekdaysStruct // M T W R F S and/or S
	Shifts_off           int            // x weeks between volunteering
	Volunteers_per_shift int            // min = 1, max = # of volunteers
	Allow_copy           bool           // bool on whether thee schedule select element should have the copy-current-schedule option
}

type left_columnStruct struct {
	Volunteer_column  []volunteer_entryStruct
	Existing_schedule bool
}

type right_columnStruct struct {
	Num_weeks []string
}

type volunteer_entryStruct struct {
	IdIndex string
	Name    string
	Dates   []string
}

// helper functions

func mustAtoI(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	return i
}

func veX_n(value any) string {
	if reflect.TypeOf(value).Kind() == reflect.Int || reflect.TypeOf(value).Kind() == reflect.String {
		return fmt.Sprintf("ve%v-n", value)
	} else {
		log.Fatal("Fatal error in veX_n. Value was not int or string.")
		return ""
	}
}

func veX_u(value any) string {
	if reflect.TypeOf(value).Kind() == reflect.Int || reflect.TypeOf(value).Kind() == reflect.String {
		return fmt.Sprintf("ve%v-u", value)
	} else {
		log.Fatal("Fatal error in veX_u. Value was not int or string.")
		return ""
	}
}

func sortSchedulesToMap(schedules []scheduleStruct) map[string]*scheduleStruct {
	scheduleMap := make(map[string]*scheduleStruct, len(schedules))
	for _, val := range schedules {
		scheduleMap[val.ScheduleName] = &val
	}
	return scheduleMap
}

func getStringMapKeys[stringMapType ~map[string]V, V any](stringMap stringMapType, sorted bool) []string {
	// code adapted from https://pkg.go.dev/golang.org/x/exp/maps#Keys
	// see https://go.dev/blog/intro-generics for an explanation about the ~
	keys := make([]string, 0, len(stringMap))
	for key := range stringMap {
		if key != "" { //exclude empty keys
			keys = append(keys, key)
		}
	}
	if sorted {
		slices.Sort(keys)
	}
	return keys
}

func createWeekdaysStruct(weekdays_slice []string) weekdaysStruct {
	return_val := weekdaysStruct{}
	if slices.Contains(weekdays_slice, "Su") {
		return_val.Sunday = true
	}
	if slices.Contains(weekdays_slice, "Mo") {
		return_val.Monday = true
	}
	if slices.Contains(weekdays_slice, "Tu") {
		return_val.Tuesday = true
	}
	if slices.Contains(weekdays_slice, "We") {
		return_val.Wednesday = true
	}
	if slices.Contains(weekdays_slice, "Th") {
		return_val.Thursday = true
	}
	if slices.Contains(weekdays_slice, "Fr") {
		return_val.Friday = true
	}
	if slices.Contains(weekdays_slice, "Sa") {
		return_val.Saturday = true
	}
	return return_val
}

func prepareTemplateStructs(scheduleName string, bIsExistingAndCopyable bool) base_pageStruct {
	if scheduleName == "" {
		volunteer_entries_slice := []volunteer_entryStruct{{"0", "", []string{}}}
		right_column_data := right_columnStruct{make([]string, 6)}
		left_column_data := left_columnStruct{volunteer_entries_slice, false}
		top_bar_data := top_barStruct{"Seth", getStringMapKeys(sortedSchedules, true), "", "", "", weekdaysStruct{}, -1, -1, bIsExistingAndCopyable}
		return base_pageStruct{top_bar_data, left_column_data, right_column_data}
	} else {
		schedule := sortedSchedules[scheduleName]
		volunteer_entries_slice := make([]volunteer_entryStruct, 0, len(schedule.VolunteerEntries)+1)
		i := 0
		keys := getStringMapKeys(schedule.VolunteerEntries, true)
		for index, key := range keys {
			volunteer_entries_slice = append(volunteer_entries_slice, volunteer_entryStruct{fmt.Sprint(index), key, schedule.VolunteerEntries[key]})
			i++
		}
		volunteer_entries_slice = append(volunteer_entries_slice, volunteer_entryStruct{fmt.Sprint(len(schedule.VolunteerEntries)), "", []string{}}) // need a blank volunteer entry
		weeks := 0
		if schedule.MaxDate != "" && schedule.MinDate != "" {
			max_date, err := time.Parse("2006-01-02", schedule.MaxDate)
			if err != nil {
				log.Fatal(err)
			}
			min_date, err := time.Parse("2006-01-02", schedule.MinDate)
			if err != nil {
				log.Fatal(err)
			}
			weeks = int(math.Floor(max_date.Sub(min_date).Abs().Hours() / 24 / 7)) // (floor of the (((absolute value of (max date minus min date)) in hours) divided by 24hrs/dy divided by 7dy/wk)) converted to an int
		}
		selected_days := createWeekdaysStruct(schedule.VolunteerDays)
		right_column_data := right_columnStruct{make([]string, weeks)}
		left_column_data := left_columnStruct{volunteer_entries_slice, bIsExistingAndCopyable}
		top_bar_data := top_barStruct{"Seth", getStringMapKeys(sortedSchedules, true), scheduleName, schedule.MinDate, schedule.MaxDate, selected_days, schedule.ShiftsOff, schedule.VolunteersPerShift, bIsExistingAndCopyable}
		return base_pageStruct{top_bar_data, left_column_data, right_column_data}
	}
}

func requestIsValid(w http.ResponseWriter, r *http.Request, intended_url string, intended_method string) bool {
	validRequest := true
	//log.Printf(`Intended URL: "%s"; Intended Method: "%s"`, intended_url, intended_method)
	if r.URL.Path != intended_url {
		http.Error(w, "404 not found.", http.StatusNotFound)
		validRequest = false
		log.Printf("Provided URL does not match the URL this function is intended to serve: %s (intended), %s (provided)", intended_url, r.URL.Path)
	} else if r.Method != intended_method {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		validRequest = false
		log.Printf("Method not supported: %s", r.Method)
	}
	//log.Printf("Is the request valid? %s", strconv.FormatBool(validRequest))
	return validRequest
}

func extractVolunteers(form url.Values) map[string][]string {
	// loop over the keys on r.Form and if the key is veX-n and there is a corresponding veX-u and form[veX-n][0] is not "",
	// then save the name as the key and the dates (cleaned of any "" values) as the value in volunteers.
	// NOTE: this function does not check that len(form[veX-n]) == 1 because this shouldn't be called without prior validation of form.
	var volunteers = map[string][]string{}
	keys := getStringMapKeys(form, true)
	for _, v := range keys {
		if veX_nRegex.MatchString(v) && slices.Contains(keys, fmt.Sprintf("%su", v[:len(v)-1])) && form[v][0] != "" {
			volunteers[form[v][0]] = slices.DeleteFunc(form[fmt.Sprintf("%su", v[:len(v)-1])], func(s string) bool { return s == "" })
		}
	}
	return volunteers
}

func parametersValidated(form url.Values, keys_to_check ...string) bool {
	// possbile keys_to_check: "schedule-selection", "schedule-name", "IdIndex" "veX-X", "min-date", "max-date", "weekday", "shifts-off", "per-shift"
	mustBeLen1 := []string{"schedule-selection", "schedule-name", "IdIndex", "min-date", "max-date", "shifts-off", "per-shift"} // veX-n must also be len 1, but that is handled later
	for _, keyToCheck := range keys_to_check {
		if slices.Contains(mustBeLen1, keyToCheck) {
			if len(form[keyToCheck]) != 1 {
				log.Printf("ERROR: \"%s\" does not have length of 1", keyToCheck)
				return false
			}
		}
		if keyToCheck == "schedule-selection" { // FIX THIS, some other values are valid. See handleScheduleSelection
			scheduleKeys := getStringMapKeys(sortedSchedules, false)
			allowedValues := make([]string, 0, len(scheduleKeys)+2)
			allowedValues = append(allowedValues, scheduleKeys...)
			allowedValues = append(allowedValues, "new-schedule", "copy-current-schedule")
			if !slices.Contains(allowedValues, form[keyToCheck][0]) {
				log.Printf("ERROR: Value of \"%s\" for \"%s\" was not a known response", form[keyToCheck][0], keyToCheck)
				return false
			}
		} else if keyToCheck == "IdIndex" || keyToCheck == "shifts-off" {
			if form[keyToCheck][0] != "" {
				value, err := strconv.Atoi(form[keyToCheck][0])
				if err != nil {
					log.Printf("ERROR: \"%s\" cannot be converted to an integer", keyToCheck)
					return false
				}
				if value < 0 {
					log.Printf("ERROR: \"%s\" is less than 0", keyToCheck)
					return false
				}
			}
		} else if keyToCheck == "schedule-name" {
			if strings.ContainsAny(form[keyToCheck][0], "\\/:*?\"<>|") {
				log.Printf("ERROR: \"%s\" contains illegal characters (\\/:*?\"<>|)", keyToCheck)
				return false
			}
		} else if keyToCheck == "veX-X" {
			for formKey, formValue := range form {
				if veX_nRegex.MatchString(formKey) {
					if len(formValue) != 1 {
						log.Printf("ERROR: \"%s\" does not have length of 1", keyToCheck)
						return false
					}
				} else if veX_uRegex.MatchString(formKey) {
					for _, stringElement := range formValue {
						if stringElement != "" {
							_, err := time.Parse("2006-01-02", stringElement)
							if err != nil {
								log.Printf("ERROR: \"%s\" value \"%s\" is not in a valid date format (YYYY-MM-DD)", formKey, stringElement)
								return false
							}
						}
					}

				}
			}

		} else if keyToCheck == "min-date" || keyToCheck == "max-date" {
			if form[keyToCheck][0] != "" {
				_, err := time.Parse("2006-01-02", form[keyToCheck][0])
				if err != nil {
					log.Printf("ERROR: \"%s\" is not in a valid date format (YYYY-MM-DD)", keyToCheck)
					return false
				}
			}
		} else if keyToCheck == "weekday" {
			if slices.ContainsFunc(form[keyToCheck], func(s string) bool { return !slices.Contains([]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}, s) }) {
				log.Printf("ERROR: \"%s\" contains non weekday values (Su, Mo, Tu, We, Th, Fr, Sa)", keyToCheck)
				return false
			}
		} else if keyToCheck == "per-shift" {
			if form[keyToCheck][0] != "" {
				value, err := strconv.Atoi(form[keyToCheck][0])
				if err != nil {
					log.Printf("ERROR: \"%s\" cannot be converted to an integer", keyToCheck)
					return false
				}
				if value < 1 {
					log.Printf("ERROR: \"%s\" is less than 1", keyToCheck)
					return false
				}
			}
		} else {
			log.Printf("ERROR: \"%s\" is present but unchecked", keyToCheck)
			return false
		}
	}
	return true
}

// handler functions
var handleFuncMap = map[string]func(http.ResponseWriter, *http.Request){
	"/":                   handleRoot,
	"/select-schedule":    handleSelectSchedule,
	"/add-unavailability": handleAddVolunteerUnavailability,
	"/mod-volunteers":     handleModVolunteers,
	"/save-parameters":    handleSaveParameters,
	"/delete-schedule":    handleDeleteSchedule,
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	//------------------------ UPDATE THIS WHEN COPYING, DUMMY ------------------------
	handlerInfo := handlerInfoStruct{"/", "handleRoot", "GET"}
	//---------------------------------------------------------------------------------
	if !requestIsValid(w, r, handlerInfo.address, handlerInfo.method) {
		log.Printf("Request to %s is invalid!", handlerInfo.funcName)
		return
	}
	base_page_data := prepareTemplateStructs("", false)
	err := templates.ExecuteTemplate(w, "base_page", base_page_data)
	if err != nil {
		log.Fatal(err)
	}
}

func handleSelectSchedule(w http.ResponseWriter, r *http.Request) {
	//------------------------ UPDATE THIS WHEN COPYING, DUMMY ------------------------
	handlerInfo := handlerInfoStruct{"/select-schedule", "handleSelectSchedule", "GET"}
	//---------------------------------------------------------------------------------
	if !requestIsValid(w, r, handlerInfo.address, handlerInfo.method) {
		log.Printf("Request to %s is invalid!", handlerInfo.funcName)
		return
	}
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	if !parametersValidated(r.Form, "schedule-selection", "schedule-name") {
		log.Fatalf("Fatal error in %s. See error message above.", handlerInfo.address)
	}
	log.Printf("Evaluating %s from get: %v", handlerInfo.address, r.Form)
	var base_page_data base_pageStruct
	if r.Form["schedule-selection"][0] == "new-schedule" { // case where no schedule is selected (the blank entry in the select element)
		base_page_data = prepareTemplateStructs("", false)
	} else if r.Form["schedule-selection"][0] == "copy-current-schedule" { // case where copying schedule
		base_page_data = prepareTemplateStructs(r.Form["schedule-name"][0], false)
		base_page_data.Top_bar.Current_schedule = fmt.Sprintf("Copy of %s", base_page_data.Top_bar.Current_schedule)
	} else { // case where selection is not new or copy
		base_page_data = prepareTemplateStructs(r.Form["schedule-selection"][0], true)
	}
	err = templates.ExecuteTemplate(w, "base_page", base_page_data)
	if err != nil {
		log.Fatal(err)
	}
}

func handleAddVolunteerUnavailability(w http.ResponseWriter, r *http.Request) {
	//------------------------ UPDATE THIS WHEN COPYING, DUMMY ------------------------
	handlerInfo := handlerInfoStruct{"/add-unavailability", "handleAddVolunteerUnavailability", "GET"}
	//---------------------------------------------------------------------------------
	if !requestIsValid(w, r, handlerInfo.address, handlerInfo.method) {
		log.Printf("Request to %s is invalid!", handlerInfo.funcName)
		return
	}
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	if !parametersValidated(r.Form, "IdIndex") {
		log.Fatalf("Fatal error in %s. See error message above.", handlerInfo.address)
	}
	log.Printf("Evaluating %s from get: %v", handlerInfo.address, r.Form)
	id_index := r.Form["IdIndex"][0]
	if slices.Contains(r.Form[veX_u(id_index)], "") {
		log.Print("Not adding new blank volunteer unavailability since one blank volunteer is already present.")
		return
	}
	err = templates.ExecuteTemplate(w, "ve_unavailable_single_blank", volunteer_entryStruct{id_index, "", []string{}})
	if err != nil {
		log.Fatal(err)
	}
}

func handleModVolunteers(w http.ResponseWriter, r *http.Request) {
	//------------------------ UPDATE THIS WHEN COPYING, DUMMY ------------------------
	handlerInfo := handlerInfoStruct{"/mod-volunteers", "handleModVolunteers", "GET"}
	//---------------------------------------------------------------------------------
	if !requestIsValid(w, r, handlerInfo.address, handlerInfo.method) {
		log.Printf("Request to %s is invalid!", handlerInfo.funcName)
		return
	}
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	if !parametersValidated(r.Form, "IdIndex") {
		log.Fatalf("Fatal error in %s. See error message above.", handlerInfo.address)
	}
	log.Printf("Evaluating %s from get: %v", handlerInfo.address, r.Form)
	id_index := r.Form["IdIndex"][0]
	count_blanks := 0
	for key, value := range r.Form {
		if veX_nRegex.MatchString(key) {
			if slices.Contains(value, "") {
				count_blanks++
			}
		}
	}
	next_index := 0
	_, ok := r.Form[veX_n(next_index)]
	for ok {
		next_index++
		_, ok = r.Form[veX_n(next_index)]
	}
	//log.Printf("Blanks: %d; IdIndex: %s", count_blanks, id_index)
	if count_blanks == 0 || (slices.Contains(r.Form[veX_n(id_index)], "") && count_blanks <= 1) {
		err = templates.ExecuteTemplate(w, "volunteer_entry", volunteer_entryStruct{fmt.Sprint(next_index), "", []string{}})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleSaveParameters(w http.ResponseWriter, r *http.Request) { // change this to save schedule parameters to include volunteers and dates plus schedule-start-date through shifts-off-before-scheduled-again
	//------------------------ UPDATE THIS WHEN COPYING, DUMMY ------------------------
	handlerInfo := handlerInfoStruct{"/save-parameters", "handleSaveParameters", "POST"}
	//---------------------------------------------------------------------------------
	if !requestIsValid(w, r, handlerInfo.address, handlerInfo.method) {
		log.Printf("Request to %s is invalid!", handlerInfo.funcName)
		return
	}
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	// Perform basic validations
	if !parametersValidated(r.Form, "schedule-selection", "schedule-name") {
		log.Fatalf("Fatal error in %s. See error message above.", handlerInfo.address)
	}
	log.Printf("Evaluating %s from post: %v", handlerInfo.address, r.Form)
	if !parametersValidated(r.Form, "veX-X", "min-date", "max-date", "weekday", "shifts-off", "per-shift") {
		return
	}
	selected_schedule_entry := r.Form["schedule-selection"][0]
	volunteer_entries := extractVolunteers(r.Form)
	if selected_schedule_entry == "new-schedule" {
		log.Print("New schedule")
		newSchedule := scheduleStruct{}
		newSchedule.ScheduleName = r.Form["schedule-name"][0]
		newSchedule.VolunteerEntries = volunteer_entries
		newSchedule.MinDate = r.Form["min-date"][0]
		newSchedule.MaxDate = r.Form["max-date"][0]
		newSchedule.VolunteerDays = r.Form["weekday"]
		if r.Form["shifts-off"][0] != "" {
			newSchedule.ShiftsOff = mustAtoI(r.Form["shifts-off"][0])
		}
		if r.Form["per-shift"][0] != "" {
			newSchedule.VolunteersPerShift = mustAtoI(r.Form["per-shift"][0])
		}
		scheduleDatabase = append(scheduleDatabase, newSchedule) // this simulates an API create message
		sortedSchedules = sortSchedulesToMap(scheduleDatabase)   // need to refresh sortedSchedules now
		log.Printf("%v", newSchedule)
		base_page_data := prepareTemplateStructs(r.Form["schedule-name"][0], true)
		err = templates.ExecuteTemplate(w, "base_page", base_page_data)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("%#v", scheduleDatabase)
		log.Print("Updating schedule")
		database_schedule_entry, ok := sortedSchedules[selected_schedule_entry]
		if ok {
			database_schedule_entry.ScheduleName = r.Form["schedule-name"][0]
			database_schedule_entry.VolunteerEntries = volunteer_entries
			database_schedule_entry.MinDate = r.Form["min-date"][0]
			database_schedule_entry.MaxDate = r.Form["max-date"][0]
			database_schedule_entry.VolunteerDays = r.Form["weekday"]
			if r.Form["shifts-off"][0] != "" {
				database_schedule_entry.ShiftsOff, _ = strconv.Atoi(r.Form["shifts-off"][0])
			}
			if r.Form["per-shift"][0] != "" {
				database_schedule_entry.VolunteersPerShift, _ = strconv.Atoi(r.Form["per-shift"][0])
			}
			log.Printf("%v", database_schedule_entry)
		}
		log.Printf("%#v", scheduleDatabase)
		//w.Header().Set("HX-Retarget", "none") // overrides hx-target="body" from `<form id="schedule-name-form"...` in top_bar_div.gohtml
		base_page_data := prepareTemplateStructs(r.Form["schedule-name"][0], true)
		err = templates.ExecuteTemplate(w, "base_page", base_page_data)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleDeleteSchedule(w http.ResponseWriter, r *http.Request) { // change this to save schedule parameters to include volunteers and dates plus schedule-start-date through shifts-off-before-scheduled-again
	//------------------------ UPDATE THIS WHEN COPYING, DUMMY ------------------------
	handlerInfo := handlerInfoStruct{"/delete-schedule", "handleDeleteSchedule", "GET"}
	//---------------------------------------------------------------------------------
	if !requestIsValid(w, r, handlerInfo.address, handlerInfo.method) {
		log.Printf("Request to %s is invalid!", handlerInfo.funcName)
		return
	}
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	if !parametersValidated(r.Form, "schedule-selection") {
		log.Fatalf("Fatal error in %s. See error message above.", handlerInfo.address)
	}
	log.Printf("Evaluating %s from get: %v", handlerInfo.address, r.Form)
	if r.Form["schedule-selection"][0] == "new-schedule" {
		w.Header().Set("HX-Retarget", "none") // overrides hx-target="body" from `<button id="schedule-delete-btn"...` in top_bar_div.gohtml
	}
	scheduleDatabase = slices.DeleteFunc(scheduleDatabase, func(s scheduleStruct) bool {
		return reflect.DeepEqual(&s, sortedSchedules[r.Form["schedule-selection"][0]])
	}) // this simulates an API delete message
	sortedSchedules = sortSchedulesToMap(scheduleDatabase) // need to refresh sortedSchedules now
	base_page_data := prepareTemplateStructs("", false)
	err = templates.ExecuteTemplate(w, "base_page", base_page_data)
	if err != nil {
		log.Fatal(err)
	}
}

func init() { // this runs once before main(). I'm using it to parse templates once.
	// parse underlying/base templates first so the blocks show up. then overwrite the blocks as needed by parsing the other template files.
	templates = template.Must(template.ParseFiles("./assets/templates/base_page.gohtml"))
	template.Must(templates.ParseFiles("./assets/templates/top_bar_div.gohtml"))
	template.Must(templates.ParseFiles("./assets/templates/left_column_div.gohtml"))
	template.Must(templates.ParseFiles("./assets/templates/right_column_div.gohtml"))
	template.Must(templates.ParseFiles("./assets/templates/volunteer_column_form.gohtml"))
	veX_nRegex = regexp.MustCompile("^ve[0-9]+-n$")
	veX_uRegex = regexp.MustCompile("^ve[0-9]+-u$")
	sortedSchedules = sortSchedulesToMap(scheduleDatabase)
}

func initDatabase() (*sql.DB, error) { // https://github.com/mattn/go-sqlite3/blob/master/_example/simple/simple.go
	dbExists := false
	if _, err := os.Stat("./vsa.db"); err == nil {
		dbExists = true
	}
	db, err := sql.Open("sqlite3", "./vsa.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if !dbExists {
		sqlStmt := `
	create table foo (id integer not null primary key, name text);
	delete from foo;
	`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return db, err
		}
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("insert into foo(id, name) values(?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		for i := 0; i < 100; i++ {
			_, err = stmt.Exec(i, fmt.Sprintf("こんにちは世界%03d", i))
			if err != nil {
				log.Fatal(err)
			}
		}
		err = tx.Commit()
		if err != nil {
			log.Fatal(err)
		}
	}

	rows, err := db.Query("select id, name from foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := db.Prepare("select name from foo where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var name string
	err = stmt.QueryRow("3").Scan(&name)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)

	_, err = db.Exec("delete from foo")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("insert into foo(id, name) values(1, 'foo'), (2, 'bar'), (3, 'baz')")
	if err != nil {
		log.Fatal(err)
	}

	rows, err = db.Query("select id, name from foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return db, nil
}

func main() {
	db, err := initDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// initialize multiplexer
	mux := http.NewServeMux()
	// handle dynamic content
	for key, value := range handleFuncMap {
		mux.HandleFunc(key, value)
	}
	// handle static content
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./assets/css"))))
	mux.Handle("/scripts/", http.StripPrefix("/scripts/", http.FileServer(http.Dir("./assets/scripts"))))
	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./assets/images"))))
	// start server
	fmt.Printf("Starting server at port %s\n", serverAddress)
	if err := http.ListenAndServe(serverAddress, mux); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"VolunteerSchedulerApp/vsadb"
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

type Env struct {
	DBModel      vsadb.VSAModel
	LoggedInUser string
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

func createWeekdaysStruct(weekdays_slice []string) (return_val weekdaysStruct) {
	if slices.Contains(weekdays_slice, "Sunday") {
		return_val.Sunday = true
	}
	if slices.Contains(weekdays_slice, "Monday") {
		return_val.Monday = true
	}
	if slices.Contains(weekdays_slice, "Tuesday") {
		return_val.Tuesday = true
	}
	if slices.Contains(weekdays_slice, "Wednesday") {
		return_val.Wednesday = true
	}
	if slices.Contains(weekdays_slice, "Thursday") {
		return_val.Thursday = true
	}
	if slices.Contains(weekdays_slice, "Friday") {
		return_val.Friday = true
	}
	if slices.Contains(weekdays_slice, "Saturday") {
		return_val.Saturday = true
	}
	return
}

func convertWeToWeekday(weekdaysSlice []string) (result []string) {
	for _, value := range weekdaysSlice {
		if value == "Su" {
			result = append(result, "Sunday")
		}
		if value == "Mo" {
			result = append(result, "Monday")
		}
		if value == "Tu" {
			result = append(result, "Tuesday")
		}
		if value == "We" {
			result = append(result, "Wednesday")
		}
		if value == "Th" {
			result = append(result, "Thursday")
		}
		if value == "Fr" {
			result = append(result, "Friday")
		}
		if value == "Sa" {
			result = append(result, "Saturday")
		}
	}
	return
}

func (env Env) prepareTemplateStructs(scheduleName string, bIsExistingAndCopyable bool) base_pageStruct {
	scheduleNames, err := env.DBModel.SendScheduleNames(env.LoggedInUser, true)
	if err != nil {
		log.Fatalf("error in prepareTemplateStructs: %v", err)
	}
	if !slices.Contains(scheduleNames, scheduleName) {
		volunteer_entries_slice := []volunteer_entryStruct{{"0", "", []string{}}}
		right_column_data := right_columnStruct{make([]string, 6)}
		left_column_data := left_columnStruct{volunteer_entries_slice, false}
		top_bar_data := top_barStruct{env.LoggedInUser, scheduleNames, "", "", "", weekdaysStruct{}, -1, -1, bIsExistingAndCopyable}
		return base_pageStruct{top_bar_data, left_column_data, right_column_data}
	} else {
		schedule, err := env.DBModel.FetchAndSendScheduleData(env.LoggedInUser, scheduleName)
		if err != nil {
			log.Fatalf("error in prepareTemplateStructs: %v", err)
		}
		volunteerNames := getStringMapKeys(schedule.VolunteerUnavailabilityData, true)
		volunteer_entries_slice := make([]volunteer_entryStruct, 0, len(volunteerNames)+1)
		i := 0
		for index, volunteerName := range volunteerNames {
			volunteer_entries_slice = append(volunteer_entries_slice, volunteer_entryStruct{fmt.Sprint(index), volunteerName, schedule.VolunteerUnavailabilityData[volunteerName]})
			i++
		}
		volunteer_entries_slice = append(volunteer_entries_slice, volunteer_entryStruct{fmt.Sprint(len(volunteerNames)), "", []string{}}) // need a blank volunteer entry
		weeks := 0
		if schedule.EndDate != "" && schedule.StartDate != "" {
			max_date, err := time.Parse("2006-01-02", schedule.EndDate)
			if err != nil {
				log.Fatalf("error in prepareTemplateStructs: method failed to parse EndDate: %v. Value of EndDate is `%s`", err, schedule.EndDate)
			}
			min_date, err := time.Parse("2006-01-02", schedule.StartDate)
			if err != nil {
				log.Fatalf("error in prepareTemplateStructs: method failed to parse StartDate: %v. Value of StartDate is `%s`", err, schedule.StartDate)
			}
			weeks = int(math.Floor(max_date.Sub(min_date).Abs().Hours() / 24 / 7)) // (floor of the (((absolute value of (max date minus min date)) in hours) divided by 24hrs/dy divided by 7dy/wk)) converted to an int
		}
		selected_days := createWeekdaysStruct(schedule.WeekdaysForSchedule)
		right_column_data := right_columnStruct{make([]string, weeks)}
		left_column_data := left_columnStruct{volunteer_entries_slice, bIsExistingAndCopyable}
		top_bar_data := top_barStruct{"Seth", scheduleNames, scheduleName, schedule.StartDate, schedule.EndDate, selected_days, schedule.ShiftsOff, schedule.VolunteersPerShift, bIsExistingAndCopyable}
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

func (env Env) parametersValidated(form url.Values, keys_to_check ...string) error {
	// possbile keys_to_check: "schedule-selection", "schedule-name", "IdIndex" "veX-X", "min-date", "max-date", "weekday", "shifts-off", "per-shift"
	mustBeLen1 := []string{"schedule-selection", "schedule-name", "IdIndex", "min-date", "max-date", "shifts-off", "per-shift"} // veX-n must also be len 1, but that is handled later
	for _, keyToCheck := range keys_to_check {
		if slices.Contains(mustBeLen1, keyToCheck) {
			if len(form[keyToCheck]) != 1 {
				return fmt.Errorf("error in parametersValidated: \"%s\" does not have length of 1", keyToCheck)
			}
		}
		if keyToCheck == "schedule-selection" {
			scheduleKeys, err := env.DBModel.SendScheduleNames(env.LoggedInUser, false)
			if err != nil {
				return fmt.Errorf("error in parametersValidated: %w", err)
			}
			allowedValues := make([]string, 0, len(scheduleKeys)+2)
			allowedValues = append(allowedValues, scheduleKeys...)
			allowedValues = append(allowedValues, "new-schedule", "copy-current-schedule")
			if !slices.Contains(allowedValues, form[keyToCheck][0]) {
				return fmt.Errorf("error in parametersValidated: Value of \"%s\" for \"%s\" was not a known response", form[keyToCheck][0], keyToCheck)
			}
		} else if keyToCheck == "IdIndex" || keyToCheck == "shifts-off" {
			if form[keyToCheck][0] != "" {
				value, err := strconv.Atoi(form[keyToCheck][0])
				if err != nil {
					return fmt.Errorf("error in parametersValidated: \"%s\" cannot be converted to an integer: %w", keyToCheck, err)
				}
				if value < 0 {
					return fmt.Errorf("error in parametersValidated: \"%s\" is less than 0", keyToCheck)
				}
			}
		} else if keyToCheck == "schedule-name" {
			if strings.ContainsAny(form[keyToCheck][0], "\\/:*?\"<>|") {
				return fmt.Errorf("error in parametersValidated: \"%s\" contains illegal characters (\\/:*?\"<>|)", keyToCheck)
			}
		} else if keyToCheck == "veX-X" {
			for formKey, formValue := range form {
				if veX_nRegex.MatchString(formKey) {
					if len(formValue) != 1 {
						return fmt.Errorf("error in parametersValidated: \"%s\" does not have length of 1", keyToCheck)
					}
				} else if veX_uRegex.MatchString(formKey) {
					for _, stringElement := range formValue {
						if stringElement != "" {
							_, err := time.Parse("2006-01-02", stringElement)
							if err != nil {
								return fmt.Errorf("error in parametersValidated: \"%s\" value \"%s\" is not in a valid date format (YYYY-MM-DD): %w", formKey, stringElement, err)
							}
						}
					}

				}
			}

		} else if keyToCheck == "min-date" || keyToCheck == "max-date" {
			if form[keyToCheck][0] != "" {
				_, err := time.Parse("2006-01-02", form[keyToCheck][0])
				if err != nil {
					return fmt.Errorf("error in parametersValidated: \"%s\" is not in a valid date format (YYYY-MM-DD): %w", keyToCheck, err)
				}
			}
		} else if keyToCheck == "weekday" {
			if slices.ContainsFunc(form[keyToCheck], func(s string) bool { return !slices.Contains([]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}, s) }) {
				return fmt.Errorf("error in parametersValidated: \"%s\" contains non weekday values (Su, Mo, Tu, We, Th, Fr, Sa)", keyToCheck)
			}
		} else if keyToCheck == "per-shift" {
			if form[keyToCheck][0] != "" {
				value, err := strconv.Atoi(form[keyToCheck][0])
				if err != nil {
					return fmt.Errorf("error in parametersValidated: \"%s\" cannot be converted to an integer: %w", keyToCheck, err)
				}
				if value < 1 {
					return fmt.Errorf("error in parametersValidated: \"%s\" is less than 1", keyToCheck)
				}
			}
		} else {
			return fmt.Errorf("error in parametersValidated: \"%s\" is present but unchecked", keyToCheck)
		}
	}
	return nil
}

// environment handler functions

func (env *Env) handleRoot(w http.ResponseWriter, r *http.Request) {
	//------------------------ UPDATE THIS WHEN COPYING, DUMMY ------------------------
	handlerInfo := handlerInfoStruct{"/", "handleRoot", "GET"}
	//---------------------------------------------------------------------------------
	if !requestIsValid(w, r, handlerInfo.address, handlerInfo.method) {
		log.Printf("Request to %s is invalid!", handlerInfo.funcName)
		return
	}
	base_page_data := env.prepareTemplateStructs("", false)
	err := templates.ExecuteTemplate(w, "base_page", base_page_data)
	if err != nil {
		log.Fatal(err)
	}
}

func (env *Env) handleSelectSchedule(w http.ResponseWriter, r *http.Request) {
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
	if err = env.parametersValidated(r.Form, "schedule-selection", "schedule-name"); err != nil {
		log.Fatalf("Fatal error in %s: %v", handlerInfo.address, err)
	}
	log.Printf("Evaluating %s from get: %v", handlerInfo.address, r.Form)
	var base_page_data base_pageStruct
	if r.Form["schedule-selection"][0] == "new-schedule" { // case where no schedule is selected (the blank entry in the select element)
		base_page_data = env.prepareTemplateStructs("", false)
	} else if r.Form["schedule-selection"][0] == "copy-current-schedule" { // case where copying schedule
		base_page_data = env.prepareTemplateStructs(r.Form["schedule-name"][0], false)
		base_page_data.Top_bar.Current_schedule = fmt.Sprintf("Copy of %s", base_page_data.Top_bar.Current_schedule)
	} else { // case where selection is not new or copy
		base_page_data = env.prepareTemplateStructs(r.Form["schedule-selection"][0], true)
	}
	err = templates.ExecuteTemplate(w, "base_page", base_page_data)
	if err != nil {
		log.Fatal(err)
	}
}

func (env *Env) handleAddVolunteerUnavailability(w http.ResponseWriter, r *http.Request) {
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
	if err = env.parametersValidated(r.Form, "IdIndex"); err != nil {
		log.Fatalf("Fatal error in %s: %v", handlerInfo.address, err)
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

func (env *Env) handleModVolunteers(w http.ResponseWriter, r *http.Request) {
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
	if err = env.parametersValidated(r.Form, "IdIndex"); err != nil {
		log.Fatalf("Fatal error in %s: %v", handlerInfo.address, err)
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

func (env *Env) handleSaveParameters(w http.ResponseWriter, r *http.Request) { // change this to save schedule parameters to include volunteers and dates plus schedule-start-date through shifts-off-before-scheduled-again
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
	if err = env.parametersValidated(r.Form, "schedule-selection", "schedule-name"); err != nil {
		log.Fatalf("Fatal error in %s: %v", handlerInfo.address, err)
	}
	log.Printf("Evaluating %s from post: %v", handlerInfo.address, r.Form)
	if err = env.parametersValidated(r.Form, "veX-X", "min-date", "max-date", "weekday", "shifts-off", "per-shift"); err != nil {
		log.Fatalf("Fatal error in %s: %v", handlerInfo.address, err)
	}
	selected_schedule_entry := r.Form["schedule-selection"][0]
	volunteer_entries := extractVolunteers(r.Form)
	bNewSchedule := selected_schedule_entry == "new-schedule"
	toBeReceived := vsadb.SendReceiveDataStruct{}
	toBeReceived.ScheduleName = r.Form["schedule-name"][0]
	toBeReceived.VolunteerUnavailabilityData = volunteer_entries
	toBeReceived.StartDate = r.Form["min-date"][0]
	toBeReceived.EndDate = r.Form["max-date"][0]
	toBeReceived.WeekdaysForSchedule = convertWeToWeekday(r.Form["weekday"])
	if r.Form["shifts-off"][0] != "" {
		toBeReceived.ShiftsOff = mustAtoI(r.Form["shifts-off"][0])
	} else {
		toBeReceived.ShiftsOff = -1
	}
	if r.Form["per-shift"][0] != "" {
		toBeReceived.VolunteersPerShift = mustAtoI(r.Form["per-shift"][0])
	} else {
		toBeReceived.VolunteersPerShift = -1
	}
	// Add saving completed schedule stuff here once it's implemented in the web app TODO
	//log.Printf("%#v", toBeReceived)
	err = env.DBModel.RecieveAndStoreData(env.LoggedInUser, toBeReceived, bNewSchedule)
	if err != nil {
		log.Fatal(err)
	}
	base_page_data := env.prepareTemplateStructs(r.Form["schedule-name"][0], true)
	err = templates.ExecuteTemplate(w, "base_page", base_page_data)
	if err != nil {
		log.Fatal(err)
	}
}

func (env *Env) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
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
	if err = env.parametersValidated(r.Form, "schedule-selection"); err != nil {
		log.Fatalf("Fatal error in %s: %v", handlerInfo.address, err)
	}
	log.Printf("Evaluating %s from get: %v", handlerInfo.address, r.Form)
	if r.Form["schedule-selection"][0] == "new-schedule" {
		w.Header().Set("HX-Retarget", "none") // overrides hx-target="body" from `<button id="schedule-delete-btn"...` in top_bar_div.gohtml
	}
	data := vsadb.SendReceiveDataStruct{ScheduleName: r.Form["schedule-selection"][0]}
	err = env.DBModel.RecieveAndDeleteData(env.LoggedInUser, data)
	if err != nil {
		log.Fatal(err)
	}
	base_page_data := env.prepareTemplateStructs("", false)
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
}

func main() {
	dbExists := false
	if _, err := os.Stat(vsadb.DbName); err == nil {
		dbExists = true
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", vsadb.DbName))
	if err != nil {
		log.Fatal(err)
	}
	env := &Env{
		DBModel:      vsadb.VSAModel{DB: db},
		LoggedInUser: "Seth",
	}
	defer env.DBModel.DB.Close()
	if !dbExists {
		if err = env.DBModel.CreateDatabase(); err != nil {
			log.Fatalf("Crashed in main() with error: %v", err)
		}
		//vsadb.FillInSampleDB(env.LoggedInUser, env.DBModel) // FOR TESTING ONLY!!
	}
	// initialize multiplexer
	mux := http.NewServeMux()
	// handle static content
	var handleMap = map[string]string{
		"/css/":     "./assets/css",
		"/scripts/": "./assets/scripts",
		"/images/":  "./assets/images",
	}
	for key, value := range handleMap {
		mux.Handle(key, http.StripPrefix(key, http.FileServer(http.Dir(value))))
	}
	// handle dynamic content
	var handleFuncMap = map[string]func(http.ResponseWriter, *http.Request){
		"/":                   env.handleRoot,
		"/select-schedule":    env.handleSelectSchedule,
		"/add-unavailability": env.handleAddVolunteerUnavailability,
		"/mod-volunteers":     env.handleModVolunteers,
		"/save-parameters":    env.handleSaveParameters,
		"/delete-schedule":    env.handleDeleteSchedule,
	}
	for key, value := range handleFuncMap {
		mux.HandleFunc(key, value)
	}
	// start server
	fmt.Printf("Starting server at port %s\n", serverAddress)
	if err := http.ListenAndServe(serverAddress, mux); err != nil {
		log.Fatal(err)
	}
}

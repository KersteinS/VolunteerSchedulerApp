package vsadb

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const testDbName = "vsaTEST.db"

var sampleVolunteers = []volunteer{
	{
		VolunteerName: "Tim",
	},
	{
		VolunteerName: "Bill",
	},
	{
		VolunteerName: "Jack",
	},
	{
		VolunteerName: "George",
	},
	{
		VolunteerName: "Bob",
	},
	{
		VolunteerName: "Lance",
	},
	{
		VolunteerName: "Larry",
	},
}

func simulateCreatedSampleVolunteers(currentUser string) (result []volunteer) {
	for i, val := range sampleVolunteers {
		val.VolunteerID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	return
}

var updatedVolunteers = []volunteer{
	{VolunteerID: 1, VolunteerName: "Timmy"},
	{VolunteerID: 2, VolunteerName: "Bill"},
	{VolunteerID: 3, VolunteerName: "Jack"},
	{VolunteerID: 4, VolunteerName: "George"},
	{VolunteerID: 5, VolunteerName: "Bob"},
	{VolunteerID: 6, VolunteerName: "Lance"},
	{VolunteerID: 7, VolunteerName: "Larry"},
}

func simulateUpdatedSampleVolunteers(currentUser string) (result []volunteer) {
	for i, val := range updatedVolunteers {
		val.VolunteerID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	return
}

func generateSampleSchedules(vsam VSAModel) (result []schedule) {
	result = append(result, schedule{
		ScheduleName:       "test0",
		ShiftsOff:          0,
		VolunteersPerShift: 1,
		StartDate:          Must(vsam.RequestDate(date{Month: 8, Day: 1, Year: 2023})).DateID,
		EndDate:            Must(vsam.RequestDate(date{Month: 9, Day: 1, Year: 2023})).DateID,
	})
	result = append(result, schedule{
		ScheduleName:       "test1",
		ShiftsOff:          3,
		VolunteersPerShift: 3,
		StartDate:          Must(vsam.RequestDate(date{Month: 1, Day: 1, Year: 2024})).DateID,
		EndDate:            Must(vsam.RequestDate(date{Month: 3, Day: 1, Year: 2024})).DateID,
	})
	result = append(result, schedule{
		ScheduleName:       "test2",
		ShiftsOff:          3,
		VolunteersPerShift: 3,
		StartDate:          Must(vsam.RequestDate(date{Month: 3, Day: 1, Year: 2024})).DateID,
		EndDate:            Must(vsam.RequestDate(date{Month: 6, Day: 1, Year: 2024})).DateID,
	})
	result = append(result, schedule{
		ScheduleName:       "test3",
		ShiftsOff:          3,
		VolunteersPerShift: 3,
		StartDate:          Must(vsam.RequestDate(date{Month: 6, Day: 1, Year: 2024})).DateID,
		EndDate:            Must(vsam.RequestDate(date{Month: 9, Day: 1, Year: 2024})).DateID,
	})
	return
}

func simulateCreatedSampleSchedules(currentUser string, generatedSchedules []schedule) (result []schedule) {
	for i, val := range generatedSchedules {
		val.ScheduleID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	return
}

func simulateUpdatedSampleSchedules(currentUser string, generatedSchedules []schedule) (result []schedule) {
	for i, val := range generatedSchedules {
		val.ScheduleID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	result[0].ScheduleName = "test1a"
	result[0].ShiftsOff = 10
	result[0].VolunteersPerShift = 10
	result[0].StartDate = 550
	result[0].EndDate = 556
	return
}

func generateSampleWFS(currentUser string, vsam VSAModel) (result []weekdayForSchedule) {
	result = append(result, weekdayForSchedule{
		Weekday:  Must(vsam.RequestWeekday(weekday{WeekdayID: 2})).WeekdayName,
		Schedule: Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test0"})).ScheduleID,
	})
	result = append(result, weekdayForSchedule{
		Weekday:  Must(vsam.RequestWeekday(weekday{WeekdayName: "Sunday"})).WeekdayName,
		Schedule: Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
	})
	result = append(result, weekdayForSchedule{
		Weekday:  Must(vsam.RequestWeekday(weekday{WeekdayName: "Wednesday"})).WeekdayName,
		Schedule: Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
	})
	result = append(result, weekdayForSchedule{
		Weekday:  Must(vsam.RequestWeekday(weekday{WeekdayName: "Friday"})).WeekdayName,
		Schedule: Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
	})
	return
}

func simulateCreatedSampleWFS(currentUser string, generatedWFS []weekdayForSchedule) (result []weekdayForSchedule) {
	for i, val := range generatedWFS {
		val.WFSID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	return
}

func simulateUpdatedSampleWFS(currentUser string, generatedWFS []weekdayForSchedule) (result []weekdayForSchedule) {
	for i, val := range generatedWFS {
		val.WFSID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	result[0].Weekday = "Saturday"
	result[0].Schedule = 4
	return
}

func generateSampleVFS(currentUser string, vsam VSAModel) (result []volunteerForSchedule) {
	result = append(result, []volunteerForSchedule{
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Tim"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Jack"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "George"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bob"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Lance"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Larry"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Tim"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Jack"})).VolunteerID,
		},
		{
			Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
			Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "George"})).VolunteerID,
		}}...)
	return
}

func simulateCreatedSampleVFS(currentUser string, generatedVFS []volunteerForSchedule) (result []volunteerForSchedule) {
	for i, val := range generatedVFS {
		val.VFSID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	return
}

func simulateUpdatedSampleVFS(currentUser string, generatedVFS []volunteerForSchedule) (result []volunteerForSchedule) {
	for i, val := range generatedVFS {
		val.VFSID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	result[0].Schedule = 4
	result[0].Volunteer = 7
	return
}

func generateSampleUFS(currentUser string, vsam VSAModel) (result []unavailabilityForSchedule) {
	result = append(result, []unavailabilityForSchedule{
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Tim"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 1, Day: 14, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 1, Day: 21, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bob"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 5, Day: 12, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Lance"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 5, Day: 19, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Jack"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 8, Day: 11, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "George"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 8, Day: 18, Year: 2024})).DateID,
		}}...)
	return
}

func simulateCreatedSampleUFS(currentUser string, generatedUFS []unavailabilityForSchedule) (result []unavailabilityForSchedule) {
	for i, val := range generatedUFS {
		val.UFSID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	return
}

func simulateUpdatedSampleUFS(currentUser string, generatedUFS []unavailabilityForSchedule) (result []unavailabilityForSchedule) {
	for i, val := range generatedUFS {
		val.UFSID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	result[0].VolunteerForSchedule = 2
	result[0].Date = 385
	return
}

func generateSampleSVOD(currentUser string, vsam VSAModel) (result []scheduledVolunteerOnDate) {
	result = append(result, []scheduledVolunteerOnDate{
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Tim"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 1, Day: 14, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 1, Day: 21, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bob"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 5, Day: 12, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Lance"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 5, Day: 19, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "Jack"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 8, Day: 11, Year: 2024})).DateID,
		},
		{
			VolunteerForSchedule: Must(vsam.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(vsam.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
				Volunteer: Must(vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: "George"})).VolunteerID,
			})).VFSID,
			Date: Must(vsam.RequestDate(date{Month: 8, Day: 18, Year: 2024})).DateID,
		}}...)
	return
}

func simulateCreatedSampleSVOD(currentUser string, generatedSVOD []scheduledVolunteerOnDate) (result []scheduledVolunteerOnDate) {
	for i, val := range generatedSVOD {
		val.SVODID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	return
}

func simulateUpdatedSampleSVOD(currentUser string, generatedSVOD []scheduledVolunteerOnDate) (result []scheduledVolunteerOnDate) {
	for i, val := range generatedSVOD {
		val.SVODID = i + 1
		val.User = currentUser
		result = append(result, val)
	}
	result[0].VolunteerForSchedule = 2
	result[0].Date = 385
	return
}

func checkResultsSlice[Slice []Struct, Struct comparable](t *testing.T, ans Slice, want Slice, input Slice, err error) {
	if !slices.Equal(ans, want) {
		if err != nil {
			t.Errorf("got error: `%v`, want `%+v`", err, want)
		} else {
			t.Errorf("got %+v, want %+v", ans, want)
		}
	} else if err != nil {
		if strings.Contains(err.Error(), "sql.") {
			t.Errorf("got error: `%v` for input: `%+v`", err, input)
		} else {
			t.Logf("logged error: `%v` for input: `%+v`", err, input)
		}
	}
}

func checkResults[T comparable](t *testing.T, ans T, want T, input T, err error) {
	if ans != want {
		if err != nil {
			t.Errorf("got error: `%v`, want `%+v`", err, want)
		} else {
			t.Errorf("got %+v, want %+v", ans, want)
		}
	} else if err != nil {
		if strings.Contains(err.Error(), "sql.") {
			t.Errorf("got error: `%v` for input: `%+v`", err, input)
		} else {
			t.Logf("logged error: `%v` for input: `%+v`", err, input)
		}
	}
}

func checkResultsErrOnly[Slice []Struct, Struct comparable](t *testing.T, input any, err error, want Slice, checkFunc func(a string, b Slice) (Slice, error), a string, b Slice) {
	check, checkErr := checkFunc(a, b)
	if checkErr != nil {
		t.Errorf("got error while generating check: `%v`, want `%+v`", err, want)
	}
	//t.Logf("check: `%+v`", check)
	if !slices.Equal(check, want) {
		if err != nil {
			t.Errorf("got error: `%v`, want `%+v`", err, want)
		} else {
			t.Errorf("got %+v, want %+v", check, want)
		}
	} else if err != nil {
		if strings.Contains(err.Error(), "sql.") {
			t.Errorf("got error: `%v` for input: `%+v`", err, input)
		} else {
			t.Logf("logged error: `%v` for input: `%+v`", err, input)
		}
	}
}

func setUpEnvironment(t *testing.T) (*SampleEnv, func(t *testing.T)) {
	t.Log("running setUpEnvironment")
	sampleModel, teardown := setUpDatabaseModel(t)
	env := &SampleEnv{
		Sample:       sampleModel,
		LoggedInUser: "Seth",
	}
	return env, teardown
}

func setUpDatabaseModel(t *testing.T) (VSAModel, func(t *testing.T)) {
	t.Log("running setUpDatabaseModel")
	testDbPath := fmt.Sprintf("%s\\%s", t.TempDir(), testDbName)
	model, teardown := setUpDatabase(t, testDbPath)
	model.CreateDatabase()
	return model, teardown
}

func setUpDatabase(t *testing.T, testDbPath string) (VSAModel, func(t *testing.T)) {
	t.Log("running setUpDatabase")
	if _, err := os.Stat(testDbPath); err == nil { // if it finds the file, err will be nil
		suberr := os.Remove(testDbPath)
		if suberr != nil {
			t.Errorf("Error: existing testdb file was not deleted during setUpDatabase: %v", suberr)
		}
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", testDbPath))
	if err != nil {
		t.Errorf("Error opening testdb file using sql package: %v", err)
	}
	testSample := VSAModel{DB: db}
	return testSample, func(t *testing.T) {
		t.Log("running tearDownDatabase")
		if err = testSample.DB.Close(); err != nil {
			t.Errorf("Error: created testdb file was not closed during tearDownDatabase: %v", err)
		}
	}
}

func TestCreateDatabase(t *testing.T) {
	testDbPath := fmt.Sprintf("%s\\%s", t.TempDir(), testDbName)
	testSample, tearDownDatabaseModel := setUpDatabase(t, testDbPath)
	defer tearDownDatabaseModel(t)
	err := testSample.CreateDatabase()
	if err != nil {
		t.Errorf("Error when calling CreateDatabase: %v", err)
	}
	if err := testSample.DB.Close(); err != nil {
		t.Errorf("Error closing database after creation but before opening for hashing: %v", err)
	}
	if _, err := os.Stat(testDbPath); err != nil {
		t.Errorf("Error: testdb file was not created: %v", err)
	}
	f, err := os.Open(testDbPath)
	if err != nil {
		t.Errorf("Error opening testdb file to calculate its hash: %v", err)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Errorf("Error while hashing testdb file %v", err)
	}
	if hex.EncodeToString(h.Sum(nil)) != "24b43ab7681e0e1a19270304241b531016ebf62b8858f9ef387bc7b9421a0543" {
		t.Errorf("Error: test testdb file does not match stored hash value. Computed hash: %x", h.Sum(nil))
	}
	if err = f.Close(); err != nil {
		t.Errorf("Error closing database file after hashing: %v", err)
	}
}

func TestRequestWeekday(t *testing.T) {
	testSample, tearDownDatabaseModel := setUpDatabaseModel(t)
	defer tearDownDatabaseModel(t)
	var tests = []struct {
		name  string
		input weekday
		want  weekday
	}{
		{"Get Sunday", weekday{WeekdayName: "Sunday"}, weekday{WeekdayID: 1, WeekdayName: "Sunday"}},
		{"Get Monday", weekday{WeekdayName: "Monday"}, weekday{WeekdayID: 2, WeekdayName: "Monday"}},
		{"Get Tuesday", weekday{WeekdayName: "Tuesday"}, weekday{WeekdayID: 3, WeekdayName: "Tuesday"}},
		{"Get Wednesday", weekday{WeekdayName: "Wednesday"}, weekday{WeekdayID: 4, WeekdayName: "Wednesday"}},
		{"Get Thursday", weekday{WeekdayName: "Thursday"}, weekday{WeekdayID: 5, WeekdayName: "Thursday"}},
		{"Get Friday", weekday{WeekdayName: "Friday"}, weekday{WeekdayID: 6, WeekdayName: "Friday"}},
		{"Get Saturday", weekday{WeekdayName: "Saturday"}, weekday{WeekdayID: 7, WeekdayName: "Saturday"}},
		{"Test Bad WeekdayID Error", weekday{WeekdayID: 8}, weekday{}},
		{"Test Bad WeekdayNameError", weekday{WeekdayName: "Thorsday"}, weekday{}},
		{"Test Disagreeing WeekdayID and WeekdayName", weekday{WeekdayID: 1, WeekdayName: "Monday"}, weekday{}},
		{"Test Empty Input Error", weekday{}, weekday{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := testSample.RequestWeekday(tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestMonth(t *testing.T) {
	testSample, tearDownDatabaseModel := setUpDatabaseModel(t)
	defer tearDownDatabaseModel(t)
	var tests = []struct {
		name  string
		input month
		want  month
	}{
		{"Get January", month{MonthName: "January"}, month{MonthID: 1, MonthName: "January"}},
		{"Get February", month{MonthName: "February"}, month{MonthID: 2, MonthName: "February"}},
		{"Get March", month{MonthName: "March"}, month{MonthID: 3, MonthName: "March"}},
		{"Get April", month{MonthName: "April"}, month{MonthID: 4, MonthName: "April"}},
		{"Get May", month{MonthName: "May"}, month{MonthID: 5, MonthName: "May"}},
		{"Get June", month{MonthName: "June"}, month{MonthID: 6, MonthName: "June"}},
		{"Get July", month{MonthName: "July"}, month{MonthID: 7, MonthName: "July"}},
		{"Get August", month{MonthName: "August"}, month{MonthID: 8, MonthName: "August"}},
		{"Get September", month{MonthName: "September"}, month{MonthID: 9, MonthName: "September"}},
		{"Get October", month{MonthName: "October"}, month{MonthID: 10, MonthName: "October"}},
		{"Get November", month{MonthName: "November"}, month{MonthID: 11, MonthName: "November"}},
		{"Get December", month{MonthName: "December"}, month{MonthID: 12, MonthName: "December"}},
		{"Test Bad MonthID Error", month{MonthID: 13}, month{}},
		{"Test Bad MonthNameError", month{MonthName: "Febraury"}, month{}},
		{"Test Disagreeing MonthID and MonthName", month{MonthID: 1, MonthName: "December"}, month{}},
		{"Test Empty Input Error", month{}, month{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := testSample.RequestMonth(tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestDate(t *testing.T) {
	testSample, tearDownDatabaseModel := setUpDatabaseModel(t)
	defer tearDownDatabaseModel(t)
	var tests = []struct {
		name  string
		input date
		want  date
	}{
		{name: "Get 9/9/2024", input: date{Month: 9, Day: 9, Year: 2024}, want: date{DateID: 618, Month: 9, Day: 9, Year: 2024, Weekday: "Monday"}},
		{name: "Get DateID 618", input: date{DateID: 618}, want: date{DateID: 618, Month: 9, Day: 9, Year: 2024, Weekday: "Monday"}},
		{name: "Fail to get DateID 6180000", input: date{DateID: 6180000}, want: date{}},
		{name: "Get more than one Date", input: date{Month: 9}, want: date{}},
		{name: "Don't provide any date fields", input: date{}, want: date{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := testSample.RequestDate(tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestDates(t *testing.T) {
	testSample, tearDownDatabaseModel := setUpDatabaseModel(t)
	defer tearDownDatabaseModel(t)
	var tests = []struct {
		name  string
		input []date
		want  []date
	}{
		{name: "Get 9/9/2024", input: []date{{Month: 9, Day: 9, Year: 2024}}, want: []date{{DateID: 618, Month: 9, Day: 9, Year: 2024, Weekday: "Monday"}}},
		{name: "Get DateID 618", input: []date{{DateID: 618}}, want: []date{{DateID: 618, Month: 9, Day: 9, Year: 2024, Weekday: "Monday"}}},
		{name: "Fail to get DateID 6180000", input: []date{{DateID: 6180000}}, want: []date{}},
		{name: "Fail to get DateID 618 due to wrong day field", input: []date{{DateID: 618, Day: 1}}, want: []date{}},
		{name: "Provide empty slice", input: []date{}, want: []date{}},
		{name: "Get all Mondays in July 2024", input: []date{{Month: 7, Year: 2024, Weekday: "Monday"}}, want: []date{{DateID: 548, Month: 7, Day: 1, Year: 2024, Weekday: "Monday"}, {DateID: 555, Month: 7, Day: 8, Year: 2024, Weekday: "Monday"}, {DateID: 562, Month: 7, Day: 15, Year: 2024, Weekday: "Monday"}, {DateID: 569, Month: 7, Day: 22, Year: 2024, Weekday: "Monday"}, {DateID: 576, Month: 7, Day: 29, Year: 2024, Weekday: "Monday"}}},
		{name: "Get DateID 618 and DateID 619", input: []date{{DateID: 618}, {DateID: 619}}, want: []date{{DateID: 618, Month: 9, Day: 9, Year: 2024, Weekday: "Monday"}, {DateID: 619, Month: 9, Day: 10, Year: 2024, Weekday: "Tuesday"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := testSample.RequestDates(tt.input)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestCreateVolunteers(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	tests := []struct {
		name  string
		input []volunteer
		want  []volunteer
	}{
		{name: "Create volunteers from sampleVolunteers", input: sampleVolunteers, want: simulateCreatedSampleVolunteers(env.LoggedInUser)},
		{name: "Create volunteers from sampleVolunteers and one duplicate volunteer", input: append(sampleVolunteers, volunteer{VolunteerName: "Tim"}), want: simulateCreatedSampleVolunteers(env.LoggedInUser)},
		{name: "Create volunteers but provide no volunteers", input: []volunteer{}, want: simulateCreatedSampleVolunteers(env.LoggedInUser)},
		{name: "Create volunteers from sampleVolunteers but provide one empty volunteer", input: append(sampleVolunteers, volunteer{}), want: simulateCreatedSampleVolunteers(env.LoggedInUser)},
		{name: "Only provide one empty volunteer", input: []volunteer{{}}, want: simulateCreatedSampleVolunteers(env.LoggedInUser)},
		{name: "Fail to provide VolunteerName", input: []volunteer{{User: "Anyone"}}, want: simulateCreatedSampleVolunteers(env.LoggedInUser)},
		{name: "Fail by providing duplicate input", input: []volunteer{{VolunteerName: "Anyone"}, {User: "Doesn'tMatter", VolunteerName: "Anyone"}}, want: simulateCreatedSampleVolunteers(env.LoggedInUser)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CreateVolunteers(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVolunteers, env.LoggedInUser, []volunteer{})
		})
	}
}

func TestRequestVolunteer(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	err := env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	tests := []struct {
		name  string
		input volunteer
		want  volunteer
	}{
		{name: "Request Tim", input: volunteer{VolunteerName: "Tim"}, want: volunteer{VolunteerID: 1, VolunteerName: "Tim", User: env.LoggedInUser}},
		{name: "Request VolunteerID 1", input: volunteer{VolunteerID: 1}, want: volunteer{VolunteerID: 1, VolunteerName: "Tim", User: env.LoggedInUser}},
		{name: "Request volunteer with empty struct", input: volunteer{}, want: volunteer{}},
		{name: "Request volunteer with invalid VolunteerID", input: volunteer{VolunteerID: 100}, want: volunteer{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestVolunteer(env.LoggedInUser, tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestVolunteers(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	err := env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	tests := []struct {
		name  string
		input []volunteer
		want  []volunteer
	}{
		{name: "Get 1 volunteer by VolunteerName", input: []volunteer{{VolunteerName: "Tim"}}, want: []volunteer{{VolunteerID: 1, VolunteerName: "Tim", User: env.LoggedInUser}}},
		{name: "Get 1 volunteer by VolunteerID", input: []volunteer{{VolunteerID: 1}}, want: []volunteer{{VolunteerID: 1, VolunteerName: "Tim", User: env.LoggedInUser}}},
		{name: "Get 1 volunteer by VolunteerID and Volunteer Name and User", input: []volunteer{{VolunteerID: 1, VolunteerName: "Tim", User: env.LoggedInUser}}, want: []volunteer{{VolunteerID: 1, VolunteerName: "Tim", User: env.LoggedInUser}}},
		{name: "Get all volunteers", input: []volunteer{}, want: []volunteer{
			{VolunteerID: 1, VolunteerName: "Tim", User: env.LoggedInUser},
			{VolunteerID: 2, VolunteerName: "Bill", User: env.LoggedInUser},
			{VolunteerID: 3, VolunteerName: "Jack", User: env.LoggedInUser},
			{VolunteerID: 4, VolunteerName: "George", User: env.LoggedInUser},
			{VolunteerID: 5, VolunteerName: "Bob", User: env.LoggedInUser},
			{VolunteerID: 6, VolunteerName: "Lance", User: env.LoggedInUser},
			{VolunteerID: 7, VolunteerName: "Larry", User: env.LoggedInUser},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestVolunteers(env.LoggedInUser, tt.input)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestUpdateVolunteers(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	err := env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	tests := []struct {
		name  string
		input []volunteer
		want  []volunteer
	}{
		{name: "Update 1 volunteer's VolunteerName by VolunteerID", input: []volunteer{{VolunteerID: 1, VolunteerName: "Timmy"}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)},
		{name: "Fail to update 1 volunteer by providing only VolunteerID", input: []volunteer{{VolunteerID: 1}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)},
		{name: "Fail to update 1 volunteer by not providing VolunteerID", input: []volunteer{{VolunteerName: "Tim"}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)},
		{name: "Fail to update because of an empty volunteer struct", input: []volunteer{{}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)},
		{name: "Update volunteers but don't provide any volunteers", input: []volunteer{}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)},
		{name: "Fail to create a duplicate volunteer (same User and VolunteerName, different VolunteerID)", input: []volunteer{{VolunteerID: 2, VolunteerName: "Timmy"}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)},
		{name: "Try to update a nonexistent volunteer", input: []volunteer{{VolunteerID: 10, VolunteerName: "Timmy"}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)}, // the query gets no matches; so no error, but also no output
		{name: "Fail to update because it would create a duplicate Volunteer (0 existing, 2 proposed)", input: []volunteer{{VolunteerID: 2, VolunteerName: "test2a"}, {VolunteerID: 2, VolunteerName: "test2a"}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.UpdateVolunteers(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVolunteers, env.LoggedInUser, []volunteer{})
		})
	}
}

func TestDeleteVolunteers(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	err := env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	tests := []struct {
		name  string
		input []volunteer
		want  []volunteer
	}{
		{name: "Delete 1 volunteer by VolunteerID", input: []volunteer{{VolunteerID: 1}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)[1:]},
		{name: "Fail due to empty input struct", input: []volunteer{{}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)[1:]},
		{name: "Fail due to empty input slice", input: []volunteer{}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)[1:]},
		{name: "Delete 2 volunteers by name", input: []volunteer{{VolunteerName: "Bill"}, {VolunteerName: "Jack"}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)[3:]},
		{name: "Fail by not providing neither VolunteerID nor VolunteerName", input: []volunteer{{User: "Doesn'tMatter"}}, want: simulateUpdatedSampleVolunteers(env.LoggedInUser)[3:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.DeleteVolunteers(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVolunteers, env.LoggedInUser, []volunteer{})
		})
	}
}

func TestCleanOrphanedVolunteers(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	err := env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	tests := []struct {
		name  string
		input []volunteer
		want  []volunteer
	}{
		{name: "Clean Orphaned Volunteers", input: simulateCreatedSampleVolunteers(env.LoggedInUser), want: []volunteer{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CleanOrphanedVolunteers(env.LoggedInUser)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVolunteers, env.LoggedInUser, []volunteer{})
		})
	}
}

func TestCreateSchedulesExtended(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	simulatedCreatedSampleSchedules := simulateCreatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input []schedule
		want  []schedule
	}{
		{name: "Create schedules from sampleSchedules", input: generatedSampleSchedules, want: simulatedCreatedSampleSchedules},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, tt.input, true)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSchedules, env.LoggedInUser, []schedule{})
		})
	}
}

func TestCreateSchedules(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)[1:]
	simulatedCreatedSampleSchedules := simulateCreatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input []schedule
		want  []schedule
	}{
		{name: "Create schedules from sampleSchedules", input: generatedSampleSchedules, want: simulatedCreatedSampleSchedules},
		{name: "Fail to create schedules from duplicate schedule", input: []schedule{generatedSampleSchedules[0]}, want: simulatedCreatedSampleSchedules},
		{name: "Create schedules but provide no schedule structs", input: []schedule{}, want: simulatedCreatedSampleSchedules},
		{name: "Only provide one empty schedule struct", input: []schedule{{}}, want: simulatedCreatedSampleSchedules},
		{name: "Do not provide ScheduleName", input: []schedule{
			{
				ShiftsOff:          4,
				VolunteersPerShift: 4,
				StartDate:          Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
				EndDate:            Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
			}},
			want: simulatedCreatedSampleSchedules},
		{name: "Provide negative ShiftsOff", input: []schedule{
			{
				ScheduleName:       "test4",
				VolunteersPerShift: 4,
				StartDate:          Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
				EndDate:            Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
			}},
			want: simulatedCreatedSampleSchedules},
		{name: "Do not provide VolunteersPerShift", input: []schedule{
			{
				ScheduleName: "test4",
				ShiftsOff:    4,
				StartDate:    Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
				EndDate:      Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
			}},
			want: simulatedCreatedSampleSchedules},
		{name: "Do not provide StartDate", input: []schedule{
			{
				ScheduleName:       "test4",
				ShiftsOff:          4,
				VolunteersPerShift: 4,
				EndDate:            Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
			}},
			want: simulatedCreatedSampleSchedules},
		{name: "Do not provide EndDate", input: []schedule{
			{
				ScheduleName:       "test4",
				ShiftsOff:          4,
				VolunteersPerShift: 4,
				StartDate:          Must(env.Sample.RequestDate(date{Month: 4, Day: 4, Year: 2024})).DateID,
			}},
			want: simulatedCreatedSampleSchedules},
		{name: "Provide ShiftsOff = 0", input: []schedule{{ScheduleName: "test01", ShiftsOff: 0, VolunteersPerShift: 1, User: "Seth", StartDate: 213, EndDate: 244}}, want: simulatedCreatedSampleSchedules},
		{name: "Fail by providing duplicate input", input: []schedule{
			{
				ScheduleName: "test01", ShiftsOff: 3, VolunteersPerShift: 1, User: "Seth", StartDate: 213, EndDate: 244,
			},
			{
				ScheduleName: "test01", ShiftsOff: 3, VolunteersPerShift: 1, User: "Doesn'tMatter", StartDate: 213, EndDate: 244,
			}},
			want: simulatedCreatedSampleSchedules},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CreateSchedules(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSchedules, env.LoggedInUser, []schedule{})
		})
	}
}

func TestRequestSchedulesExtended(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSchedules := simulateCreatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input []schedule
		want  []schedule
	}{
		{name: "Request all schedules", input: []schedule{}, want: simulatedCreatedSampleSchedules},
		{name: "Request all schedules with User", input: []schedule{{ShiftsOff: -1, User: "Seth"}}, want: simulatedCreatedSampleSchedules},
		{name: "Request fully specified schedule", input: []schedule{{ScheduleID: 1, ScheduleName: "test0", ShiftsOff: 0, VolunteersPerShift: 1, User: "Seth", StartDate: 213, EndDate: 244}}, want: simulatedCreatedSampleSchedules[:1]},
		{name: "Fail due to empty/default struct (manually set ShiftsOff to -1)", input: []schedule{{ShiftsOff: -1}}, want: []schedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestSchedulesExtended(env.LoggedInUser, tt.input, true)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestSchedules(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)[1:]
	err := env.Sample.CreateSchedules(env.LoggedInUser, generatedSampleSchedules)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedules failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSchedules := simulateCreatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input []schedule
		want  []schedule
	}{
		{name: "Request all schedules", input: []schedule{}, want: simulatedCreatedSampleSchedules},
		{name: "Fail due to manually setting ShiftsOff to -1 when using RequestSchedule", input: []schedule{{ShiftsOff: -1}}, want: []schedule{}},
		{name: "Fail by providing empty input struct", input: []schedule{{}}, want: []schedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestSchedules(env.LoggedInUser, tt.input)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestSchedule(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)[1:]
	err := env.Sample.CreateSchedules(env.LoggedInUser, generatedSampleSchedules)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedules failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSchedules := simulateCreatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input schedule
		want  schedule
	}{
		{name: "Request schedule 1", input: schedule{ScheduleID: 1}, want: simulatedCreatedSampleSchedules[0]},
		{name: "Fail by providing empty input struct", input: schedule{}, want: schedule{}},
		{name: "Fail due to manually setting ShiftsOff to -1 when using RequestSchedule", input: schedule{ShiftsOff: -1}, want: schedule{}},
		{name: "Fail by requesting more than one schedule", input: schedule{User: "Seth"}, want: schedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestSchedule(env.LoggedInUser, tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestUpdateSchedulesExtended(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	simulatedUpdatedSampleSchedules := simulateUpdatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input []schedule
		want  []schedule
	}{
		{name: "Update 1 schedule", input: []schedule{{ScheduleID: 1, ScheduleName: "test1a", ShiftsOff: 10, VolunteersPerShift: 10, StartDate: 550, EndDate: 556}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update a schedule by providing an empty input struct", input: []schedule{{ShiftsOff: -1}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update a schedule by not providing a ScheduleID", input: []schedule{{ScheduleName: "test1", ShiftsOff: 4}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update a schedule by only providing a ScheduleID", input: []schedule{{ScheduleID: 1, ShiftsOff: -1}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update a schedule because it would create a duplicate schedule", input: []schedule{{ScheduleID: 2, ScheduleName: "test1a", ShiftsOff: 10, VolunteersPerShift: 10, StartDate: 550, EndDate: 556}}, want: simulatedUpdatedSampleSchedules},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.UpdateSchedulesExtended(env.LoggedInUser, tt.input, true)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSchedules, env.LoggedInUser, []schedule{})
		})
	}
}

func TestUpdateSchedules(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)[1:]
	err := env.Sample.CreateSchedules(env.LoggedInUser, generatedSampleSchedules)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedules failed): %v", err)
		t.FailNow()
	}
	simulatedUpdatedSampleSchedules := simulateUpdatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input []schedule
		want  []schedule
	}{
		{name: "Update 1 schedule", input: []schedule{{ScheduleID: 1, ScheduleName: "test1a", ShiftsOff: 10, VolunteersPerShift: 10, StartDate: 550, EndDate: 556}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update a schedule by providing an empty input struct", input: []schedule{{}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update a schedule by not providing a ScheduleID", input: []schedule{{ScheduleName: "test1", ShiftsOff: 4}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update a schedule by only providing a ScheduleID", input: []schedule{{ScheduleID: 1}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update because it would create a duplicate Schedule (1 existing, 1 proposed)", input: []schedule{{ScheduleID: 2, ScheduleName: "test1a", ShiftsOff: 10, VolunteersPerShift: 10, StartDate: 550, EndDate: 556}}, want: simulatedUpdatedSampleSchedules},
		{name: "Fail to update because it would create a duplicate Schedule (0 existing, 2 proposed)", input: []schedule{{ScheduleID: 2, ScheduleName: "test2a", ShiftsOff: 10, VolunteersPerShift: 10, StartDate: 550, EndDate: 556}, {ScheduleID: 3, ScheduleName: "test2a", ShiftsOff: 10, VolunteersPerShift: 10, StartDate: 550, EndDate: 556}}, want: simulatedUpdatedSampleSchedules},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.UpdateSchedules(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSchedules, env.LoggedInUser, []schedule{})
		})
	}
}

func TestDeleteSchedules(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSchedules := simulateCreatedSampleSchedules(env.LoggedInUser, generatedSampleSchedules)
	tests := []struct {
		name  string
		input []schedule
		want  []schedule
	}{
		{name: "Fail by providing an empty input struct", input: []schedule{{}}, want: simulatedCreatedSampleSchedules},
		{name: "Fail by providing an everything but ScheduleID and ScheduleName", input: []schedule{
			{
				ShiftsOff:          0,
				VolunteersPerShift: 1,
				StartDate:          Must(env.Sample.RequestDate(date{Month: 8, Day: 1, Year: 2023})).DateID,
				EndDate:            Must(env.Sample.RequestDate(date{Month: 9, Day: 1, Year: 2023})).DateID,
			}},
			want: simulatedCreatedSampleSchedules},
		{name: "Delete one schedule by ScheduleID", input: []schedule{{ScheduleID: 1}}, want: simulatedCreatedSampleSchedules[1:]},
		{name: "Delete one schedule by ScheduleName", input: []schedule{{ScheduleName: "test1"}}, want: simulatedCreatedSampleSchedules[2:]},
		{name: "Fail to delete all schedules", input: []schedule{}, want: simulatedCreatedSampleSchedules[2:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.DeleteSchedules(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSchedules, env.LoggedInUser, []schedule{})
		})
	}
}

func TestCreateWFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	generatedSampleWFS := generateSampleWFS(env.LoggedInUser, env.Sample)
	simulatedCreatedSampleWFS := simulateCreatedSampleWFS(env.LoggedInUser, generatedSampleWFS)
	tests := []struct {
		name  string
		input []weekdayForSchedule
		want  []weekdayForSchedule
	}{
		{name: "Create WFS from sampleWFS", input: generatedSampleWFS, want: simulatedCreatedSampleWFS},
		{name: "Fail to create WFS from duplicate WFS", input: []weekdayForSchedule{generatedSampleWFS[0]}, want: simulatedCreatedSampleWFS},
		{name: "Fail to create WFS by providing no wfs structs", input: []weekdayForSchedule{}, want: simulatedCreatedSampleWFS},
		{name: "Fail to create WFS by providing one empty wfs struct", input: []weekdayForSchedule{{}}, want: simulatedCreatedSampleWFS},
		{name: "Fail to create WFS by not providing a Weekday", input: []weekdayForSchedule{{Schedule: 5}}, want: simulatedCreatedSampleWFS},
		{name: "Fail to create WFS by not providing a Schedule", input: []weekdayForSchedule{{Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Thursday"})).WeekdayName}}, want: simulatedCreatedSampleWFS},
		{name: "Fail to create WFS by providing a duplicate input", input: []weekdayForSchedule{{Weekday: "Thursday", Schedule: 5}, {User: "Doesn'tMatter", Weekday: "Thursday", Schedule: 5}}, want: simulatedCreatedSampleWFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CreateWFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestWFS, env.LoggedInUser, []weekdayForSchedule{})
		})
	}
}

func TestRequestWFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	generatedSampleWFS := generateSampleWFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateWFS(env.LoggedInUser, generatedSampleWFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateWFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleWFS := simulateCreatedSampleWFS(env.LoggedInUser, generatedSampleWFS)
	tests := []struct {
		name  string
		input []weekdayForSchedule
		want  []weekdayForSchedule
	}{
		{name: "Request all WFS", input: []weekdayForSchedule{}, want: simulatedCreatedSampleWFS},
		{name: "Request a fully specified WFS", input: simulatedCreatedSampleWFS[:1], want: simulatedCreatedSampleWFS[:1]},
		{name: "Fail by requesting an empty WFS", input: []weekdayForSchedule{{}}, want: []weekdayForSchedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestWFS(env.LoggedInUser, tt.input)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestWFSSingle(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	generatedSampleWFS := generateSampleWFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateWFS(env.LoggedInUser, generatedSampleWFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateWFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleWFS := simulateCreatedSampleWFS(env.LoggedInUser, generatedSampleWFS)
	tests := []struct {
		name  string
		input weekdayForSchedule
		want  weekdayForSchedule
	}{
		{name: "Request a fully specified WFS", input: simulatedCreatedSampleWFS[1], want: simulatedCreatedSampleWFS[1]},
		{name: "Fail by requesting an empty WFS", input: weekdayForSchedule{}, want: weekdayForSchedule{}},
		{name: "Fail by requesting multiple WFS", input: weekdayForSchedule{User: env.LoggedInUser}, want: weekdayForSchedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestWFSSingle(env.LoggedInUser, tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestUpdateWFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	generatedSampleWFS := generateSampleWFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateWFS(env.LoggedInUser, generatedSampleWFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateWFS failed): %v", err)
		t.FailNow()
	}
	simulatedUpdatedSampleWFS := simulateUpdatedSampleWFS(env.LoggedInUser, generatedSampleWFS)
	tests := []struct {
		name  string
		input []weekdayForSchedule
		want  []weekdayForSchedule
	}{
		{name: "Update 1 WFS", input: []weekdayForSchedule{{WFSID: 1, Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Saturday"})).WeekdayName, Schedule: 4}}, want: simulatedUpdatedSampleWFS},
		{name: "Update 1 WFS schedule", input: []weekdayForSchedule{{WFSID: 1, Schedule: 4}}, want: simulatedUpdatedSampleWFS},
		{name: "Update 1 WFS weekday", input: []weekdayForSchedule{{WFSID: 1, Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Saturday"})).WeekdayName}}, want: simulatedUpdatedSampleWFS},
		{name: "Fail to update by only providing one value in WFS", input: []weekdayForSchedule{{WFSID: 1}}, want: simulatedUpdatedSampleWFS},
		{name: "Fail to update by not providing WFSID", input: []weekdayForSchedule{{Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Saturday"})).WeekdayName}}, want: simulatedUpdatedSampleWFS},
		{name: "Fail to update by providing an empty WFS struct", input: []weekdayForSchedule{{}}, want: simulatedUpdatedSampleWFS},
		{name: "Fail to update by providing an empty WFS slice", input: []weekdayForSchedule{}, want: simulatedUpdatedSampleWFS},
		{name: "Fail to update because it would create a duplicate WFS (1 existing, 1 proposed)", input: []weekdayForSchedule{{WFSID: 2, Schedule: 4, Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Saturday"})).WeekdayName}}, want: simulatedUpdatedSampleWFS},
		{name: "Fail to update because it would create a duplicate WFS (0 existing, 2 proposed)", input: []weekdayForSchedule{{WFSID: 2, Schedule: 4, Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Tuesday"})).WeekdayName}, {WFSID: 3, Schedule: 4, Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Tuesday"})).WeekdayName}}, want: simulatedUpdatedSampleWFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.UpdateWFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestWFS, env.LoggedInUser, []weekdayForSchedule{})
		})
	}
}

func TestDeleteWFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	generatedSampleWFS := generateSampleWFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateWFS(env.LoggedInUser, generatedSampleWFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateWFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleWFS := simulateCreatedSampleWFS(env.LoggedInUser, generatedSampleWFS)
	tests := []struct {
		name  string
		input []weekdayForSchedule
		want  []weekdayForSchedule
	}{
		{name: "Delete one WFS by WFSID", input: []weekdayForSchedule{{WFSID: 1}}, want: simulatedCreatedSampleWFS[1:]},
		{name: "Delete one WFS by Schedule and Weekday", input: []weekdayForSchedule{{Schedule: 2, Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Sunday"})).WeekdayName}}, want: simulatedCreatedSampleWFS[2:]},
		{name: "Fail to delete one WFS by WFSID", input: []weekdayForSchedule{{WFSID: 1}}, want: simulatedCreatedSampleWFS[2:]},
		{name: "Fail to delete one WFS by providing only Schedule", input: []weekdayForSchedule{{Schedule: 3}}, want: simulatedCreatedSampleWFS[2:]},
		{name: "Fail to delete by not providing any WFS structs", input: []weekdayForSchedule{}, want: simulatedCreatedSampleWFS[2:]},
		{name: "Fail to delete by providing empty WFS struct", input: []weekdayForSchedule{{}}, want: simulatedCreatedSampleWFS[2:]},
		{name: "Fail to delete by not providing Schedule nor Weekday nor WFSID", input: []weekdayForSchedule{{User: "Doesn'tMatter"}}, want: simulatedCreatedSampleWFS[2:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.DeleteWFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestWFS, env.LoggedInUser, []weekdayForSchedule{})
		})
	}
}

func TestCleanOrphanedWFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	generatedSampleWFS := generateSampleWFS(env.LoggedInUser, env.Sample)
	plusOrphanWFS := append(generatedSampleWFS, weekdayForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Weekday: Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Friday"})).WeekdayName})
	err = env.Sample.CreateWFS(env.LoggedInUser, plusOrphanWFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateWFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleWFS := simulateCreatedSampleWFS(env.LoggedInUser, generatedSampleWFS)
	tests := []struct {
		name  string
		input map[schedule][]weekday
		want  []weekdayForSchedule
	}{
		{name: "Clean Orphaned WFS", input: map[schedule][]weekday{
			Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})): {Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Sunday"}))},
			Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test2"})): {Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Wednesday"}))},
		}, want: simulatedCreatedSampleWFS},
		{name: "Fail by not providing a schedule with a ScheduleID", input: map[schedule][]weekday{
			{ScheduleName: "test1"}: {Must(env.Sample.RequestWeekday(weekday{WeekdayName: "Sunday"}))},
		}, want: simulatedCreatedSampleWFS},
		{name: "Fail by not providing a weekday with a WeekdayName", input: map[schedule][]weekday{
			Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test2"})): {{WeekdayID: 3}},
		}, want: simulatedCreatedSampleWFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CleanOrphanedWFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestWFS, env.LoggedInUser, []weekdayForSchedule{})
		})
	}
}

func TestCreateVFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	generatedSampleVFS := generateSampleVFS(env.LoggedInUser, env.Sample)
	simulatedCreatedSampleVFS := simulateCreatedSampleVFS(env.LoggedInUser, generatedSampleVFS)
	tests := []struct {
		name  string
		input []volunteerForSchedule
		want  []volunteerForSchedule
	}{
		{name: "Create VFS", input: generatedSampleVFS, want: simulatedCreatedSampleVFS},
		{name: "Fail by trying to create an existing VFS", input: []volunteerForSchedule{generatedSampleVFS[0]}, want: simulatedCreatedSampleVFS},
		{name: "Fail by providing duplicate inputs", input: []volunteerForSchedule{{Schedule: 2, Volunteer: 6}, {User: "Doesn'tMatter", Schedule: 2, Volunteer: 6}}, want: simulatedCreatedSampleVFS},
		{name: "Fail by not providing a Schedule", input: []volunteerForSchedule{{User: "Anybody", Volunteer: 6}}, want: simulatedCreatedSampleVFS},
		{name: "Fail by not providing a Volunteer", input: []volunteerForSchedule{{User: "Anybody", Schedule: 2}}, want: simulatedCreatedSampleVFS},
		{name: "Fail by providing an empty/default values VFS struct", input: []volunteerForSchedule{{}}, want: simulatedCreatedSampleVFS},
		{name: "Fail by providing no input", input: []volunteerForSchedule{}, want: simulatedCreatedSampleVFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CreateVFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVFS, env.LoggedInUser, []volunteerForSchedule{})
		})
	}
}

func TestRequestVFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	generatedSampleVFS := generateSampleVFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateVFS(env.LoggedInUser, generatedSampleVFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleVFS := simulateCreatedSampleVFS(env.LoggedInUser, generatedSampleVFS)
	tests := []struct {
		name  string
		input []volunteerForSchedule
		want  []volunteerForSchedule
	}{
		{name: "Request all VFS", input: []volunteerForSchedule{}, want: simulatedCreatedSampleVFS},
		{name: "Request a fully specified VFS", input: simulatedCreatedSampleVFS[:1], want: simulatedCreatedSampleVFS[:1]},
		{name: "Fail by requesting an empty VFS", input: []volunteerForSchedule{{}}, want: []volunteerForSchedule{}},
		{name: "Request a nonexistent VFS", input: []volunteerForSchedule{{Schedule: 100}}, want: []volunteerForSchedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestVFS(env.LoggedInUser, tt.input)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestVFSSingle(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	generatedSampleVFS := generateSampleVFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateVFS(env.LoggedInUser, generatedSampleVFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleVFS := simulateCreatedSampleVFS(env.LoggedInUser, generatedSampleVFS)
	tests := []struct {
		name  string
		input volunteerForSchedule
		want  volunteerForSchedule
	}{
		{name: "Request a fully specified VFS", input: simulatedCreatedSampleVFS[1], want: simulatedCreatedSampleVFS[1]},
		{name: "Fail by requesting an empty VFS", input: volunteerForSchedule{}, want: volunteerForSchedule{}},
		{name: "Fail by requesting an multiple VFS", input: volunteerForSchedule{User: env.LoggedInUser}, want: volunteerForSchedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestVFSSingle(env.LoggedInUser, tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestUpdateVFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	generatedSampleVFS := generateSampleVFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateVFS(env.LoggedInUser, generatedSampleVFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	simulatedUpdatedSampleVFS := simulateUpdatedSampleVFS(env.LoggedInUser, generatedSampleVFS)
	tests := []struct {
		name  string
		input []volunteerForSchedule
		want  []volunteerForSchedule
	}{
		{name: "Update 1 VFS", input: []volunteerForSchedule{{VFSID: 1, Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test3"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Larry"})).VolunteerID}}, want: simulatedUpdatedSampleVFS},
		{name: "Update 1 VFS schedule", input: []volunteerForSchedule{{VFSID: 1, Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test3"})).ScheduleID}}, want: simulatedUpdatedSampleVFS},
		{name: "Update 1 VFS weekday", input: []volunteerForSchedule{{VFSID: 1, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Larry"})).VolunteerID}}, want: simulatedUpdatedSampleVFS},
		{name: "Fail to update by only providing one value in VFS", input: []volunteerForSchedule{{VFSID: 1}}, want: simulatedUpdatedSampleVFS},
		{name: "Fail to update by not providing VFSID", input: []volunteerForSchedule{{Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Larry"})).VolunteerID}}, want: simulatedUpdatedSampleVFS},
		{name: "Fail to update by providing an empty VFS struct", input: []volunteerForSchedule{{}}, want: simulatedUpdatedSampleVFS},
		{name: "Fail to update by providing an empty VFS slice", input: []volunteerForSchedule{}, want: simulatedUpdatedSampleVFS},
		{name: "Fail to update because it would create a duplicate VFS (1 existing, 1 proposed)", input: []volunteerForSchedule{{VFSID: 3, Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test3"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Larry"})).VolunteerID}}, want: simulatedUpdatedSampleVFS},
		{name: "Fail to update because it would create a duplicate VFS (0 existing, 2 proposed)", input: []volunteerForSchedule{{VFSID: 3, Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Larry"})).VolunteerID}, {VFSID: 2, Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Larry"})).VolunteerID}}, want: simulatedUpdatedSampleVFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.UpdateVFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVFS, env.LoggedInUser, []volunteerForSchedule{})
		})
	}
}

func TestDeleteVFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	generatedSampleVFS := generateSampleVFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateVFS(env.LoggedInUser, generatedSampleVFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleVFS := simulateCreatedSampleVFS(env.LoggedInUser, generatedSampleVFS)
	tests := []struct {
		name  string
		input []volunteerForSchedule
		want  []volunteerForSchedule
	}{
		{name: "Delete one VFS by VFSID", input: []volunteerForSchedule{{VFSID: 1}}, want: simulatedCreatedSampleVFS[1:]},
		{name: "Delete one VFS by Schedule and Volunteer", input: []volunteerForSchedule{{Schedule: 2, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID}}, want: simulatedCreatedSampleVFS[2:]},
		{name: "Fail to delete one VFS by VFSID", input: []volunteerForSchedule{{VFSID: 1}}, want: simulatedCreatedSampleVFS[2:]},
		{name: "Fail to delete one VFS by providing only Schedule", input: []volunteerForSchedule{{Schedule: 3}}, want: simulatedCreatedSampleVFS[2:]},
		{name: "Fail to delete by not providing any VFS structs", input: []volunteerForSchedule{}, want: simulatedCreatedSampleVFS[2:]},
		{name: "Fail to delete by providing empty VFS struct", input: []volunteerForSchedule{{}}, want: simulatedCreatedSampleVFS[2:]},
		{name: "Fail to delete by not providing Schedule nor Volunteer nor VFSID", input: []volunteerForSchedule{{User: "Doesn'tMatter"}}, want: simulatedCreatedSampleVFS[2:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.DeleteVFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVFS, env.LoggedInUser, []volunteerForSchedule{})
		})
	}
}

func TestCleanOrphanedVFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	generatedSampleVFS := generateSampleVFS(env.LoggedInUser, env.Sample)
	plusOrphanVFS := append(generatedSampleVFS, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Larry"})).VolunteerID})
	err = env.Sample.CreateVFS(env.LoggedInUser, plusOrphanVFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleVFS := simulateCreatedSampleVFS(env.LoggedInUser, generatedSampleVFS)
	tests := []struct {
		name  string
		input map[schedule][]volunteer
		want  []volunteerForSchedule
	}{
		{name: "Clean Orphaned VFS", input: map[schedule][]volunteer{
			Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})): {
				Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Tim"})),
				Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})),
				Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Jack"})),
				Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "George"})),
			},
		}, want: simulatedCreatedSampleVFS},
		{name: "Fail by not providing a schedule with a ScheduleID", input: map[schedule][]volunteer{
			{ScheduleName: "test1"}: {Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Tim"}))},
		}, want: simulatedCreatedSampleVFS},
		{name: "Fail by not providing a Volunteer with a VolunteerName", input: map[schedule][]volunteer{
			Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test2"})): {{VolunteerID: 5}},
		}, want: simulatedCreatedSampleVFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CleanOrphanedVFS(env.LoggedInUser, tt.input, true, false)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestVFS, env.LoggedInUser, []volunteerForSchedule{})
		})
	}
}

func TestCreateUFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleUFS := generateSampleUFS(env.LoggedInUser, env.Sample)
	simulatedCreatedSampleUFS := simulateCreatedSampleUFS(env.LoggedInUser, generatedSampleUFS)
	tests := []struct {
		name  string
		input []unavailabilityForSchedule
		want  []unavailabilityForSchedule
	}{
		{name: "Create UFS", input: generatedSampleUFS, want: simulatedCreatedSampleUFS},
		{name: "Fail by trying to create an existing UFS", input: []unavailabilityForSchedule{generatedSampleUFS[0]}, want: simulatedCreatedSampleUFS},
		{name: "Fail by providing duplicate inputs", input: []unavailabilityForSchedule{{VolunteerForSchedule: 2, Date: 6}, {User: "Doesn'tMatter", VolunteerForSchedule: 2, Date: 6}}, want: simulatedCreatedSampleUFS},
		{name: "Fail by not providing a VolunteerForSchedule", input: []unavailabilityForSchedule{{User: "Anybody", Date: 6}}, want: simulatedCreatedSampleUFS},
		{name: "Fail by not providing a Date", input: []unavailabilityForSchedule{{User: "Anybody", VolunteerForSchedule: 2}}, want: simulatedCreatedSampleUFS},
		{name: "Fail by providing an empty/default values UFS struct", input: []unavailabilityForSchedule{{}}, want: simulatedCreatedSampleUFS},
		{name: "Fail by providing no input", input: []unavailabilityForSchedule{}, want: simulatedCreatedSampleUFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CreateUFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestUFS, env.LoggedInUser, []unavailabilityForSchedule{})
		})
	}
}

func TestRequestUFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleUFS := generateSampleUFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateUFS(env.LoggedInUser, generateSampleUFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateUFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleUFS := simulateCreatedSampleUFS(env.LoggedInUser, generatedSampleUFS)
	tests := []struct {
		name  string
		input []unavailabilityForSchedule
		want  []unavailabilityForSchedule
	}{
		{name: "Request all UFS", input: []unavailabilityForSchedule{}, want: simulatedCreatedSampleUFS},
		{name: "Request a fully specified UFS", input: simulatedCreatedSampleUFS[:1], want: simulatedCreatedSampleUFS[:1]},
		{name: "Fail by requesting an empty UFS", input: []unavailabilityForSchedule{{}}, want: []unavailabilityForSchedule{}},
		{name: "Request a nonexistent UFS", input: []unavailabilityForSchedule{{VolunteerForSchedule: 100}}, want: []unavailabilityForSchedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestUFS(env.LoggedInUser, tt.input)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestUFSSingle(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleUFS := generateSampleUFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateUFS(env.LoggedInUser, generateSampleUFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateUFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleUFS := simulateCreatedSampleUFS(env.LoggedInUser, generatedSampleUFS)
	tests := []struct {
		name  string
		input unavailabilityForSchedule
		want  unavailabilityForSchedule
	}{
		{name: "Request a fully specified UFS", input: simulatedCreatedSampleUFS[1], want: simulatedCreatedSampleUFS[1]},
		{name: "Fail by requesting an empty UFS", input: unavailabilityForSchedule{}, want: unavailabilityForSchedule{}},
		{name: "Fail by requesting an multiple UFS", input: unavailabilityForSchedule{User: env.LoggedInUser}, want: unavailabilityForSchedule{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestUFSSingle(env.LoggedInUser, tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestUpdateUFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleUFS := generateSampleUFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateUFS(env.LoggedInUser, generateSampleUFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateUFS failed): %v", err)
		t.FailNow()
	}
	simulatedUpdatedSampleUFS := simulateUpdatedSampleUFS(env.LoggedInUser, generatedSampleUFS)
	tests := []struct {
		name  string
		input []unavailabilityForSchedule
		want  []unavailabilityForSchedule
	}{
		{name: "Update 1 UFS", input: []unavailabilityForSchedule{
			{
				UFSID:                1,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleUFS},
		{name: "Update 1 UFS VolunteerForSchedule", input: []unavailabilityForSchedule{
			{
				UFSID:                1,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
			}}, want: simulatedUpdatedSampleUFS},
		{name: "Update 1 UFS Date", input: []unavailabilityForSchedule{
			{
				UFSID: 1,
				Date:  Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleUFS},
		{name: "Fail to update by only providing one value in UFS", input: []unavailabilityForSchedule{{UFSID: 1}}, want: simulatedUpdatedSampleUFS},
		{name: "Fail to update by not providing UFSID", input: []unavailabilityForSchedule{
			{
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleUFS},
		{name: "Fail to update by providing an empty UFS struct", input: []unavailabilityForSchedule{{}}, want: simulatedUpdatedSampleUFS},
		{name: "Fail to update by providing an empty UFS slice", input: []unavailabilityForSchedule{}, want: simulatedUpdatedSampleUFS},
		{name: "Fail to update because it would create a duplicate UFS (1 existing, 1 proposed)", input: []unavailabilityForSchedule{
			{
				UFSID:                3,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleUFS},
		{name: "Fail to update because it would create a duplicate UFS (0 existing, 2 proposed)", input: []unavailabilityForSchedule{
			{
				UFSID:                3,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 5, Day: 20, Year: 2024})).DateID,
			},
			{
				UFSID:                4,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 5, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleUFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.UpdateUFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestUFS, env.LoggedInUser, []unavailabilityForSchedule{})
		})
	}
}

func TestDeleteUFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleUFS := generateSampleUFS(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateUFS(env.LoggedInUser, generateSampleUFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateUFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleUFS := simulateCreatedSampleUFS(env.LoggedInUser, generatedSampleUFS)
	tests := []struct {
		name  string
		input []unavailabilityForSchedule
		want  []unavailabilityForSchedule
	}{
		{name: "Delete one UFS by UFSID", input: []unavailabilityForSchedule{{UFSID: 1}}, want: simulatedCreatedSampleUFS[1:]},
		{name: "Delete one UFS by VolunteerForSchedule and Date", input: []unavailabilityForSchedule{
			{
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{
					Schedule:  Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID,
					Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
				})).VFSID,
				Date: Must(env.Sample.RequestDate(date{Month: 1, Day: 21, Year: 2024})).DateID,
			}}, want: simulatedCreatedSampleUFS[2:]},
		{name: "Fail to delete one UFS by UFSID", input: []unavailabilityForSchedule{{UFSID: 1}}, want: simulatedCreatedSampleUFS[2:]},
		{name: "Fail to delete one UFS by providing only VolunteerForSchedule", input: []unavailabilityForSchedule{
			{
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{
					Schedule:  Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test2"})).ScheduleID,
					Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bob"})).VolunteerID,
				})).VFSID,
			}}, want: simulatedCreatedSampleUFS[2:]},
		{name: "Fail to delete by not providing any UFS structs", input: []unavailabilityForSchedule{}, want: simulatedCreatedSampleUFS[2:]},
		{name: "Fail to delete by providing empty UFS struct", input: []unavailabilityForSchedule{{}}, want: simulatedCreatedSampleUFS[2:]},
		{name: "Fail to delete by not providing Schedule nor Volunteer nor UFSID", input: []unavailabilityForSchedule{{User: "Doesn'tMatter"}}, want: simulatedCreatedSampleUFS[2:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.DeleteUFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestUFS, env.LoggedInUser, []unavailabilityForSchedule{})
		})
	}
}

func TestCleanOrphanedUFS(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleUFS := generateSampleUFS(env.LoggedInUser, env.Sample)
	plusOrphanUFS := append(generatedSampleUFS, unavailabilityForSchedule{VolunteerForSchedule: 2, Date: Must(env.Sample.RequestDate(date{Month: 1, Day: 28, Year: 2024})).DateID})
	err = env.Sample.CreateUFS(env.LoggedInUser, plusOrphanUFS)
	if err != nil {
		t.Errorf("Error setting up test (CreateUFS failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleUFS := simulateCreatedSampleUFS(env.LoggedInUser, generatedSampleUFS)
	tests := []struct {
		name  string
		input map[volunteerForSchedule][]date
		want  []unavailabilityForSchedule
	}{
		{name: "Clean Orphaned UFS", input: map[volunteerForSchedule][]date{
			Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})): {
				Must(env.Sample.RequestDate(date{Month: 1, Day: 21, Year: 2024})),
			},
		}, want: simulatedCreatedSampleUFS},
		{name: "Fail by not providing a VFS with a VFSID", input: map[volunteerForSchedule][]date{
			{Schedule: 1}: {Must(env.Sample.RequestDate(date{Month: 1, Day: 21, Year: 2024}))},
		}, want: simulatedCreatedSampleUFS},
		{name: "Fail by not providing a Date with a DateID", input: map[volunteerForSchedule][]date{
			Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})): {
				{Month: 1, Day: 21, Year: 2024},
			},
		}, want: simulatedCreatedSampleUFS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CleanOrphanedUFS(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestUFS, env.LoggedInUser, []unavailabilityForSchedule{})
		})
	}
}

func TestCreateSVOD(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleSVOD := generateSampleSVOD(env.LoggedInUser, env.Sample)
	simulatedCreatedSampleSVOD := simulateCreatedSampleSVOD(env.LoggedInUser, generatedSampleSVOD)
	tests := []struct {
		name  string
		input []scheduledVolunteerOnDate
		want  []scheduledVolunteerOnDate
	}{
		{name: "Create SVOD", input: generatedSampleSVOD, want: simulatedCreatedSampleSVOD},
		{name: "Fail by trying to create an existing SVOD", input: []scheduledVolunteerOnDate{generatedSampleSVOD[0]}, want: simulatedCreatedSampleSVOD},
		{name: "Fail by providing duplicate inputs", input: []scheduledVolunteerOnDate{{VolunteerForSchedule: 2, Date: 6}, {User: "Doesn'tMatter", VolunteerForSchedule: 2, Date: 6}}, want: simulatedCreatedSampleSVOD},
		{name: "Fail by not providing a VolunteerForSchedule", input: []scheduledVolunteerOnDate{{User: "Anybody", Date: 6}}, want: simulatedCreatedSampleSVOD},
		{name: "Fail by not providing a Date", input: []scheduledVolunteerOnDate{{User: "Anybody", VolunteerForSchedule: 2}}, want: simulatedCreatedSampleSVOD},
		{name: "Fail by providing an empty/default values SVOD struct", input: []scheduledVolunteerOnDate{{}}, want: simulatedCreatedSampleSVOD},
		{name: "Fail by providing no input", input: []scheduledVolunteerOnDate{}, want: simulatedCreatedSampleSVOD},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CreateSVOD(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSVOD, env.LoggedInUser, []scheduledVolunteerOnDate{})
		})
	}
}

func TestRequestSVOD(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleSVOD := generateSampleSVOD(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateSVOD(env.LoggedInUser, generateSampleSVOD(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateSVOD failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSVOD := simulateCreatedSampleSVOD(env.LoggedInUser, generatedSampleSVOD)
	tests := []struct {
		name  string
		input []scheduledVolunteerOnDate
		want  []scheduledVolunteerOnDate
	}{
		{name: "Request all SVOD", input: []scheduledVolunteerOnDate{}, want: simulatedCreatedSampleSVOD},
		{name: "Request a fully specified SVOD", input: simulatedCreatedSampleSVOD[:1], want: simulatedCreatedSampleSVOD[:1]},
		{name: "Fail by requesting an empty SVOD", input: []scheduledVolunteerOnDate{{}}, want: []scheduledVolunteerOnDate{}},
		{name: "Request a nonexistent SVOD", input: []scheduledVolunteerOnDate{{VolunteerForSchedule: 100}}, want: []scheduledVolunteerOnDate{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestSVOD(env.LoggedInUser, tt.input)
			checkResultsSlice(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestRequestSVODSingle(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleSVOD := generateSampleSVOD(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateSVOD(env.LoggedInUser, generateSampleSVOD(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateSVOD failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSVOD := simulateCreatedSampleSVOD(env.LoggedInUser, generatedSampleSVOD)
	tests := []struct {
		name  string
		input scheduledVolunteerOnDate
		want  scheduledVolunteerOnDate
	}{
		{name: "Request a fully specified SVOD", input: simulatedCreatedSampleSVOD[1], want: simulatedCreatedSampleSVOD[1]},
		{name: "Fail by requesting an empty SVOD", input: scheduledVolunteerOnDate{}, want: scheduledVolunteerOnDate{}},
		{name: "Fail by requesting an multiple SVOD", input: scheduledVolunteerOnDate{User: env.LoggedInUser}, want: scheduledVolunteerOnDate{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ans, err := env.Sample.RequestSVODSingle(env.LoggedInUser, tt.input)
			checkResults(t, ans, tt.want, tt.input, err)
		})
	}
}

func TestUpdateSVOD(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleSVOD := generateSampleSVOD(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateSVOD(env.LoggedInUser, generateSampleSVOD(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateSVOD failed): %v", err)
		t.FailNow()
	}
	simulatedUpdatedSampleSVOD := simulateUpdatedSampleSVOD(env.LoggedInUser, generatedSampleSVOD)
	tests := []struct {
		name  string
		input []scheduledVolunteerOnDate
		want  []scheduledVolunteerOnDate
	}{
		{name: "Update 1 SVOD", input: []scheduledVolunteerOnDate{
			{
				SVODID:               1,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleSVOD},
		{name: "Update 1 SVOD VolunteerForSchedule", input: []scheduledVolunteerOnDate{
			{
				SVODID:               1,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
			}}, want: simulatedUpdatedSampleSVOD},
		{name: "Update 1 SVOD Date", input: []scheduledVolunteerOnDate{
			{
				SVODID: 1,
				Date:   Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleSVOD},
		{name: "Fail to update by only providing one value in SVOD", input: []scheduledVolunteerOnDate{{SVODID: 1}}, want: simulatedUpdatedSampleSVOD},
		{name: "Fail to update by not providing SVODID", input: []scheduledVolunteerOnDate{
			{
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleSVOD},
		{name: "Fail to update by providing an empty SVOD struct", input: []scheduledVolunteerOnDate{{}}, want: simulatedUpdatedSampleSVOD},
		{name: "Fail to update by providing an empty SVOD slice", input: []scheduledVolunteerOnDate{}, want: simulatedUpdatedSampleSVOD},
		{name: "Fail to update because it would create a duplicate SVOD (1 existing, 1 proposed)", input: []scheduledVolunteerOnDate{
			{
				SVODID:               3,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 1, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleSVOD},
		{name: "Fail to update because it would create a duplicate SVOD (0 existing, 2 proposed)", input: []scheduledVolunteerOnDate{
			{
				SVODID:               3,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 5, Day: 20, Year: 2024})).DateID,
			},
			{
				SVODID:               4,
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})).VFSID,
				Date:                 Must(env.Sample.RequestDate(date{Month: 5, Day: 20, Year: 2024})).DateID,
			}}, want: simulatedUpdatedSampleSVOD},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.UpdateSVOD(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSVOD, env.LoggedInUser, []scheduledVolunteerOnDate{})
		})
	}
}

func TestDeleteSVOD(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleSVOD := generateSampleSVOD(env.LoggedInUser, env.Sample)
	err = env.Sample.CreateSVOD(env.LoggedInUser, generateSampleSVOD(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateSVOD failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSVOD := simulateCreatedSampleSVOD(env.LoggedInUser, generatedSampleSVOD)
	tests := []struct {
		name  string
		input []scheduledVolunteerOnDate
		want  []scheduledVolunteerOnDate
	}{
		{name: "Delete one SVOD by SVODID", input: []scheduledVolunteerOnDate{{SVODID: 1}}, want: simulatedCreatedSampleSVOD[1:]},
		{name: "Delete one SVOD by VolunteerForSchedule and Date", input: []scheduledVolunteerOnDate{
			{
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{
					Schedule:  Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID,
					Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
				})).VFSID,
				Date: Must(env.Sample.RequestDate(date{Month: 1, Day: 21, Year: 2024})).DateID,
			}}, want: simulatedCreatedSampleSVOD[2:]},
		{name: "Fail to delete one SVOD by SVODID", input: []scheduledVolunteerOnDate{{SVODID: 1}}, want: simulatedCreatedSampleSVOD[2:]},
		{name: "Fail to delete one SVOD by providing only VolunteerForSchedule", input: []scheduledVolunteerOnDate{
			{
				VolunteerForSchedule: Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{
					Schedule:  Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test2"})).ScheduleID,
					Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bob"})).VolunteerID,
				})).VFSID,
			}}, want: simulatedCreatedSampleSVOD[2:]},
		{name: "Fail to delete by not providing any SVOD structs", input: []scheduledVolunteerOnDate{}, want: simulatedCreatedSampleSVOD[2:]},
		{name: "Fail to delete by providing empty SVOD struct", input: []scheduledVolunteerOnDate{{}}, want: simulatedCreatedSampleSVOD[2:]},
		{name: "Fail to delete by not providing Schedule nor Volunteer nor SVODID", input: []scheduledVolunteerOnDate{{User: "Doesn'tMatter"}}, want: simulatedCreatedSampleSVOD[2:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.DeleteSVOD(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSVOD, env.LoggedInUser, []scheduledVolunteerOnDate{})
		})
	}
}

func TestCleanOrphanedSVOD(t *testing.T) {
	env, tearDownEnvironment := setUpEnvironment(t)
	defer tearDownEnvironment(t)
	generatedSampleSchedules := generateSampleSchedules(env.Sample)
	err := env.Sample.CreateSchedulesExtended(env.LoggedInUser, generatedSampleSchedules, true)
	if err != nil {
		t.Errorf("Error setting up test (CreateSchedulesExtended failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVolunteers(env.LoggedInUser, sampleVolunteers)
	if err != nil {
		t.Errorf("Error setting up test (CreateVolunteers failed): %v", err)
		t.FailNow()
	}
	err = env.Sample.CreateVFS(env.LoggedInUser, generateSampleVFS(env.LoggedInUser, env.Sample))
	if err != nil {
		t.Errorf("Error setting up test (CreateVFS failed): %v", err)
		t.FailNow()
	}
	generatedSampleSVOD := generateSampleSVOD(env.LoggedInUser, env.Sample)
	plusOrphanSVOD := append(generatedSampleSVOD, scheduledVolunteerOnDate{VolunteerForSchedule: 2, Date: Must(env.Sample.RequestDate(date{Month: 1, Day: 28, Year: 2024})).DateID})
	err = env.Sample.CreateSVOD(env.LoggedInUser, plusOrphanSVOD)
	if err != nil {
		t.Errorf("Error setting up test (CreateSVOD failed): %v", err)
		t.FailNow()
	}
	simulatedCreatedSampleSVOD := simulateCreatedSampleSVOD(env.LoggedInUser, generatedSampleSVOD)
	tests := []struct {
		name  string
		input map[volunteerForSchedule][]date
		want  []scheduledVolunteerOnDate
	}{
		{name: "Clean Orphaned SVOD", input: map[volunteerForSchedule][]date{
			Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})): {
				Must(env.Sample.RequestDate(date{Month: 1, Day: 21, Year: 2024})),
			},
		}, want: simulatedCreatedSampleSVOD},
		{name: "Fail by not providing a VFS with a VFSID", input: map[volunteerForSchedule][]date{

			{Schedule: 1}: {Must(env.Sample.RequestDate(date{Month: 1, Day: 21, Year: 2024}))},
		}, want: simulatedCreatedSampleSVOD},
		{name: "Fail by not providing a Date with a DateID", input: map[volunteerForSchedule][]date{

			Must(env.Sample.RequestVFSSingle(env.LoggedInUser, volunteerForSchedule{Schedule: Must(env.Sample.RequestSchedule(env.LoggedInUser, schedule{ScheduleName: "test1"})).ScheduleID, Volunteer: Must(env.Sample.RequestVolunteer(env.LoggedInUser, volunteer{VolunteerName: "Bill"})).VolunteerID})): {
				{Month: 1, Day: 21, Year: 2024},
			},
		}, want: simulatedCreatedSampleSVOD},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Sample.CleanOrphanedSVOD(env.LoggedInUser, tt.input)
			checkResultsErrOnly(t, tt.input, err, tt.want, env.Sample.RequestSVOD, env.LoggedInUser, []scheduledVolunteerOnDate{})
		})
	}
}

func TestMain(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		output any
	}{
		{name: "run main", input: nil, output: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Remove("./sample.db")
			main()
		})
	}
}

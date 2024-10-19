package vsadb

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const DbName = "./vsa.db"

type SampleEnv struct { //define in main module
	Sample       VSAModel //would need to reference submodule with ".", i.e. models.SampleModel.
	LoggedInUser string
}

type VSAModel struct { //define in submodule for db model
	DB *sql.DB
}

type weekday struct {
	WeekdayID   int
	WeekdayName string
}

type month struct {
	MonthID   int
	MonthName string
}

type date struct {
	DateID  int
	Month   int
	Day     int
	Year    int
	Weekday string
}

type user struct {
	UserName string
	Password []byte
}

type volunteer struct {
	VolunteerID   int
	VolunteerName string
	User          string
}

type schedule struct {
	ScheduleID         int
	ScheduleName       string
	ShiftsOff          int
	VolunteersPerShift int
	User               string
	StartDate          int
	EndDate            int
}

type weekdayForSchedule struct {
	WFSID    int
	User     string
	Weekday  string
	Schedule int
}

type volunteerForSchedule struct {
	VFSID     int
	User      string
	Schedule  int
	Volunteer int
}

type unavailabilityForSchedule struct {
	UFSID                int
	User                 string
	VolunteerForSchedule int
	Date                 int
}

type scheduledVolunteerOnDate struct {
	SVODID               int
	User                 string
	VolunteerForSchedule int
	Date                 int
}

type SendReceiveDataStruct struct {
	ScheduleName                string
	ShiftsOff                   int
	VolunteersPerShift          int
	User                        string
	StartDate                   string
	EndDate                     string
	WeekdaysForSchedule         []string
	VolunteerUnavailabilityData map[string][]string
	VolunteerScheduledData      map[string][]string
}

func (d date) ToString() string {
	return fmt.Sprintf("%d-%02d-%02d", d.Year, d.Month, d.Day)
}

func (d date) FromString(str string) (date, error) {
	parsedStr, err := time.Parse("2006-01-02", str)
	if err != nil {
		return date{}, fmt.Errorf("error in FromString: \"%s\" is not in a valid date format (YYYY-MM-DD): %w", str, err)
	}
	d.Month = int(parsedStr.Month())
	d.Day = parsedStr.Day()
	d.Year = parsedStr.Year()
	return d, nil
}

func CsvSlice(stringSlice []string, trimQuotes bool) string {
	jsonEncodedSlice, err := json.Marshal(stringSlice)
	if err != nil {
		log.Fatal(err)
	}
	if trimQuotes {
		return strings.ReplaceAll(strings.Trim(string(jsonEncodedSlice), "[]"), `"`, ``)
	}
	return strings.Trim(string(jsonEncodedSlice), "[]")
}

func countGTZero(intSlice []int) int {
	count := 0
	for _, val := range intSlice {
		if val > 0 {
			count++
		}
	}
	return count
}

func testEmpty[T comparable](sliceOfT []T, emptyT T) (bool, T) {
	for _, val := range sliceOfT {
		if val == emptyT {
			return true, val
		}
	}
	return false, emptyT
}

func Must[T any](value T, err error) T { // only to be used in main function testing code. Actual implementations need to handle the errors without crashing the program (unless the final step is to crash).
	if err != nil {
		log.Fatalf("Error from Must: %v", err)
	}
	return value
}

func (vsam VSAModel) CreateDatabase() error {
	initialTxQuery := `
	create table Weekdays (
		WeekdayID integer primary key autoincrement,
		WeekdayName text not null unique
	);
	create table Months (
		MonthID integer primary key autoincrement,
		MonthName text not null unique
	);
	create table Dates (
		DateID integer primary key autoincrement,
		Month integer not null check (Month > 0),
		Day integer not null check (Day > 0),
		Year integer not null check (Year > 0),
		Weekday text not null,
		foreign key (Month) references Months(MonthID),
		foreign key (Weekday) references Weekdays(WeekdayName)
	);
	create table Users (
		UserName text primary key,
		Password blob(64)
	) without rowid;
	create table Volunteers (
		VolunteerID integer primary key autoincrement,
		VolunteerName text not null,
		User text,
		foreign key (User) references Users(UserName)
	);
	create table Schedules (
		ScheduleID integer primary key autoincrement,
		ScheduleName text not null,
		ShiftsOff integer not null check (ShiftsOff > -1),
		VolunteersPerShift integer not null check (VolunteersPerShift > 0),
		User text,
		StartDate integer check (StartDate > 0),
		EndDate integer check (EndDate > 0),
		foreign key (User) references Users(UserName),
		foreign key (StartDate) references Dates(DateID),
		foreign key (EndDate) references Dates(DateID)
	);
	create table WeekdaysForSchedule (
		WFSID integer primary key autoincrement,
		User text,
		Weekday text,
		Schedule integer,
		foreign key (User) references Users(UserName),
		foreign key (Weekday) references Weekdays(WeekdayName),
		foreign key (Schedule) references Schedules(ScheduleID)
	);
	create table VolunteersForSchedule (
		VFSID integer primary key autoincrement,
		User text,
		Schedule integer,
		Volunteer integer,
		foreign key (User) references Users(UserName),
		foreign key (Schedule) references Schedules(ScheduleID),
		foreign key (Volunteer) references Volunteers(VolunteerID)
	);
	create table UnavailabilitiesForSchedule (
		UFSID integer primary key autoincrement,
		User text,
		VolunteerForSchedule integer,
		Date integer,
		foreign key (User) references Users(UserName),
		foreign key (VolunteerForSchedule) references VolunteersForSchedule(VFSID),
		foreign key (Date) references Dates(DateID)
	);
	create table scheduledVolunteersOnDates (
		SVODID integer primary key autoincrement,
		User text,
		VolunteerForSchedule integer,
		Date integer,
		foreign key (User) references Users(UserName),
		foreign key (VolunteerForSchedule) references VolunteersForSchedule(VFSID),
		foreign key (Date) references Dates(DateID)
	);
	`
	fillWeekdaysTxQuery := `insert into Weekdays (WeekdayName) values ("Sunday"), ("Monday"), ("Tuesday"), ("Wednesday"), ("Thursday"), ("Friday"), ("Saturday");`
	fillMonthsTxQuery := `insert into Months (MonthName) values ("January"), ("February"), ("March"), ("April"), ("May"), ("June"), ("July"), ("August"), ("September"), ("October"), ("November"), ("December");`
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CreateDatabase: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback() // this will still be executed if tx.Commit() is called, but it will return sql.ErrTxDone, which can be ignored
	_, err = tx.Exec(initialTxQuery)
	if err != nil {
		return fmt.Errorf("error in CreateDatabase: sql.Tx.Exec error: %w. Value of initialTxQuery is `%s`", err, initialTxQuery)
	}
	_, err = tx.Exec(fillWeekdaysTxQuery)
	if err != nil {
		return fmt.Errorf("error in CreateDatabase: sql.Tx.Exec error: %w. Value of fillWeekdaysTxQuery is `%s`", err, fillWeekdaysTxQuery)
	}
	_, err = tx.Exec(fillMonthsTxQuery)
	if err != nil {
		return fmt.Errorf("error in CreateDatabase: sql.Tx.Exec error: %w. Value of fillMonthsTxQuery is `%s`", err, fillMonthsTxQuery)
	}
	_, err = tx.Exec(`insert into Users (UserName) values ("Seth")`) // for testing only
	if err != nil {
		return fmt.Errorf("error in CreateDatabase: sql.Tx.Exec error: %w. This one is to create a user for testing, and should not appear in production", err)
	}
	fillDatesTableString := `insert into Dates (Month, Day, Year, Weekday) values (?, ?, ?, ?)`
	fillDatesTableStmt, err := tx.Prepare(fillDatesTableString)
	if err != nil {
		return fmt.Errorf("error in CreateDatabase: sql.Tx.Prepare error: %w. Value of fillDatesTableString is `%s`", err, fillDatesTableString)
	}
	defer fillDatesTableStmt.Close()
	initDate := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 365.25*40; i++ {
		workingDate := initDate.AddDate(0, 0, i)
		dateStruct := date{Month: int(workingDate.Month()), Day: workingDate.Day(), Year: workingDate.Year(), Weekday: fmt.Sprint(workingDate.Weekday())}
		_, err = fillDatesTableStmt.Exec(dateStruct.Month, dateStruct.Day, dateStruct.Year, dateStruct.Weekday)
		if err != nil {
			return fmt.Errorf("error in CreateDatabase: sql.Stmt.Exec error: %w. Value of dateStruct is `%+v`", err, dateStruct)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CreateDatabase: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) SendScheduleNames(currentUser string, sorted bool) (result []string, err error) {
	scheduleStructs, err := vsam.RequestSchedules(currentUser, []schedule{})
	if err != nil {
		return []string{}, fmt.Errorf("error in SendScheduleNames: %w", err)
	}
	for i := 0; i < len(scheduleStructs); i++ {
		result = append(result, scheduleStructs[i].ScheduleName)
	}
	if sorted {
		slices.Sort(result)
	}
	return result, nil
}

func (vsam VSAModel) FetchAndSendScheduleData(currentUser string, selectedSchedule string) (result SendReceiveDataStruct, err error) {
	scheduleRecord, err := vsam.RequestSchedule(currentUser, schedule{ScheduleName: selectedSchedule})
	if err != nil {
		return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendData: %w", err)
	}
	// Do the easy stuff: schedule name, shifts off, volunteers per shift, and user
	result.ScheduleName = scheduleRecord.ScheduleName
	result.ShiftsOff = scheduleRecord.ShiftsOff
	result.VolunteersPerShift = scheduleRecord.VolunteersPerShift
	result.User = scheduleRecord.User
	// Get the start date as a string
	startDate, err := vsam.RequestDate(date{DateID: scheduleRecord.StartDate})
	if err != nil {
		return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendData: %w", err)
	}
	result.StartDate = startDate.ToString()
	// Get the end date as a string
	endDate, err := vsam.RequestDate(date{DateID: scheduleRecord.EndDate})
	if err != nil {
		return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendData: %w", err)
	}
	result.EndDate = endDate.ToString()
	// Get the weekdays for schedule
	weekdaysForSchedule, err := vsam.RequestWFS(currentUser, []weekdayForSchedule{{Schedule: scheduleRecord.ScheduleID}})
	if err != nil {
		return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendScheduleData: %w", err)
	}
	for _, val := range weekdaysForSchedule {
		result.WeekdaysForSchedule = append(result.WeekdaysForSchedule, val.Weekday)
	}
	// Now for the complicated parts. Get the volunteers for schedule, then for each of those, make a map of volunteer names to a slice of volunteer unavailabilities and then a map of volunteer names to a slice of volunteer schedule dates
	volunteersForSchedule, err := vsam.RequestVFS(currentUser, []volunteerForSchedule{{Schedule: scheduleRecord.ScheduleID}})
	if err != nil {
		return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendScheduleData: %w", err)
	}
	result.VolunteerUnavailabilityData = make(map[string][]string, len(volunteersForSchedule))
	result.VolunteerScheduledData = make(map[string][]string, len(volunteersForSchedule))
	for _, vfsVal := range volunteersForSchedule {
		volunteerRecord, err := vsam.RequestVolunteer(currentUser, volunteer{VolunteerID: vfsVal.Volunteer})
		if err != nil {
			return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendScheduleData: %w", err)
		}
		// Do volunteers for schedule
		result.VolunteerUnavailabilityData[volunteerRecord.VolunteerName] = []string{}
		// Do unavailabilities for schedule
		unavailabilitiesForSchedule, err := vsam.RequestUFS(currentUser, []unavailabilityForSchedule{{VolunteerForSchedule: vfsVal.VFSID}})
		if err != nil {
			return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendScheduleData: %w", err)
		}
		for _, ufsVal := range unavailabilitiesForSchedule {
			ufsDate, err := vsam.RequestDate(date{DateID: ufsVal.Date})
			if err != nil {
				return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendScheduleData: %w", err)
			}
			result.VolunteerUnavailabilityData[volunteerRecord.VolunteerName] = append(result.VolunteerUnavailabilityData[volunteerRecord.VolunteerName], ufsDate.ToString())
		}
		// Do scheduled volunteer dates
		scheduledVolunteersOnDates, err := vsam.RequestSVOD(currentUser, []scheduledVolunteerOnDate{{VolunteerForSchedule: vfsVal.VFSID}})
		if err != nil {
			return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendScheduleData: %w", err)
		}
		for _, svodVal := range scheduledVolunteersOnDates {
			svodDate, err := vsam.RequestDate(date{DateID: svodVal.Date})
			if err != nil {
				return SendReceiveDataStruct{}, fmt.Errorf("error in FetchAndSendScheduleData: %w", err)
			}
			result.VolunteerScheduledData[volunteerRecord.VolunteerName] = append(result.VolunteerScheduledData[volunteerRecord.VolunteerName], svodDate.ToString())
		}
	}
	return result, nil
}

func (vsam VSAModel) RecieveAndStoreData(currentUser string, data SendReceiveDataStruct, bNewSchedule bool) (err error) {
	bDidWrite := false
	scheduleRecord := schedule{ScheduleName: data.ScheduleName}
	if !bNewSchedule {
		scheduleRecord, err = vsam.RequestSchedule(currentUser, scheduleRecord)
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
	}
	// Ensure ShiftsOff was set then write it to scheduleRecord
	if data.ShiftsOff > -1 && scheduleRecord.ShiftsOff != data.ShiftsOff {
		scheduleRecord.ShiftsOff = data.ShiftsOff
		bDidWrite = true
	}
	// Ensure VolunteersPerShift was set then write it to scheduleRecord
	if data.VolunteersPerShift > -1 && scheduleRecord.VolunteersPerShift != data.VolunteersPerShift {
		scheduleRecord.VolunteersPerShift = data.VolunteersPerShift
		bDidWrite = true
	}
	// Validate StartDate then write it to scheduleRecord
	startDate, err := date{}.FromString(data.StartDate)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	startDate, err = vsam.RequestDate(startDate)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	if scheduleRecord.StartDate != startDate.DateID {
		scheduleRecord.StartDate = startDate.DateID
		bDidWrite = true
	}
	// Validate EndDate then write it to scheduleRecord
	endDate, err := date{}.FromString(data.EndDate)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	endDate, err = vsam.RequestDate(endDate)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	if scheduleRecord.EndDate != endDate.DateID {
		scheduleRecord.EndDate = endDate.DateID
		bDidWrite = true
	}
	bZeroShiftsOff := data.ShiftsOff == 0
	if bNewSchedule {
		err = vsam.CreateSchedulesExtended(currentUser, []schedule{scheduleRecord}, bZeroShiftsOff)
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
	} else if bDidWrite {
		err = vsam.UpdateSchedulesExtended(currentUser, []schedule{scheduleRecord}, bZeroShiftsOff)
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
	}
	scheduleRecord, err = vsam.RequestSchedule(currentUser, scheduleRecord)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	// Validate WeekdaysForSchedule weekday names and whether there's already a WFS in the database for it then add it to wfsToCreate
	wfsToCreate := []weekdayForSchedule{}
	for _, val := range data.WeekdaysForSchedule {
		// data.WeekdaysForSchedule is a []string of full weekday names
		weekdayStruct, err := vsam.RequestWeekday(weekday{WeekdayName: val})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		wfsStruct := weekdayForSchedule{Schedule: scheduleRecord.ScheduleID, Weekday: weekdayStruct.WeekdayName}
		wfsSlice, err := vsam.RequestWFS(currentUser, []weekdayForSchedule{wfsStruct})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		if len(wfsSlice) == 0 {
			wfsToCreate = append(wfsToCreate, wfsStruct)
		}
	}
	if len(wfsToCreate) > 0 {
		err = vsam.CreateWFS(currentUser, wfsToCreate)
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
	}
	volunteersToCreate := []volunteer{}
	for key := range data.VolunteerUnavailabilityData {
		volunteerStruct := volunteer{VolunteerName: key}
		volunteerSlice, err := vsam.RequestVolunteers(currentUser, []volunteer{volunteerStruct})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		if len(volunteerSlice) == 0 {
			volunteersToCreate = append(volunteersToCreate, volunteerStruct)
		}
	}
	if len(volunteersToCreate) > 0 {
		vsam.CreateVolunteers(currentUser, volunteersToCreate)
	}
	vfsToCreate := []volunteerForSchedule{}
	for key := range data.VolunteerUnavailabilityData {
		volunteerRecord, err := vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: key})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		vfsStruct := volunteerForSchedule{Schedule: scheduleRecord.ScheduleID, Volunteer: volunteerRecord.VolunteerID}
		vfsSlice, err := vsam.RequestVFS(currentUser, []volunteerForSchedule{vfsStruct})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		if len(vfsSlice) == 0 {
			vfsToCreate = append(vfsToCreate, vfsStruct)
		}
	}
	if len(vfsToCreate) > 0 {
		err = vsam.CreateVFS(currentUser, vfsToCreate)
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
	}
	ufsToCreate := []unavailabilityForSchedule{}
	for key, value := range data.VolunteerUnavailabilityData {
		volunteerRecord, err := vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: key})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		vfsStruct, err := vsam.RequestVFSSingle(currentUser, volunteerForSchedule{Schedule: scheduleRecord.ScheduleID, Volunteer: volunteerRecord.VolunteerID})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		for _, dateString := range value {
			dateStruct, err := date{}.FromString(dateString)
			if err != nil {
				return fmt.Errorf("error in RecieveAndStoreData: %w", err)
			}
			dateStruct, err = vsam.RequestDate(dateStruct)
			if err != nil {
				return fmt.Errorf("error in RecieveAndStoreData: %w", err)
			}
			if dateStruct.DateID < 1 {
				return fmt.Errorf("error in RecieveAndStoreData: date provided for UFS does not exist in database: `%s`", dateString)
			}
			ufsStruct := unavailabilityForSchedule{VolunteerForSchedule: vfsStruct.VFSID, Date: dateStruct.DateID}
			ufsSlice, err := vsam.RequestUFS(currentUser, []unavailabilityForSchedule{ufsStruct})
			if err != nil {
				return fmt.Errorf("error in RecieveAndStoreData: %w", err)
			}
			if len(ufsSlice) == 0 {
				ufsToCreate = append(ufsToCreate, ufsStruct)
			}
		}
	}
	if len(ufsToCreate) > 0 {
		err = vsam.CreateUFS(currentUser, ufsToCreate)
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
	}
	svodToCreate := []scheduledVolunteerOnDate{}
	for key, value := range data.VolunteerScheduledData {
		volunteerRecord, err := vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: key})
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		vfsStruct, err := vsam.RequestVFSSingle(currentUser, volunteerForSchedule{Schedule: scheduleRecord.ScheduleID, Volunteer: volunteerRecord.VolunteerID})
		if err != nil { // I want this to error if somehow we are trying to create an SVOD for a Volunteer without a VFS, because that should have been taken care of already (at the latest by the call to CreateVFS above).
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
		for _, dateString := range value {
			dateStruct, err := date{}.FromString(dateString)
			if err != nil {
				return fmt.Errorf("error in RecieveAndStoreData: %w", err)
			}
			dateStruct, err = vsam.RequestDate(dateStruct)
			if err != nil {
				return fmt.Errorf("error in RecieveAndStoreData: %w", err)
			}
			if dateStruct.DateID < 1 {
				return fmt.Errorf("error in RecieveAndStoreData: date provided for SVOD does not exist in database: `%s`", dateString)
			}
			svodToCreate = append(svodToCreate, scheduledVolunteerOnDate{VolunteerForSchedule: vfsStruct.VFSID, Date: dateStruct.DateID})
		}
	}
	if len(svodToCreate) > 0 {
		err = vsam.CreateSVOD(currentUser, svodToCreate)
		if err != nil {
			return fmt.Errorf("error in RecieveAndStoreData: %w", err)
		}
	}
	// CleanOrphanedVolunteers is easy, but affects all schedules
	err = vsam.CleanOrphanedVolunteers(currentUser)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	// Clean orphans related to the schedule specified by data
	err = vsam.CleanOrphansForSchedule(currentUser, data)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	return nil
}

func (vsam VSAModel) RecieveAndDeleteData(currentUser string, data SendReceiveDataStruct) error {
	scheduleRecord, err := vsam.RequestSchedule(currentUser, schedule{ScheduleName: data.ScheduleName})
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	// need to delete the UFS, VFS, WFS, SVODs, and unused volunteers for the provided schedule
	err = vsam.CleanOrphanedVFS(currentUser, map[schedule][]volunteer{scheduleRecord: {}}, true, true)
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	err = vsam.CleanOrphanedWFS(currentUser, map[schedule][]weekday{scheduleRecord: {}})
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	err = vsam.DeleteSchedules(currentUser, []schedule{scheduleRecord})
	if err != nil {
		return fmt.Errorf("error in RecieveAndStoreData: %w", err)
	}
	return nil
}

func (vsam VSAModel) CleanOrphansForSchedule(currentUser string, data SendReceiveDataStruct) error {
	scheduleRecord, err := vsam.RequestSchedule(currentUser, schedule{ScheduleName: data.ScheduleName})
	if err != nil {
		return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
	}
	if scheduleRecord.ScheduleID < 1 {
		return fmt.Errorf("error in CleanOrphansForSchedule: could not locate schedule. Value of data is `%#v`", data)
	}
	// Clean orphaned WFS
	correctWeekdays := []weekday{}
	for _, value := range data.WeekdaysForSchedule {
		wd, err := vsam.RequestWeekday(weekday{WeekdayName: value})
		if err != nil {
			return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
		}
		correctWeekdays = append(correctWeekdays, wd)
	}
	err = vsam.CleanOrphanedWFS(currentUser, map[schedule][]weekday{scheduleRecord: correctWeekdays})
	if err != nil {
		return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
	}
	// Clean orphaned VFS (which optionally does delete UFS for VFS that are going to be cleaned) then clean UFS
	correctVolunteers := []volunteer{}
	correctUFS := map[volunteerForSchedule][]date{}
	for key, value := range data.VolunteerUnavailabilityData {
		v, err := vsam.RequestVolunteer(currentUser, volunteer{VolunteerName: key})
		if err != nil {
			return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
		}
		correctVolunteers = append(correctVolunteers, v)
		vfs, err := vsam.RequestVFSSingle(currentUser, volunteerForSchedule{Volunteer: v.VolunteerID, Schedule: scheduleRecord.ScheduleID})
		if err != nil {
			return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
		}
		for _, dateString := range value {
			dateStruct, err := date{}.FromString(dateString)
			if err != nil {
				return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
			}
			dateStruct, err = vsam.RequestDate(dateStruct)
			if err != nil {
				return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
			}
			correctUFS[vfs] = append(correctUFS[vfs], dateStruct)
		}
	}
	err = vsam.CleanOrphanedVFS(currentUser, map[schedule][]volunteer{scheduleRecord: correctVolunteers}, true, false)
	if err != nil {
		return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
	}
	err = vsam.CleanOrphanedUFS(currentUser, correctUFS)
	if err != nil {
		return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
	}
	// Clean orphaned SVOD
	correctSVOD := map[volunteerForSchedule][]date{}
	// TODO Implement this
	err = vsam.CleanOrphanedSVOD(currentUser, correctSVOD)
	if err != nil {
		return fmt.Errorf("error in CleanOrphansForSchedule: %w", err)
	}
	return nil
}

// This function exists to validate WeekdayName spelling and provide WeekdayID if needed. There is no RequestWeekdays method
func (vsam VSAModel) RequestWeekday(weekdayStruct weekday) (weekday, error) {
	if weekdayStruct == (weekday{}) {
		return weekday{}, errors.New("error in RequestWeekday: method failed because all of the values in weekdayStruct had an empty/default values")
	}
	var weekdays []weekday
	weekdayQuery := fmt.Sprintf(`select * from Weekdays where WeekdayID=%d or WeekdayName="%s"`, weekdayStruct.WeekdayID, weekdayStruct.WeekdayName)
	rows, err := vsam.DB.Query(weekdayQuery)
	if err != nil {
		return weekday{}, fmt.Errorf("error in RequestWeekday: sql.DB.Query error: %w. Value of weekdayQuery is `%s`", err, weekdayQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var weekdayStruct weekday
		err = rows.Scan(&weekdayStruct.WeekdayID, &weekdayStruct.WeekdayName)
		if err != nil {
			return weekday{}, fmt.Errorf("error in RequestWeekday: sql.Rows.Scan error: %w. Value of weekdayStruct is `%+v`", err, weekdayStruct)
		}
		weekdays = append(weekdays, weekdayStruct)
	}
	err = rows.Err()
	if err != nil {
		return weekday{}, fmt.Errorf("error in RequestWeekday: sql.Rows.Err error: %w", err)
	}
	if len(weekdays) != 1 {
		return weekday{}, fmt.Errorf("error in RequestWeekday: method failed to locate exactly one weekday matching `%+v`. Found %d matches", weekdayStruct, len(weekdays))
	}
	return weekdays[0], nil
}

// This function exists to validate MonthName spelling and provide MonthID if needed. There is no RequestMonths method
func (vsam VSAModel) RequestMonth(monthStruct month) (month, error) {
	if monthStruct == (month{}) {
		return month{}, errors.New("error in RequestMonth: method failed because all of the values in monthStruct had an empty/default values")
	}
	var months []month
	monthQuery := fmt.Sprintf(`select * from Months where MonthID=%d or MonthName="%s"`, monthStruct.MonthID, monthStruct.MonthName)
	rows, err := vsam.DB.Query(monthQuery)
	if err != nil {
		return month{}, fmt.Errorf("error in RequestMonth: sql.DB.Query error: %w. Value of monthQuery is `%s`", err, monthQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var monthStruct month
		err = rows.Scan(&monthStruct.MonthID, &monthStruct.MonthName)
		if err != nil {
			return month{}, fmt.Errorf("error in RequestMonth: sql.Rows.Scan error: %w. Value of monthStruct is `%+v`", err, monthStruct)
		}
		months = append(months, monthStruct)
	}
	err = rows.Err()
	if err != nil {
		return month{}, fmt.Errorf("error in RequestMonth: sql.Rows.Err error: %w", err)
	}
	if len(months) != 1 {
		return month{}, fmt.Errorf("error in RequestMonth: method failed to locate exactly one month matching `%+v`. Found %d matches", monthStruct, len(months))
	}
	return months[0], nil
}

func (vsam VSAModel) RequestDate(dateStruct date) (date, error) {
	dates, err := vsam.RequestDates([]date{dateStruct})
	if err != nil {
		return date{}, fmt.Errorf("error in RequestDate %+v: %w", dateStruct, err)
	}
	if len(dates) != 1 {
		return date{}, fmt.Errorf("error in RequestDate %+v: Failed to locate exactly one date matching dateStruct. Found %d matches", dateStruct, len(dates))
	}
	return dates[0], nil
}

func (vsam VSAModel) RequestDates(dates []date) ([]date, error) {
	dateQuery := `select * from Dates`
	if len(dates) > 0 {
		if check, failed := testEmpty(dates, date{}); check {
			return []date{}, fmt.Errorf("error in RequestDates: method failed because one of the values in dates had an empty/default values date struct: %+v", failed)
		}
		dateQuery = fmt.Sprintf(`%s where (`, dateQuery)
	} else {
		return []date{}, errors.New("error in RequestDates: method failed because the dates argument was an empty slice. At least one date must be requested")
	}
	for i := 0; i < len(dates); i++ {
		count := countGTZero([]int{dates[i].DateID, dates[i].Month, dates[i].Day, dates[i].Year, len(dates[i].Weekday)})
		// count must be at least 1 because the testEmpty check passed
		//fmt.Println(count)
		dateQuery = fmt.Sprintf(`%s(`, dateQuery)
		if dates[i].DateID > 0 {
			dateQuery = fmt.Sprintf(`%sDateID = %d`, dateQuery, dates[i].DateID)
			count--
			if count > 0 {
				dateQuery = fmt.Sprintf(`%s and `, dateQuery)
			}
		}
		if dates[i].Month > 0 {
			dateQuery = fmt.Sprintf(`%sMonth = %d`, dateQuery, dates[i].Month)
			count--
			if count > 0 {
				dateQuery = fmt.Sprintf(`%s and `, dateQuery)
			}
		}
		if dates[i].Day > 0 {
			dateQuery = fmt.Sprintf(`%sDay = %d`, dateQuery, dates[i].Day)
			count--
			if count > 0 {
				dateQuery = fmt.Sprintf(`%s and `, dateQuery)
			}
		}
		if dates[i].Year > 0 {
			dateQuery = fmt.Sprintf(`%sYear = %d`, dateQuery, dates[i].Year)
			count--
			if count > 0 {
				dateQuery = fmt.Sprintf(`%s and `, dateQuery)
			}
		}
		if len(dates[i].Weekday) > 0 {
			dateQuery = fmt.Sprintf(`%sWeekday = "%s"`, dateQuery, dates[i].Weekday)
		}
		dateQuery = fmt.Sprintf(`%s)`, dateQuery)
		if i+1 < len(dates) {
			dateQuery = fmt.Sprintf(`%s or `, dateQuery)
		}
		//fmt.Println(count)
		//fmt.Println(dateQuery)
	}
	if len(dates) > 0 {
		dateQuery = fmt.Sprintf(`%s)`, dateQuery)
	}
	//fmt.Println(dateQuery)
	var result []date
	rows, err := vsam.DB.Query(dateQuery)
	if err != nil {
		return []date{}, fmt.Errorf("error in RequestDates: sql.DB.Query error: %w. Value of dateQuery is `%s`", err, dateQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var dateStruct date
		err = rows.Scan(&dateStruct.DateID, &dateStruct.Month, &dateStruct.Day, &dateStruct.Year, &dateStruct.Weekday)
		if err != nil {
			return []date{}, fmt.Errorf("error in RequestDates: sql.Rows.Scan error: %w. Value of dateStruct is `%+v`", err, dateStruct)
		}
		result = append(result, dateStruct)
	}
	err = rows.Err()
	if err != nil {
		return []date{}, fmt.Errorf("error in RequestDates: sql.Rows.Err error: %w", err)
	}
	return result, nil
}

func (vsam VSAModel) CreateVolunteers(currentUser string, toCreate []volunteer) error {
	check, err := vsam.RequestVolunteers(currentUser, toCreate)
	if err != nil {
		return fmt.Errorf("error in CreateVolunteers: %w", err)
	}
	if len(check) > 0 {
		return fmt.Errorf("error in CreateVolunteers: method failed because at least one of the volunteers to be created already exists in the database. Existing volunteer(s): %+v", check)
	}
	checkDuplicates := []volunteer{}
	for _, val := range toCreate { // User and VolunteerID do not need to be provided in the volunteer structs
		if val.VolunteerName == (volunteer{}.VolunteerName) {
			return fmt.Errorf("error in CreateVolunteers: method failed because at least one of the volunteer structs in toCreate did not have a value for VolunteerName: %+v", val)
		}
		if !slices.Contains(checkDuplicates, volunteer{VolunteerName: val.VolunteerName}) {
			checkDuplicates = append(checkDuplicates, volunteer{VolunteerName: val.VolunteerName})
		} else {
			return fmt.Errorf("error in CreateVolunteers: method failed because at least one of the volunteer structs in toCreate was a duplicate of another volunteer struct in toCreate: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CreateVolunteers: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	fillVolunteersTableString := `insert into Volunteers (VolunteerName, User) values (?, ?)`
	fillVolunteersTableStmt, err := tx.Prepare(fillVolunteersTableString)
	if err != nil {
		return fmt.Errorf("error in CreateVolunteers: sql.Tx.Prepare error: %w. Value of fillVolunteersTableString is `%s`", err, fillVolunteersTableString)
	}
	defer fillVolunteersTableStmt.Close()
	for i := 0; i < len(toCreate); i++ {
		_, err = fillVolunteersTableStmt.Exec(toCreate[i].VolunteerName, currentUser)
		if err != nil {
			return fmt.Errorf("error in CreateVolunteers: sql.Stmt.Exec error: %w. toCreate[i] is `%+v`", err, toCreate[i])
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CreateVolunteers: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) RequestVolunteer(currentUser string, volunteerStruct volunteer) (volunteer, error) {
	volunteers, err := vsam.RequestVolunteers(currentUser, []volunteer{volunteerStruct})
	if err != nil {
		return volunteer{}, fmt.Errorf("error in RequestVolunteer %+v: %w", volunteerStruct, err)
	}
	if len(volunteers) != 1 {
		return volunteer{}, fmt.Errorf("error in RequestVolunteer %+v. Failed to locate exactly one volunteer matching volunteerStruct. Found %d matches", volunteerStruct, len(volunteers))
	}
	return volunteers[0], nil
}

func (vsam VSAModel) RequestVolunteers(currentUser string, volunteers []volunteer) ([]volunteer, error) {
	volunteersQuery := fmt.Sprintf(`select * from Volunteers where User = "%s"`, currentUser)
	if len(volunteers) > 0 {
		if check, failed := testEmpty(volunteers, volunteer{}); check {
			return []volunteer{}, fmt.Errorf("error in RequestVolunteers: method failed because one of the values in volunteers had an empty/default values volunteer struct: %+v", failed)
		}
		volunteersQuery = fmt.Sprintf(`%s and (`, volunteersQuery)
	}
	for i := 0; i < len(volunteers); i++ {
		count := countGTZero([]int{volunteers[i].VolunteerID, len(volunteers[i].VolunteerName), len(volunteers[i].User)})
		// count must be at least 1 because the testEmpty check passed
		//fmt.Println(count)
		volunteersQuery = fmt.Sprintf(`%s(`, volunteersQuery)
		if volunteers[i].VolunteerID > 0 {
			volunteersQuery = fmt.Sprintf(`%sVolunteerID = %d`, volunteersQuery, volunteers[i].VolunteerID)
			count--
			if count > 0 {
				volunteersQuery = fmt.Sprintf(`%s and `, volunteersQuery)
			}
		}
		if len(volunteers[i].VolunteerName) > 0 {
			volunteersQuery = fmt.Sprintf(`%sVolunteerName = "%s"`, volunteersQuery, volunteers[i].VolunteerName)
			count--
			if count > 0 {
				volunteersQuery = fmt.Sprintf(`%s and `, volunteersQuery)
			}
		}
		if len(volunteers[i].User) > 0 {
			volunteersQuery = fmt.Sprintf(`%sUser = "%s"`, volunteersQuery, volunteers[i].User)
		}
		volunteersQuery = fmt.Sprintf(`%s)`, volunteersQuery)
		if i+1 < len(volunteers) {
			volunteersQuery = fmt.Sprintf(`%s or `, volunteersQuery)
		}
		//fmt.Println(count)
		//fmt.Println(volunteersQuery)
	}
	if len(volunteers) > 0 {
		volunteersQuery = fmt.Sprintf(`%s)`, volunteersQuery)
	}
	//fmt.Println(volunteersQuery)
	var result []volunteer
	rows, err := vsam.DB.Query(volunteersQuery)
	if err != nil {
		return []volunteer{}, fmt.Errorf("error in RequestVolunteers: sql.DB.Query error: %w. Value of volunteersQuery is `%s`", err, volunteersQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var volunteerStruct volunteer
		err = rows.Scan(&volunteerStruct.VolunteerID, &volunteerStruct.VolunteerName, &volunteerStruct.User)
		if err != nil {
			return []volunteer{}, fmt.Errorf("error in RequestVolunteers: sql.Rows.Scan error: %w. Value of volunteerStruct is `%+v`", err, volunteerStruct)
		}
		result = append(result, volunteerStruct)
	}
	err = rows.Err()
	if err != nil {
		return []volunteer{}, fmt.Errorf("error in RequestVolunteers: sql.Rows.Err error: %w", err)
	}
	return result, nil
}

func (vsam VSAModel) UpdateVolunteers(currentUser string, toUpdate []volunteer) error {
	if check, failed := testEmpty(toUpdate, volunteer{}); check {
		return fmt.Errorf("error in UpdateVolunteers: method failed because one of the values in toUpdate had an empty/default values volunteer struct: %+v", failed)
	}
	checkDuplicates := []volunteer{}
	for _, val := range toUpdate {
		if val.VolunteerID == (volunteer{}.VolunteerID) { // User does not need to be provided in the volunteer struct
			return fmt.Errorf("error in UpdateVolunteers: method failed because one of the volunteer structs in toUpdate had an empty/default value for VolunteerID: %+v", val)
		} else if val.VolunteerName == (volunteer{}.VolunteerName) {
			return fmt.Errorf("error in UpdateVolunteers: method failed because one of the volunteer structs in toUpdate had an empty/default value for VolunteerName: %+v", val)
		} else if check, err := vsam.RequestVolunteers(currentUser, []volunteer{{VolunteerName: val.VolunteerName, User: currentUser}}); len(check) > 0 {
			if err != nil {
				return fmt.Errorf("error in UpdateVolunteers `%+v`: %w", val, err)
			}
			return fmt.Errorf("error in UpdateVolunteers: method failed because one of the volunteer structs in toUpdate would create a duplicate volunteer (each volunteer name must be unique per user): %+v", val)
		}
		if !slices.Contains(checkDuplicates, volunteer{VolunteerName: val.VolunteerName}) {
			checkDuplicates = append(checkDuplicates, volunteer{VolunteerName: val.VolunteerName})
		} else {
			return fmt.Errorf("error in UpdateVolunteers: method failed because at least two of the volunteer structs in toUpdate would create duplicate volunteer structs in the database: %+v", volunteer{VolunteerName: val.VolunteerName})
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in UpdateVolunteers: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	updateVolunteersString := fmt.Sprintf(`update Volunteers set VolunteerName=? where User="%s" and VolunteerID=?`, currentUser)
	updateVolunteersStmt, err := tx.Prepare(updateVolunteersString)
	if err != nil {
		return fmt.Errorf("error in UpdateVolunteers: sql.Tx.Prepare error: %w. value of updateVolunteersString is `%s`", err, updateVolunteersString)
	}
	defer updateVolunteersStmt.Close()
	for _, val := range toUpdate {
		_, err = updateVolunteersStmt.Exec(val.VolunteerName, val.VolunteerID)
		if err != nil {
			return fmt.Errorf("error in UpdateVolunteers: sql.Stmt.Exec error: %w. Value of val is `%+v`", err, val)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in UpdateVolunteers: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// Will delete Volunteers database entries that match the VolunteerID or that match the VolunteerName provided in each volunteer struct. If a VolunteerID > 0 is provided, the value for VolunteerName is ignored for that volunteer struct.
func (vsam VSAModel) DeleteVolunteers(currentUser string, toDelete []volunteer) error {
	if check, failed := testEmpty(toDelete, volunteer{}); check {
		return fmt.Errorf("error in DeleteVolunteers: method failed because one of the values in toDelete had an empty/default values volunteer struct: %+v", failed)
	}
	for _, val := range toDelete {
		if val.VolunteerID == (volunteer{}.VolunteerID) && val.VolunteerName == (volunteer{}.VolunteerName) { // User does not need to be provided in the volunteer struct. One of VolunteerID and VolunteerName must be provided
			return fmt.Errorf("error in DeleteVolunteers: method failed because one of the volunteer structs in toDelete had empty/default values for VolunteerID and VolunteerName (at least one must be provided): %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in DeleteVolunteers: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	for _, val := range toDelete {
		var deleteVolunteerString string
		if val.VolunteerID > 0 {
			deleteVolunteerString = fmt.Sprintf(`delete from Volunteers where User="%s" and VolunteerID=%d`, currentUser, val.VolunteerID)
		} else {
			deleteVolunteerString = fmt.Sprintf(`delete from Volunteers where User="%s" and VolunteerName="%s"`, currentUser, val.VolunteerName)
		}
		_, err := tx.Exec(deleteVolunteerString)
		if err != nil {
			return fmt.Errorf("error in DeleteVolunteers: sql.Tx.Exec error %w. Value of deleteVolunteerString is `%s`", err, deleteVolunteerString)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in DeleteVolunteers: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// Deletes all volunteers who do not have a VFS entry
func (vsam VSAModel) CleanOrphanedVolunteers(currentUser string) error {
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CleanOrphanedVolunteers: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	cleanOrphanedVolunteersString := fmt.Sprintf(`delete from Volunteers where User = "%s" and VolunteerID not in (select Volunteer from VolunteersForSchedule)`, currentUser)
	_, err = tx.Exec(cleanOrphanedVolunteersString)
	if err != nil {
		return fmt.Errorf("error in CleanOrphanedVolunteers: sql.Tx.Exec error: %w. Value of cleanOrphanedVolunteersString is `%s`", err, cleanOrphanedVolunteersString)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CleanOrphanedVolunteers: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// This function is the simplified version of CreateSchedulesExtended and does not allow schedules to be created where ShiftsOff = 0
func (vsam VSAModel) CreateSchedules(currentUser string, toCreate []schedule) error {
	err := vsam.CreateSchedulesExtended(currentUser, toCreate, false)
	if err != nil {
		return fmt.Errorf("error in CreateSchedules: %w", err)
	}
	return nil
}

func (vsam VSAModel) CreateSchedulesExtended(currentUser string, toCreate []schedule, includeShiftsOff0 bool) error {
	check, err := vsam.RequestSchedulesExtended(currentUser, toCreate, includeShiftsOff0)
	if err != nil {
		return fmt.Errorf("error in CreateSchedulesExtended: %w", err)
	}
	if len(check) > 0 {
		return fmt.Errorf("error in CreateSchedulesExtended: method failed because at least one of the schedules to be created already exists in the database. Existing schedule(s): %+v", check)
	}
	checkDuplicates := []schedule{}
	for _, val := range toCreate { // User and ScheduleID do not need to be provided in the schedule structs
		if val.ScheduleName <= (schedule{}.ScheduleName) {
			return fmt.Errorf("error in CreateSchedulesExtended: method failed because at least one of the schedule structs in toCreate did not have a valid value for ScheduleName: %+v", val)
		}
		if !includeShiftsOff0 && val.ShiftsOff <= (schedule{}.ShiftsOff) || val.ShiftsOff <= -1 {
			return fmt.Errorf("error in CreateSchedulesExtended: method failed because at least one of the schedule structs in toCreate did not have a valid value for ShiftsOff: %+v", val)
		}
		if val.VolunteersPerShift <= (schedule{}.VolunteersPerShift) {
			return fmt.Errorf("error in CreateSchedulesExtended: method failed because at least one of the schedule structs in toCreate did not have a valid value for VolunteersPerShift: %+v", val)
		}
		if val.StartDate <= (schedule{}.StartDate) {
			return fmt.Errorf("error in CreateSchedulesExtended: method failed because at least one of the schedule structs in toCreate did not have a valid value for StartDate: %+v", val)
		}
		if val.EndDate <= (schedule{}.EndDate) {
			return fmt.Errorf("error in CreateSchedulesExtended: method failed because at least one of the schedule structs in toCreate did not have a valid value for EndDate: %+v", val)
		}
		if !slices.Contains(checkDuplicates, schedule{ScheduleName: val.ScheduleName, ShiftsOff: val.ShiftsOff, VolunteersPerShift: val.VolunteersPerShift, StartDate: val.StartDate, EndDate: val.EndDate}) {
			checkDuplicates = append(checkDuplicates, schedule{ScheduleName: val.ScheduleName, ShiftsOff: val.ShiftsOff, VolunteersPerShift: val.VolunteersPerShift, StartDate: val.StartDate, EndDate: val.EndDate})
		} else {
			return fmt.Errorf("error in CreateSchedulesExtended: method failed because at least one of the schedule structs in toCreate was a duplicate of another schedule struct in toCreate: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CreateSchedulesExtended: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	fillSchedulesTableString := `insert into Schedules (ScheduleName, ShiftsOff, VolunteersPerShift, User, StartDate, EndDate) values (?, ?, ?, ?, ?, ?)`
	fillSchedulesTableStmt, err := tx.Prepare(fillSchedulesTableString)
	if err != nil {
		return fmt.Errorf("error in CreateSchedulesExtended: sql.Tx.Prepare error: %w. Value of fillSchedulesTableString is `%s`", err, fillSchedulesTableString)
	}
	defer fillSchedulesTableStmt.Close()
	for i := 0; i < len(toCreate); i++ {
		_, err = fillSchedulesTableStmt.Exec(toCreate[i].ScheduleName, toCreate[i].ShiftsOff, toCreate[i].VolunteersPerShift, currentUser, toCreate[i].StartDate, toCreate[i].EndDate)
		if err != nil {
			return fmt.Errorf("error in CreateSchedulesExtended: sql.Stmt.Exec error: %w. Value of toCreate[i] is `%+v`", err, toCreate[i])
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CreateSchedulesExtended: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// Calls RequestSchedules with scheduleStruct and verifies exactly one database row matches.
func (vsam VSAModel) RequestSchedule(currentUser string, scheduleStruct schedule) (schedule, error) {
	schedules, err := vsam.RequestSchedules(currentUser, []schedule{scheduleStruct})
	if err != nil {
		return schedule{}, fmt.Errorf("error in RequestSchedule: %w", err)
	}
	if len(schedules) != 1 {
		return schedule{}, fmt.Errorf("error in RequestSchedule: method failed to locate exactly one schedule matching %+v. Found %d matches", scheduleStruct, len(schedules))
	}
	return schedules[0], nil
}

// This function is the simple version of RequestSchedulesExtended and does not allow ShiftsOff = 0 to be queried.
func (vsam VSAModel) RequestSchedules(currentUser string, schedules []schedule) ([]schedule, error) {
	retrievedSchedules, err := vsam.RequestSchedulesExtended(currentUser, schedules, false)
	if err != nil {
		return []schedule{}, fmt.Errorf("error in RequestSchedules: %w", err)
	}
	return retrievedSchedules, nil
}

// This version of RequestSchedules allows ShiftsOff = 0 to be queried, but any default schedule structs will have ShiftsOff: 0 implicitly, so ShiftsOff must be set to a desired value or to -1 to be ignored.
func (vsam VSAModel) RequestSchedulesExtended(currentUser string, schedules []schedule, includeShiftsOff0 bool) ([]schedule, error) {
	schedulesQuery := fmt.Sprintf(`select * from Schedules where User = "%s"`, currentUser)
	if !includeShiftsOff0 { // I have to check for this edge case
		for _, val := range schedules {
			if val.ShiftsOff <= -1 {
				return []schedule{}, fmt.Errorf("error in RequestSchedulesExtended: method failed because includeShiftsOff0 is false but one of the schedule structs contains ShiftsOff<=-1: %+v", val)
			}
		}
	}
	if len(schedules) > 0 {
		var checkAgainst schedule
		if includeShiftsOff0 {
			checkAgainst = schedule{ShiftsOff: -1}
		} else {
			checkAgainst = schedule{}
		}
		if check, failed := testEmpty(schedules, checkAgainst); check {
			return []schedule{}, fmt.Errorf("error in RequestSchedulesExtended: method failed because one of the values in schedules had an empty/default values schedule struct: %+v", failed)
		}
		schedulesQuery = fmt.Sprintf(`%s and (`, schedulesQuery)
	}
	for i := 0; i < len(schedules); i++ {
		var count int
		if includeShiftsOff0 {
			count = countGTZero([]int{schedules[i].ScheduleID, len(schedules[i].ScheduleName), schedules[i].ShiftsOff + 1, schedules[i].VolunteersPerShift, len(schedules[i].User), schedules[i].StartDate, schedules[i].EndDate})
		} else {
			count = countGTZero([]int{schedules[i].ScheduleID, len(schedules[i].ScheduleName), schedules[i].ShiftsOff, schedules[i].VolunteersPerShift, len(schedules[i].User), schedules[i].StartDate, schedules[i].EndDate})
		}
		// count must be at least 1 because the testEmpty check passed
		//fmt.Println(count)
		schedulesQuery = fmt.Sprintf(`%s(`, schedulesQuery)
		if schedules[i].ScheduleID > 0 {
			schedulesQuery = fmt.Sprintf(`%sScheduleID = %d`, schedulesQuery, schedules[i].ScheduleID)
			count--
			if count > 0 {
				schedulesQuery = fmt.Sprintf(`%s and `, schedulesQuery)
			}
		}
		if len(schedules[i].ScheduleName) > 0 {
			schedulesQuery = fmt.Sprintf(`%sScheduleName = "%s"`, schedulesQuery, schedules[i].ScheduleName)
			count--
			if count > 0 {
				schedulesQuery = fmt.Sprintf(`%s and `, schedulesQuery)
			}
		}
		if schedules[i].ShiftsOff > 0 || includeShiftsOff0 && schedules[i].ShiftsOff > -1 {
			schedulesQuery = fmt.Sprintf(`%sShiftsOff = %d`, schedulesQuery, schedules[i].ShiftsOff)
			count--
			if count > 0 {
				schedulesQuery = fmt.Sprintf(`%s and `, schedulesQuery)
			}
		}
		if schedules[i].VolunteersPerShift > 0 {
			schedulesQuery = fmt.Sprintf(`%sVolunteersPerShift = %d`, schedulesQuery, schedules[i].VolunteersPerShift)
			count--
			if count > 0 {
				schedulesQuery = fmt.Sprintf(`%s and `, schedulesQuery)
			}
		}
		if len(schedules[i].User) > 0 {
			schedulesQuery = fmt.Sprintf(`%sUser = "%s"`, schedulesQuery, schedules[i].User)
			count--
			if count > 0 {
				schedulesQuery = fmt.Sprintf(`%s and `, schedulesQuery)
			}
		}
		if schedules[i].StartDate > 0 {
			schedulesQuery = fmt.Sprintf(`%sStartDate = %d`, schedulesQuery, schedules[i].StartDate)
			count--
			if count > 0 {
				schedulesQuery = fmt.Sprintf(`%s and `, schedulesQuery)
			}
		}
		if schedules[i].EndDate > 0 {
			schedulesQuery = fmt.Sprintf(`%sEndDate = %d`, schedulesQuery, schedules[i].EndDate)
		}
		schedulesQuery = fmt.Sprintf(`%s)`, schedulesQuery)
		if i+1 < len(schedules) {
			schedulesQuery = fmt.Sprintf(`%s or `, schedulesQuery)
		}
		//fmt.Println(count)
		//fmt.Println(schedulesQuery)
	}
	if len(schedules) > 0 {
		schedulesQuery = fmt.Sprintf(`%s)`, schedulesQuery)
	}
	//fmt.Println(schedulesQuery)
	var result []schedule
	rows, err := vsam.DB.Query(schedulesQuery)
	if err != nil {
		return []schedule{}, fmt.Errorf("error in RequestSchedulesExtended: sql.DB.Query error: %w. Value of schedulesQuery is `%s`", err, schedulesQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var userSchedule schedule
		err = rows.Scan(&userSchedule.ScheduleID, &userSchedule.ScheduleName, &userSchedule.ShiftsOff, &userSchedule.VolunteersPerShift, &userSchedule.User, &userSchedule.StartDate, &userSchedule.EndDate)
		if err != nil {
			return []schedule{}, fmt.Errorf("error in RequestSchedulesExtended: sql.Rows.Scan error: %w. Value of userSchedule is `%+v`", err, userSchedule)
		}
		result = append(result, userSchedule)
	}
	err = rows.Err()
	if err != nil {
		return []schedule{}, fmt.Errorf("error in RequestSchedulesExtended: sql.Rows.Err error: %w", err)
	}
	return result, nil
}

// This function is the simple version of UpdateSchedulesExtended and does not allow columns to set ShiftsOff = 0.
func (vsam VSAModel) UpdateSchedules(currentUser string, toUpdate []schedule) error {
	return vsam.UpdateSchedulesExtended(currentUser, toUpdate, false)
}

// This version of UpdateSchedulesExtended allows ShiftsOff = 0 to be queried, but any default schedule structs will have ShiftsOff: 0 implicitly, so ShiftsOff must be set to a desired value or to -1 to be ignored.
func (vsam VSAModel) UpdateSchedulesExtended(currentUser string, toUpdate []schedule, includeShiftsOff0 bool) error {
	var checkAgainst schedule
	if includeShiftsOff0 {
		checkAgainst = schedule{ShiftsOff: -1}
	} else {
		checkAgainst = schedule{}
	}
	if check, failed := testEmpty(toUpdate, checkAgainst); check {
		return fmt.Errorf("error in UpdateSchedulesExtended: method failed because one of the values in toUpdate had an empty/default values schedule struct: %+v", failed)
	}
	head := `update Schedules set`
	tail := fmt.Sprintf(`where User="%s" and ScheduleID=?`, currentUser)
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in UpdateSchedulesExtended: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	checkDuplicates := []schedule{}
	for _, val := range toUpdate {
		if val.ScheduleID == 0 {
			return fmt.Errorf("error in UpdateSchedulesExtended: method failed because one of the values in toUpdate had an empty/default value for ScheduleID: %+v", val)
		}
		currentSchedule, err := vsam.RequestSchedule(currentUser, schedule{ScheduleID: val.ScheduleID})
		if err != nil {
			return fmt.Errorf("error in UpdateSchedulesExtended: %w", err)
		}
		checkStruct := schedule{ScheduleName: val.ScheduleName, ShiftsOff: val.ShiftsOff, VolunteersPerShift: val.VolunteersPerShift, StartDate: val.StartDate, EndDate: val.EndDate}
		if !slices.Contains(checkDuplicates, checkStruct) {
			checkDuplicates = append(checkDuplicates, checkStruct)
		} else {
			return fmt.Errorf("error in UpdateSchedulesExtended: method failed because at least two of the schedule structs in toUpdate would create duplicate schedule structs in the database: %+v", checkStruct)
		}
		currentSchedule.ScheduleID = 0
		updateSchedulesString := head
		var count int
		if includeShiftsOff0 {
			count = countGTZero([]int{val.ScheduleID, len(val.ScheduleName), val.ShiftsOff + 1, val.VolunteersPerShift, len(val.User), val.StartDate, val.EndDate})
		} else {
			count = countGTZero([]int{val.ScheduleID, len(val.ScheduleName), val.ShiftsOff, val.VolunteersPerShift, len(val.User), val.StartDate, val.EndDate})
		}
		count-- // This is needed because a ScheduleID has been provided (verified at the start of this loop).
		if count == 0 {
			return fmt.Errorf("error in UpdateSchedulesExtended: method failed because only one value was provided in a schedule struct. At least two values (a ScheduleID and a value to update) must be provided: %+v", val)
		}
		// count is at least 1
		//fmt.Println(count)
		//fmt.Println(updateSchedulesString)
		if len(val.ScheduleName) > 0 {
			updateSchedulesString = fmt.Sprintf(`%s ScheduleName="%s"`, updateSchedulesString, val.ScheduleName)
			count--
			currentSchedule.ScheduleName = val.ScheduleName
			if count > 0 {
				updateSchedulesString = fmt.Sprintf(`%s,`, updateSchedulesString)
			}
			//fmt.Println(count)
			//fmt.Println(updateSchedulesString)
		}
		if val.ShiftsOff > 0 || includeShiftsOff0 && val.ShiftsOff > -1 {
			updateSchedulesString = fmt.Sprintf(`%s ShiftsOff=%d`, updateSchedulesString, val.ShiftsOff)
			count--
			currentSchedule.ShiftsOff = val.ShiftsOff
			if count > 0 {
				updateSchedulesString = fmt.Sprintf(`%s,`, updateSchedulesString)
			}
			//fmt.Println(count)
			//fmt.Println(updateSchedulesString)
		}
		if val.VolunteersPerShift > 0 {
			updateSchedulesString = fmt.Sprintf(`%s VolunteersPerShift=%d`, updateSchedulesString, val.VolunteersPerShift)
			count--
			currentSchedule.VolunteersPerShift = val.VolunteersPerShift
			if count > 0 {
				updateSchedulesString = fmt.Sprintf(`%s,`, updateSchedulesString)
			}
			//fmt.Println(count)
			//fmt.Println(updateSchedulesString)
		}
		if val.StartDate > 0 {
			updateSchedulesString = fmt.Sprintf(`%s StartDate=%d`, updateSchedulesString, val.StartDate)
			count--
			currentSchedule.StartDate = val.StartDate
			if count > 0 {
				updateSchedulesString = fmt.Sprintf(`%s,`, updateSchedulesString)
			}
			//fmt.Println(count)
			//fmt.Println(updateSchedulesString)
		}
		if val.EndDate > 0 {
			updateSchedulesString = fmt.Sprintf(`%s EndDate=%d`, updateSchedulesString, val.EndDate)
			currentSchedule.EndDate = val.EndDate
			//fmt.Println(count)
			//fmt.Println(updateSchedulesString)
		}
		updateSchedulesString = fmt.Sprintf(`%s %s`, updateSchedulesString, tail)
		//fmt.Println(count)
		//fmt.Println(updateSchedulesString)
		if check, err := vsam.RequestSchedulesExtended(currentUser, []schedule{currentSchedule}, includeShiftsOff0); err != nil {
			return fmt.Errorf("error in UpdateSchedulesExtended: %w", err)
		} else if len(check) > 0 {
			return fmt.Errorf("error in UpdateSchedulesExtended: method failed because it would create a duplicate schedule: %+v", val)
		}
		updateSchedulesStmt, err := tx.Prepare(updateSchedulesString)
		if err != nil {
			return fmt.Errorf("error in UpdateSchedulesExtended: sql.Tx.Prepare error: %w. Value of updateSchedulesString is `%s`", err, updateSchedulesString)
		}
		defer updateSchedulesStmt.Close()
		_, err = updateSchedulesStmt.Exec(val.ScheduleID)
		if err != nil {
			return fmt.Errorf("error in UpdateSchedulesExtended: sql.Stmt.Exec error: %w. Value of val is %+v", err, val)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in UpdateSchedulesExtended: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// Will delete Schedule database entries that match the ScheduleID or that match the ScheduleName provided in each schedule struct. If a ScheduleID > 0 is provided, the value for ScheduleName is ignored for that schedule struct.
func (vsam VSAModel) DeleteSchedules(currentUser string, toDelete []schedule) error {
	for _, val := range toDelete {
		if val.ScheduleID < 1 && len(val.ScheduleName) == 0 {
			return fmt.Errorf("error in DeleteSchedules: method failed because one of the schedule structs did not have a value for ScheduleID or ScheduleName (at least one must be provided): %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in DeleteSchedules: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	for _, val := range toDelete {
		var deleteScheduleString string
		if val.ScheduleID > 0 {
			deleteScheduleString = fmt.Sprintf(`delete from Schedules where User="%s" and ScheduleID=%d`, currentUser, val.ScheduleID)
		} else {
			deleteScheduleString = fmt.Sprintf(`delete from Schedules where User="%s" and ScheduleName="%s"`, currentUser, val.ScheduleName)
		}
		_, err := tx.Exec(deleteScheduleString)
		if err != nil {
			return fmt.Errorf("error in DeleteSchedules: sql.Tx.Exec error: %w. Value of deleteScheduleString is %s", err, deleteScheduleString)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in DeleteSchedules: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) CreateWFS(currentUser string, toCreate []weekdayForSchedule) error {
	check, err := vsam.RequestWFS(currentUser, toCreate)
	if err != nil {
		return fmt.Errorf("error in CreateWFS: %w", err)
	}
	if len(check) > 0 {
		return fmt.Errorf("error in CreateWFS: method failed because at least one of the weekdayForSchedule entries to be created already exists in the database. Existing weekdayForSchedule(s): %+v", check)
	}
	checkDuplicates := []weekdayForSchedule{}
	for _, val := range toCreate { // User and WFSID do not need to be provided in the weekdayForSchedule structs
		if val.Weekday == (weekdayForSchedule{}.Weekday) {
			return fmt.Errorf("error in CreateWFS: method failed because at least one of the weekdayForSchedule structs in toCreate did not have a value for Weekday: %+v", val)
		}
		if val.Schedule == (weekdayForSchedule{}.Schedule) {
			return fmt.Errorf("error in CreateWFS: method failed because at least one of the weekdayForSchedule structs in toCreate did not have a value for Schedule: %+v", val)
		}
		if !slices.Contains(checkDuplicates, weekdayForSchedule{Weekday: val.Weekday, Schedule: val.Schedule}) {
			checkDuplicates = append(checkDuplicates, weekdayForSchedule{Weekday: val.Weekday, Schedule: val.Schedule})
		} else {
			return fmt.Errorf("error in CreateWFS: method failed because at least one of the weekdayForSchedule structs in toCreate was a duplicate of another weekdayForSchedule struct in toCreate: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CreateWFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	fillWFSTableString := `insert into WeekdaysForSchedule (User, Weekday, Schedule) values (?, ?, ?)`
	fillWFSTableStmt, err := tx.Prepare(fillWFSTableString)
	if err != nil {
		return fmt.Errorf("error in CreateWFS: sql.Tx.Prepare error: %w. Value of fillWFSTableString is `%s`", err, fillWFSTableString)
	}
	defer fillWFSTableStmt.Close()
	for i := 0; i < len(toCreate); i++ {
		_, err = fillWFSTableStmt.Exec(currentUser, toCreate[i].Weekday, toCreate[i].Schedule)
		if err != nil {
			return fmt.Errorf("error in CreateWFS: sql.Stmt.Exec error: %w. Value of toCreate[i] is `%+v`", err, toCreate[i])
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CreateWFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) RequestWFSSingle(currentUser string, volunteerForScheduleStruct weekdayForSchedule) (weekdayForSchedule, error) {
	weekdaysForSchedule, err := vsam.RequestWFS(currentUser, []weekdayForSchedule{volunteerForScheduleStruct})
	if err != nil {
		return weekdayForSchedule{}, fmt.Errorf("error in RequestWFSSingle: %w", err)
	}
	if len(weekdaysForSchedule) != 1 {
		return weekdayForSchedule{}, fmt.Errorf("error in RequestWFSSingle: method failed to locate exactly one WFS matching %+v. Found %d matches", volunteerForScheduleStruct, len(weekdaysForSchedule))
	}
	return weekdaysForSchedule[0], nil
}

func (vsam VSAModel) RequestWFS(currentUser string, weekdaysForSchedule []weekdayForSchedule) ([]weekdayForSchedule, error) {
	weekdaysForScheduleQuery := fmt.Sprintf(`select * from WeekdaysForSchedule where User = "%s"`, currentUser)
	if len(weekdaysForSchedule) > 0 {
		if check, failed := testEmpty(weekdaysForSchedule, weekdayForSchedule{}); check {
			return []weekdayForSchedule{}, fmt.Errorf("error in RequestWFS: method failed because one of the values in weekdaysForSchedule had an empty/default values weekdayForSchedule struct: %+v", failed)
		}
		weekdaysForScheduleQuery = fmt.Sprintf(`%s and (`, weekdaysForScheduleQuery)
	}
	for i := 0; i < len(weekdaysForSchedule); i++ {
		count := countGTZero([]int{weekdaysForSchedule[i].WFSID, len(weekdaysForSchedule[i].User), weekdaysForSchedule[i].Schedule, len(weekdaysForSchedule[i].Weekday)})
		// count must be at least 1 because the testEmpty check passed
		//fmt.Println(count)
		weekdaysForScheduleQuery = fmt.Sprintf(`%s(`, weekdaysForScheduleQuery)
		if weekdaysForSchedule[i].WFSID > 0 {
			weekdaysForScheduleQuery = fmt.Sprintf(`%sWFSID = %d`, weekdaysForScheduleQuery, weekdaysForSchedule[i].WFSID)
			count--
			if count > 0 {
				weekdaysForScheduleQuery = fmt.Sprintf(`%s and `, weekdaysForScheduleQuery)
			}
		}
		if len(weekdaysForSchedule[i].User) > 0 {
			weekdaysForScheduleQuery = fmt.Sprintf(`%sUser = "%s"`, weekdaysForScheduleQuery, weekdaysForSchedule[i].User)
			count--
			if count > 0 {
				weekdaysForScheduleQuery = fmt.Sprintf(`%s and `, weekdaysForScheduleQuery)
			}
		}
		if weekdaysForSchedule[i].Schedule > 0 {
			weekdaysForScheduleQuery = fmt.Sprintf(`%sSchedule = %d`, weekdaysForScheduleQuery, weekdaysForSchedule[i].Schedule)
			count--
			if count > 0 {
				weekdaysForScheduleQuery = fmt.Sprintf(`%s and `, weekdaysForScheduleQuery)
			}
		}
		if len(weekdaysForSchedule[i].Weekday) > 0 {
			weekdaysForScheduleQuery = fmt.Sprintf(`%sWeekday = "%s"`, weekdaysForScheduleQuery, weekdaysForSchedule[i].Weekday)
		}
		weekdaysForScheduleQuery = fmt.Sprintf(`%s)`, weekdaysForScheduleQuery)
		if i+1 < len(weekdaysForSchedule) {
			weekdaysForScheduleQuery = fmt.Sprintf(`%s or `, weekdaysForScheduleQuery)
		}
		//fmt.Println(count)
		//fmt.Println(weekdaysForScheduleQuery)
	}
	if len(weekdaysForSchedule) > 0 {
		weekdaysForScheduleQuery = fmt.Sprintf(`%s)`, weekdaysForScheduleQuery)
	}
	//fmt.Println(weekdaysForScheduleQuery)
	var result []weekdayForSchedule
	rows, err := vsam.DB.Query(weekdaysForScheduleQuery)
	if err != nil {
		return []weekdayForSchedule{}, fmt.Errorf("error in RequestWFS: sql.DB.Query error: %w. Value of weekdaysForScheduleQuery is `%s`", err, weekdaysForScheduleQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var weekdayForScheduleStruct weekdayForSchedule
		err = rows.Scan(&weekdayForScheduleStruct.WFSID, &weekdayForScheduleStruct.User, &weekdayForScheduleStruct.Weekday, &weekdayForScheduleStruct.Schedule)
		if err != nil {
			return []weekdayForSchedule{}, fmt.Errorf("error in RequestWFS: sql.Rows.Scan error: %w. value of weekdayForScheduleStruct is `%+v`", err, weekdayForScheduleStruct)
		}
		result = append(result, weekdayForScheduleStruct)
	}
	err = rows.Err()
	if err != nil {
		return []weekdayForSchedule{}, fmt.Errorf("error in RequestWFS: sql.Rows.Err error: %w", err)
	}
	return result, nil
}

func (vsam VSAModel) UpdateWFS(currentUser string, toUpdate []weekdayForSchedule) error {
	if check, failed := testEmpty(toUpdate, weekdayForSchedule{}); check {
		return fmt.Errorf("error in UpdateWFS: method failed because one of the values in toUpdate had an empty/default values weekdayForSchedule struct: %+v", failed)
	}
	head := `update WeekdaysForSchedule set`
	tail := fmt.Sprintf(`where User="%s" and WFSID=?`, currentUser)
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in UpdateWFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	checkDuplicates := []weekdayForSchedule{}
	for _, val := range toUpdate {
		if val.WFSID == 0 {
			return fmt.Errorf("error in UpdateWFS: method failed because one of the values in toUpdate had an empty/default value for WFSID: %+v", val)
		}
		currentWFS, err := vsam.RequestWFSSingle(currentUser, weekdayForSchedule{WFSID: val.WFSID})
		if err != nil {
			return fmt.Errorf("error in UpdateWFS: %w", err)
		}
		if !slices.Contains(checkDuplicates, weekdayForSchedule{Weekday: val.Weekday, Schedule: val.Schedule}) {
			checkDuplicates = append(checkDuplicates, weekdayForSchedule{Weekday: val.Weekday, Schedule: val.Schedule})
		} else {
			return fmt.Errorf("error in UpdateWFS: method failed because at least two of the weekdayForSchedule structs in toUpdate would create duplicate weekdayForSchedule structs in the database: %+v", weekdayForSchedule{Weekday: val.Weekday, Schedule: val.Schedule})
		}
		currentWFS.WFSID = 0
		updateWFSString := head
		count := countGTZero([]int{val.WFSID, len(val.User), len(val.Weekday), val.Schedule})
		count-- // This is needed because a WFSID has been provided (verified at the start of this loop).
		if count == 0 {
			return fmt.Errorf("error in UpdateWFS: method failed because only one value was provided in a weekdayForSchedule struct. At least two values (a WFSID and a value to update) must be provided: %+v", val)
		}
		// count is at least 1
		//fmt.Println(count)
		//fmt.Println(updateWFSString)
		if len(val.Weekday) > 0 {
			updateWFSString = fmt.Sprintf(`%s Weekday="%s"`, updateWFSString, val.Weekday)
			count--
			currentWFS.Weekday = val.Weekday
			if count > 0 {
				updateWFSString = fmt.Sprintf(`%s,`, updateWFSString)
			}
			//fmt.Println(count)
			//fmt.Println(updateWFSString)
		}
		if val.Schedule > 0 {
			updateWFSString = fmt.Sprintf(`%s Schedule=%d`, updateWFSString, val.Schedule)
			//fmt.Println(count)
			//fmt.Println(updateWFSString)
			currentWFS.Schedule = val.Schedule
		}
		updateWFSString = fmt.Sprintf(`%s %s`, updateWFSString, tail)
		//fmt.Println(count)
		//fmt.Println(updateWFSString)
		if check, err := vsam.RequestWFS(currentUser, []weekdayForSchedule{currentWFS}); err != nil {
			return fmt.Errorf("error in UpdateWFS: %w", err)
		} else if len(check) > 0 {
			return fmt.Errorf("error in UpdateWFS: method failed because it would create a duplicate WFS: %+v", val)
		}
		updateSchedulesStmt, err := tx.Prepare(updateWFSString)
		if err != nil {
			return fmt.Errorf("error in UpdateWFS: sql.Tx.Prepare error: %w", err)
		}
		defer updateSchedulesStmt.Close()
		_, err = updateSchedulesStmt.Exec(val.WFSID)
		if err != nil {
			return fmt.Errorf("error in UpdateWFS: sql.Stmt.Exec error: %w", err)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in UpdateWFS sql.Tx.Commit: %w", err)
	}
	return nil
}

// Will delete WFS database entries that match the WFSID or that match the Weekday and Schedule provided in each WFS struct. If a WFSID > 0 is provided, the values for Weekday and Schedule are ignored for that WFS struct.
func (vsam VSAModel) DeleteWFS(currentUser string, toDelete []weekdayForSchedule) error {
	for _, val := range toDelete {
		if val.WFSID < 1 && (len(val.Weekday) == 0 || val.Schedule < 1) {
			return fmt.Errorf("error in DeleteWFS: method failed because one of the weekdayForSchedule structs did not have a value for WFSID or Weekday and Schedule: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in DeleteWFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	for _, val := range toDelete {
		var deleteWFSString string
		if val.WFSID > 0 {
			deleteWFSString = fmt.Sprintf(`delete from WeekdaysForSchedule where User="%s" and WFSID=%d`, currentUser, val.WFSID)
		} else {
			deleteWFSString = fmt.Sprintf(`delete from WeekdaysForSchedule where User="%s" and Weekday="%s" and Schedule=%d`, currentUser, val.Weekday, val.Schedule)
		}
		_, err := tx.Exec(deleteWFSString)
		if err != nil {
			return fmt.Errorf("error in DeleteWFS: sql.Tx.Exec error: %w", err)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in DeleteWFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// correctWFS is a map with schedule structs as keys and slices of weekday structs that define WeekdayName as values. If a WFS row is linked to a schedule, but doesn't have a matching weekday, delete that WFS row.
func (vsam VSAModel) CleanOrphanedWFS(currentUser string, correctWFS map[schedule][]weekday) error {
	var WFSToDelete []string
	for key, value := range correctWFS {
		if key.ScheduleID == 0 {
			return fmt.Errorf("error in CleanOrphanedWFS: method failed because one of the provided schedule structs did not have a ScheduleID: %+v", key)
		}
		var weekdays []string
		for _, weekdayStruct := range value {
			if len(weekdayStruct.WeekdayName) < 6 { // smallest weekday name is 6 characters long.
				return fmt.Errorf("error in CleanOrphanedWFS: method failed because one of the provided weekday structs did not have a WeekdayName: %+v", value)
			}
			weekdays = append(weekdays, weekdayStruct.WeekdayName)
		}
		wfsCheck, _ := vsam.RequestWFS(currentUser, []weekdayForSchedule{{Schedule: key.ScheduleID}}) // catch error when updating this func
		for _, wfs := range wfsCheck {
			if !slices.Contains(weekdays, wfs.Weekday) {
				WFSToDelete = append(WFSToDelete, strconv.Itoa(wfs.WFSID))
				//fmt.Println(WFSToDelete)
			}
		}
		tx, err := vsam.DB.Begin()
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedWFS: sql.DB.Begin error: %w", err)
		}
		defer tx.Rollback()
		deleteWFSQuery := fmt.Sprintf(`delete from WeekdaysForSchedule where User = "%s" and WFSID in (%s)`, currentUser, CsvSlice(WFSToDelete, true))
		//fmt.Println(deleteWFSQuery)
		_, err = tx.Exec(deleteWFSQuery)
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedWFS: sql.Tx.Exec error: %w. Value of deleteWFSQuery is `%s`", err, deleteWFSQuery)
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedWFS: sql.Tx.Commit error: %w", err)
		}
	}
	return nil
}

func (vsam VSAModel) CreateVFS(currentUser string, toCreate []volunteerForSchedule) error {
	check, err := vsam.RequestVFS(currentUser, toCreate)
	if err != nil {
		return fmt.Errorf("error in CreateVFS: %w", err)
	}
	if len(check) > 0 {
		return fmt.Errorf("error in CreateVFS: method failed because at least one of the volunteerForSchedule entries to be created already exists in the database. Existing volunteerForSchedule entry(s): %+v", check)
	}
	checkDuplicates := []volunteerForSchedule{}
	for _, val := range toCreate { // User and VFSID do not need to be provided in the volunteerForSchedule structs
		if val.Schedule == (volunteerForSchedule{}.Schedule) {
			return fmt.Errorf("error in CreateVFS: method failed because at least one of the volunteerForSchedule structs in toCreate did not have a value for Schedule: %+v", val)
		}
		if val.Volunteer == (volunteerForSchedule{}.Volunteer) {
			return fmt.Errorf("error in CreateVFS: method failed because at least one of the volunteerForSchedule structs in toCreate did not have a value for Volunteer: %+v", val)
		}
		if !slices.Contains(checkDuplicates, volunteerForSchedule{Schedule: val.Schedule, Volunteer: val.Volunteer}) {
			checkDuplicates = append(checkDuplicates, volunteerForSchedule{Schedule: val.Schedule, Volunteer: val.Volunteer})
		} else {
			return fmt.Errorf("error in CreateVFS: method failed because at least one of the volunteerForSchedule structs in toCreate was a duplicate of another volunteerForSchedule struct in toCreate: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CreateVFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	fillVFSTableString := `insert into VolunteersForSchedule (User, Schedule, Volunteer) values (?, ?, ?)`
	fillVFSTableStmt, err := tx.Prepare(fillVFSTableString)
	if err != nil {
		return fmt.Errorf("error in CreateVFS: sql.Tx.Prepare error: %w. Value of fillVFSTableString is `%s`", err, fillVFSTableString)
	}
	defer fillVFSTableStmt.Close()
	for i := 0; i < len(toCreate); i++ {
		_, err = fillVFSTableStmt.Exec(currentUser, toCreate[i].Schedule, toCreate[i].Volunteer)
		if err != nil {
			return fmt.Errorf("error in CreateVFS: sql.Stmt.Exec error: %w. Value of toCreate[i] is `%+v`", err, toCreate[i])
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CreateVFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) RequestVFSSingle(currentUser string, volunteerForScheduleStruct volunteerForSchedule) (volunteerForSchedule, error) {
	volunteersForSchedule, err := vsam.RequestVFS(currentUser, []volunteerForSchedule{volunteerForScheduleStruct})
	if err != nil {
		return volunteerForSchedule{}, fmt.Errorf("error in RequestVFSSingle: %w", err)
	}
	if len(volunteersForSchedule) != 1 {
		return volunteerForSchedule{}, fmt.Errorf("error in RequestVFSSingle: method failed to locate exactly one VFS matching %+v. Found %d matches", volunteerForScheduleStruct, len(volunteersForSchedule))
	}
	return volunteersForSchedule[0], nil
}

func (vsam VSAModel) RequestVFS(currentUser string, volunteersForSchedule []volunteerForSchedule) ([]volunteerForSchedule, error) {
	VFSQuery := fmt.Sprintf(`select * from VolunteersForSchedule where User = "%s"`, currentUser)
	if len(volunteersForSchedule) > 0 {
		if check, failed := testEmpty(volunteersForSchedule, volunteerForSchedule{}); check {
			return []volunteerForSchedule{}, fmt.Errorf("error in RequestVFS: method failed because one of the values in volunteersForSchedule had an empty/default values volunteerForSchedule struct: %+v", failed)
		}
		VFSQuery = fmt.Sprintf(`%s and (`, VFSQuery)
	}
	for i := 0; i < len(volunteersForSchedule); i++ {
		count := countGTZero([]int{volunteersForSchedule[i].VFSID, len(volunteersForSchedule[i].User), volunteersForSchedule[i].Schedule, volunteersForSchedule[i].Volunteer})
		VFSQuery = fmt.Sprintf(`%s(`, VFSQuery)
		if volunteersForSchedule[i].VFSID > 0 {
			VFSQuery = fmt.Sprintf(`%sVFSID = %d`, VFSQuery, volunteersForSchedule[i].VFSID)
			count--
			if count > 0 {
				VFSQuery = fmt.Sprintf(`%s and `, VFSQuery)
			}
		}
		if len(volunteersForSchedule[i].User) > 0 {
			VFSQuery = fmt.Sprintf(`%sUser = "%s"`, VFSQuery, volunteersForSchedule[i].User)
			count--
			if count > 0 {
				VFSQuery = fmt.Sprintf(`%s and `, VFSQuery)
			}
		}
		if volunteersForSchedule[i].Schedule > 0 {
			VFSQuery = fmt.Sprintf(`%sSchedule = %d`, VFSQuery, volunteersForSchedule[i].Schedule)
			count--
			if count > 0 {
				VFSQuery = fmt.Sprintf(`%s and `, VFSQuery)
			}
		}
		if volunteersForSchedule[i].Volunteer > 0 {
			VFSQuery = fmt.Sprintf(`%sVolunteer = %d`, VFSQuery, volunteersForSchedule[i].Volunteer)
		}
		VFSQuery = fmt.Sprintf(`%s)`, VFSQuery)
		if i+1 < len(volunteersForSchedule) {
			VFSQuery = fmt.Sprintf(`%s or `, VFSQuery)
		}
		//fmt.Println(count)
		//fmt.Println(VFSQuery)
	}
	if len(volunteersForSchedule) > 0 {
		VFSQuery = fmt.Sprintf(`%s)`, VFSQuery)
	}
	//fmt.Println(VFSQuery)
	var result []volunteerForSchedule
	rows, err := vsam.DB.Query(VFSQuery)
	if err != nil {
		return []volunteerForSchedule{}, fmt.Errorf("error in RequestVFS: sql.DB.Query error: %w. Value of VFSQuery is `%s`", err, VFSQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var VFSStruct volunteerForSchedule
		err = rows.Scan(&VFSStruct.VFSID, &VFSStruct.User, &VFSStruct.Schedule, &VFSStruct.Volunteer)
		if err != nil {
			return []volunteerForSchedule{}, fmt.Errorf("error in RequestVFS: sql.Rows.Scan error: %w. Value of VFSStruct is `%+v`", err, VFSStruct)
		}
		result = append(result, VFSStruct)
	}
	err = rows.Err()
	if err != nil {
		return []volunteerForSchedule{}, fmt.Errorf("error in RequestVFS: sql.Rows.Err error: %w", err)
	}
	return result, nil
}

func (vsam VSAModel) UpdateVFS(currentUser string, toUpdate []volunteerForSchedule) error {
	if check, failed := testEmpty(toUpdate, volunteerForSchedule{}); check {
		return fmt.Errorf("error in UpdateVFS: method failed because one of the values in toUpdate had an empty/default values volunteerForSchedule struct: %+v", failed)
	}
	head := `update VolunteersForSchedule set`
	tail := fmt.Sprintf(`where User="%s" and VFSID=?`, currentUser)
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in UpdateVFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	checkDuplicates := []volunteerForSchedule{}
	for _, val := range toUpdate {
		if val.VFSID == 0 {
			return fmt.Errorf("error in UpdateVFS: method failed because one of the values in toUpdate had an empty/default value for VFSID: %+v", val)
		}
		currentVFS, err := vsam.RequestVFSSingle(currentUser, volunteerForSchedule{VFSID: val.VFSID})
		if err != nil {
			return fmt.Errorf("error in UpdateWFS: %w", err)
		}
		if !slices.Contains(checkDuplicates, volunteerForSchedule{Schedule: val.Schedule, Volunteer: val.Volunteer}) {
			checkDuplicates = append(checkDuplicates, volunteerForSchedule{Schedule: val.Schedule, Volunteer: val.Volunteer})
		} else {
			return fmt.Errorf("error in UpdateVFS: method failed because at least two of the volunteerForSchedule structs in toUpdate would create duplicate volunteerForSchedule structs in the database: %+v", volunteerForSchedule{Schedule: val.Schedule, Volunteer: val.Volunteer})
		}
		currentVFS.VFSID = 0
		updateVFSString := head
		count := countGTZero([]int{val.VFSID, len(val.User), val.Schedule, val.Volunteer})
		count-- // This is needed because a VFSID has been provided (verified at the start of this loop).
		if count == 0 {
			return fmt.Errorf("error in UpdateVFS: method failed because only one value was provided in a volunteerForSchedule struct. At least two values (a VFSID and a value to update) must be provided: %+v", val)
		}
		// count is at least 1
		//fmt.Println(count)
		//fmt.Println(updateVFSString)
		if val.Schedule > 0 {
			updateVFSString = fmt.Sprintf(`%s Schedule=%d`, updateVFSString, val.Schedule)
			count--
			currentVFS.Schedule = val.Schedule
			if count > 0 {
				updateVFSString = fmt.Sprintf(`%s,`, updateVFSString)
			}
			//fmt.Println(count)
			//fmt.Println(updateVFSString)
		}
		if val.Volunteer > 0 {
			updateVFSString = fmt.Sprintf(`%s Volunteer=%d`, updateVFSString, val.Volunteer)
			//fmt.Println(count)
			//fmt.Println(updateVFSString)
			currentVFS.Volunteer = val.Volunteer
		}
		updateVFSString = fmt.Sprintf(`%s %s`, updateVFSString, tail)
		//fmt.Println(count)
		//fmt.Println(updateVFSString)
		if check, err := vsam.RequestVFS(currentUser, []volunteerForSchedule{currentVFS}); err != nil {
			return fmt.Errorf("error in UpdateVFS: %w", err)
		} else if len(check) > 0 {
			return fmt.Errorf("error in UpdateVFS: method failed because it would create a duplicate VFS: %+v", val)
		}
		updateSchedulesStmt, err := tx.Prepare(updateVFSString)
		if err != nil {
			return fmt.Errorf("error in UpdateVFS: sql.Tx.Prepare error: %w. Value of updateVFSString is `%s`", err, updateVFSString)
		}
		defer updateSchedulesStmt.Close()
		_, err = updateSchedulesStmt.Exec(val.VFSID)
		if err != nil {
			return fmt.Errorf("error in UpdateVFS: sql.Stmt.Exec error: %w. Value of val is `%+v`", err, val)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in UpdateVFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// Will delete VFS database entries that match the VFSID or that match the Schedule and Volunteer provided in each VFS struct. If a VFSID > 0 is provided, the values for Schedule and Volunteer are ignored for that VFS struct.
func (vsam VSAModel) DeleteVFS(currentUser string, toDelete []volunteerForSchedule) error {
	for _, val := range toDelete {
		if val.VFSID < 1 && (val.Schedule < 1 || val.Volunteer < 1) {
			return fmt.Errorf("error in DeleteVFS: method failed because one of the volunteerForSchedule structs did not have a value for VFSID or Schedule and Volunteer: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in DeleteVFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	for _, val := range toDelete {
		var deleteVFSString string
		if val.VFSID > 0 {
			deleteVFSString = fmt.Sprintf(`delete from VolunteersForSchedule where User="%s" and VFSID=%d`, currentUser, val.VFSID)
		} else {
			deleteVFSString = fmt.Sprintf(`delete from VolunteersForSchedule where User="%s" and Schedule=%d and Volunteer=%d`, currentUser, val.Schedule, val.Volunteer)
		}
		_, err := tx.Exec(deleteVFSString)
		if err != nil {
			return fmt.Errorf("error in DeleteVFS: sql.Tx.Exec error: %w. Value of deleteVFSString is `%s`", err, deleteVFSString)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in DeleteVFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// correctVFS is a slices of maps with schedule structs as keys and slices of volunteers containing VolunteerNames as values. If a VFS row is linked to a schedule, but doesn't have a matching volunteer, delete that VFS row.
func (vsam VSAModel) CleanOrphanedVFS(currentUser string, correctVFS map[schedule][]volunteer, deleteChildUFS bool, deleteChildSVOD bool) error {
	var VFSToDelete []string
	for key, value := range correctVFS {
		if key.ScheduleID == 0 {
			return fmt.Errorf("error in CleanOrphanedVFS: method failed because one of the provided schedule structs did not have a ScheduleID: %+v", value)
		}
		var volunteers []string
		for _, volunteerStruct := range value {
			if len(volunteerStruct.VolunteerName) < 1 {
				return fmt.Errorf("error in CleanOrphanedVFS: method failed because one of the provided volunteer structs did not have a VolunteerName: %+v", value)
			}
			volunteers = append(volunteers, volunteerStruct.VolunteerName)
		}
		vfsCheck, err := vsam.RequestVFS(currentUser, []volunteerForSchedule{{Schedule: key.ScheduleID}})
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedVFS: %w", err)
		}
		for _, vfs := range vfsCheck {
			nameCheck, err := vsam.RequestVolunteer(currentUser, volunteer{VolunteerID: vfs.Volunteer})
			if err != nil {
				return fmt.Errorf("error in CleanOrphanedVFS: %w", err)
			}
			if !slices.Contains(volunteers, nameCheck.VolunteerName) {
				VFSToDelete = append(VFSToDelete, strconv.Itoa(vfs.VFSID))
				//log.Printf("VFSToDelete=`%v`; nameCheck.VolunteerName=`%s`", VFSToDelete, nameCheck.VolunteerName)
			}
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CleanOrphanedVFS: sql.DB.begin error: %w", err)
	}
	defer tx.Rollback()
	if deleteChildUFS {
		UFSToDelete := []unavailabilityForSchedule{}
		for _, vfsidString := range VFSToDelete {
			vfsidInt, err := strconv.Atoi(vfsidString)
			if err != nil {
				return fmt.Errorf("error in CleanOrphanedVFS: %w", err)
			}
			ufsSlice, err := vsam.RequestUFS(currentUser, []unavailabilityForSchedule{{VolunteerForSchedule: vfsidInt}})
			if err != nil {
				return fmt.Errorf("error in CleanOrphanedVFS: %w", err)
			}
			UFSToDelete = append(UFSToDelete, ufsSlice...)
		}
		vsam.DeleteUFS(currentUser, UFSToDelete)
	}
	if deleteChildSVOD {
		SVODToDelete := []scheduledVolunteerOnDate{}
		for _, vfsidString := range VFSToDelete {
			vfsidInt, err := strconv.Atoi(vfsidString)
			if err != nil {
				return fmt.Errorf("error in CleanOrphanedVFS: %w", err)
			}
			svodSlice, err := vsam.RequestSVOD(currentUser, []scheduledVolunteerOnDate{{VolunteerForSchedule: vfsidInt}})
			if err != nil {
				return fmt.Errorf("error in CleanOrphanedVFS: %w", err)
			}
			SVODToDelete = append(SVODToDelete, svodSlice...)
		}
		vsam.DeleteSVOD(currentUser, SVODToDelete)
	}
	deleteVFSQuery := fmt.Sprintf(`delete from VolunteersForSchedule where User = "%s" and VFSID in (%s)`, currentUser, CsvSlice(VFSToDelete, true))
	//fmt.Println(deleteVFSQuery)
	_, err = tx.Exec(deleteVFSQuery)
	if err != nil {
		return fmt.Errorf("error in CleanOrphanedVFS: sql.Tx.Exec error: %w. Value of deleteVFSQuery is `%s`", err, deleteVFSQuery)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CleanOrphanedVFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) CreateUFS(currentUser string, toCreate []unavailabilityForSchedule) error {
	check, err := vsam.RequestUFS(currentUser, toCreate)
	if err != nil {
		return fmt.Errorf("error in CreateUFS: %w", err)
	}
	if len(check) > 0 {
		return fmt.Errorf("error in CreateUFS: method failed because at least one of the unavailabilityForSchedule entries to be created already exists in the database. Existing unavailabilityForSchedule entry(s): %+v", check)
	}
	checkDuplicates := []unavailabilityForSchedule{}
	for _, val := range toCreate { // User and UFSID do not need to be provided in the unavailabilityForSchedule structs
		if val.VolunteerForSchedule == (unavailabilityForSchedule{}.VolunteerForSchedule) {
			return fmt.Errorf("error in CreateUFS: method failed because at least one of the unavailabilityForSchedule structs in toCreate did not have a value for VolunteerForSchedule: %+v", val)
		}
		if val.Date == (unavailabilityForSchedule{}.Date) {
			return fmt.Errorf("error in CreateUFS: method failed because at least one of the unavailabilityForSchedule structs in toCreate did not have a value for Date: %+v", val)
		}
		if !slices.Contains(checkDuplicates, unavailabilityForSchedule{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date}) {
			checkDuplicates = append(checkDuplicates, unavailabilityForSchedule{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date})
		} else {
			return fmt.Errorf("error in CreateUFS: method failed because at least one of the unavailabilityForSchedule structs in toCreate was a duplicate of another unavailabilityForSchedule struct in toCreate: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CreateUFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	fillUFSTableString := `insert into UnavailabilitiesForSchedule (User, VolunteerForSchedule, Date) values (?, ?, ?)`
	fillUFSTableStmt, err := tx.Prepare(fillUFSTableString)
	if err != nil {
		return fmt.Errorf("error in CreateUFS: sql.Tx.Prepare error: %w. Value of fillUFSTableString is `%s`", err, fillUFSTableString)
	}
	defer fillUFSTableStmt.Close()
	for i := 0; i < len(toCreate); i++ {
		_, err = fillUFSTableStmt.Exec(currentUser, toCreate[i].VolunteerForSchedule, toCreate[i].Date)
		if err != nil {
			return fmt.Errorf("error in CreateUFS: sql.Stmt.Exec error: %w. Value of toCreate[i] is `%+v`", err, toCreate[i])
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CreateUFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) RequestUFSSingle(currentUser string, unavailabilityForScheduleStruct unavailabilityForSchedule) (unavailabilityForSchedule, error) {
	unavailabilitiesForSchedule, err := vsam.RequestUFS(currentUser, []unavailabilityForSchedule{unavailabilityForScheduleStruct})
	if err != nil {
		return unavailabilityForSchedule{}, fmt.Errorf("error in RequestUFSSingle: %w", err)
	}
	if len(unavailabilitiesForSchedule) != 1 {
		return unavailabilityForSchedule{}, fmt.Errorf("error in RequestUFSSingle: method failed to locate exactly one UFS matching %+v. Found %d matches", unavailabilityForScheduleStruct, len(unavailabilitiesForSchedule))
	}
	return unavailabilitiesForSchedule[0], nil
}

func (vsam VSAModel) RequestUFS(currentUser string, unavailabilitiesForSchedule []unavailabilityForSchedule) ([]unavailabilityForSchedule, error) {
	UFSQuery := fmt.Sprintf(`select * from UnavailabilitiesForSchedule where User = "%s"`, currentUser)
	if len(unavailabilitiesForSchedule) > 0 {
		if check, failed := testEmpty(unavailabilitiesForSchedule, unavailabilityForSchedule{}); check {
			return []unavailabilityForSchedule{}, fmt.Errorf("error in RequestUFS: method failed because one of the values in unavailabilitiesForSchedule had an empty/default values unavailabilityForSchedule struct: %+v", failed)
		}
		UFSQuery = fmt.Sprintf(`%s and (`, UFSQuery)
	}
	for i := 0; i < len(unavailabilitiesForSchedule); i++ {
		count := countGTZero([]int{unavailabilitiesForSchedule[i].UFSID, len(unavailabilitiesForSchedule[i].User), unavailabilitiesForSchedule[i].VolunteerForSchedule, unavailabilitiesForSchedule[i].Date})
		// count must be at least 1 because the testEmpty check passed
		//fmt.Println(count)
		UFSQuery = fmt.Sprintf(`%s(`, UFSQuery)
		if unavailabilitiesForSchedule[i].UFSID > 0 {
			UFSQuery = fmt.Sprintf(`%sUFSID = %d`, UFSQuery, unavailabilitiesForSchedule[i].UFSID)
			count--
			if count > 0 {
				UFSQuery = fmt.Sprintf(`%s and `, UFSQuery)
			}
		}
		if len(unavailabilitiesForSchedule[i].User) > 0 {
			UFSQuery = fmt.Sprintf(`%sUser = "%s"`, UFSQuery, unavailabilitiesForSchedule[i].User)
			count--
			if count > 0 {
				UFSQuery = fmt.Sprintf(`%s and `, UFSQuery)
			}
		}
		if unavailabilitiesForSchedule[i].VolunteerForSchedule > 0 {
			UFSQuery = fmt.Sprintf(`%sVolunteerForSchedule = %d`, UFSQuery, unavailabilitiesForSchedule[i].VolunteerForSchedule)
			count--
			if count > 0 {
				UFSQuery = fmt.Sprintf(`%s and `, UFSQuery)
			}
		}
		if unavailabilitiesForSchedule[i].Date > 0 {
			UFSQuery = fmt.Sprintf(`%sDate = %d`, UFSQuery, unavailabilitiesForSchedule[i].Date)
		}
		UFSQuery = fmt.Sprintf(`%s)`, UFSQuery)
		if i+1 < len(unavailabilitiesForSchedule) {
			UFSQuery = fmt.Sprintf(`%s or `, UFSQuery)
		}
		//fmt.Println(count)
		//fmt.Println(UFSQuery)
	}
	if len(unavailabilitiesForSchedule) > 0 {
		UFSQuery = fmt.Sprintf(`%s)`, UFSQuery)
	}
	//fmt.Println(UFSQuery)
	var result []unavailabilityForSchedule
	rows, err := vsam.DB.Query(UFSQuery)
	if err != nil {
		return []unavailabilityForSchedule{}, fmt.Errorf("error in RequestUFS: sql.DB.Query error: %w. Value of UFSQuery is `%s`", err, UFSQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var UFSStruct unavailabilityForSchedule
		err = rows.Scan(&UFSStruct.UFSID, &UFSStruct.User, &UFSStruct.VolunteerForSchedule, &UFSStruct.Date)
		if err != nil {
			return []unavailabilityForSchedule{}, fmt.Errorf("error in RequestUFS: sql.Rows.Scan error: %w. Value of UFSStruct is `%+v`", err, UFSStruct)
		}
		result = append(result, UFSStruct)
	}
	err = rows.Err()
	if err != nil {
		return []unavailabilityForSchedule{}, fmt.Errorf("error in RequestUFS: sql.Rows.Err error: %w", err)
	}
	return result, nil
}

func (vsam VSAModel) UpdateUFS(currentUser string, toUpdate []unavailabilityForSchedule) error {
	if check, failed := testEmpty(toUpdate, unavailabilityForSchedule{}); check {
		return fmt.Errorf("error in UpdateUFS: method failed because one of the values in toUpdate had an empty/default values unavailabilityForSchedule struct: %+v", failed)
	}
	head := `update UnavailabilitiesForSchedule set`
	tail := fmt.Sprintf(`where User="%s" and UFSID=?`, currentUser)
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in UpdateUFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	checkDuplicates := []unavailabilityForSchedule{}
	for _, val := range toUpdate {
		if val.UFSID == 0 {
			return fmt.Errorf("error in UpdateUFS: method failed because one of the values in toUpdate had an empty/default value for UFSID: %+v", val)
		}
		currentUFS, err := vsam.RequestUFSSingle(currentUser, unavailabilityForSchedule{UFSID: val.UFSID})
		if err != nil {
			return fmt.Errorf("error in UpdateUFS: %w", err)
		}
		if !slices.Contains(checkDuplicates, unavailabilityForSchedule{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date}) {
			checkDuplicates = append(checkDuplicates, unavailabilityForSchedule{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date})
		} else {
			return fmt.Errorf("error in UpdateUFS: method failed because at least two of the unavailabilityForSchedule structs in toUpdate would create duplicate unavailabilityForSchedule structs in the database: %+v", unavailabilityForSchedule{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date})
		}
		currentUFS.UFSID = 0
		updateUFSString := head
		count := countGTZero([]int{val.UFSID, len(val.User), val.VolunteerForSchedule, val.Date})
		count-- // This is needed because a UFSID has been provided (verified at the start of this loop).
		if count == 0 {
			return fmt.Errorf("error in UpdateUFS: method failed because only one value was provided in an unavailabilityForSchedule struct. At least two values (a UFSID and a value to update) must be provided: %+v", val)
		}
		// count is at least 1
		//fmt.Println(count)
		//fmt.Println(updateUFSString)
		if val.VolunteerForSchedule > 0 {
			updateUFSString = fmt.Sprintf(`%s VolunteerForSchedule=%d`, updateUFSString, val.VolunteerForSchedule)
			count--
			currentUFS.VolunteerForSchedule = val.VolunteerForSchedule
			if count > 0 {
				updateUFSString = fmt.Sprintf(`%s,`, updateUFSString)
			}
			//fmt.Println(count)
			//fmt.Println(updateUFSString)
		}
		if val.Date > 0 {
			updateUFSString = fmt.Sprintf(`%s Date=%d`, updateUFSString, val.Date)
			//fmt.Println(count)
			//fmt.Println(updateUFSString)
			currentUFS.Date = val.Date
		}
		updateUFSString = fmt.Sprintf(`%s %s`, updateUFSString, tail)
		//fmt.Println(count)
		//fmt.Println(updateUFSString)
		if check, err := vsam.RequestUFS(currentUser, []unavailabilityForSchedule{currentUFS}); err != nil {
			return fmt.Errorf("error in UpdateUFS: %w", err)
		} else if len(check) > 0 {
			return fmt.Errorf("error in UpdateUFS: method failed because it would create a duplicate UFS: %+v", val)
		}
		updateSchedulesStmt, err := tx.Prepare(updateUFSString)
		if err != nil {
			return fmt.Errorf("error in UpdateUFS: sql.Stmt.Prepare error: %w. Value of updateUFSString is `%s`", err, updateUFSString)
		}
		defer updateSchedulesStmt.Close()
		_, err = updateSchedulesStmt.Exec(val.UFSID)
		if err != nil {
			return fmt.Errorf("error in UpdateUFS: sql.Stmt.Exec error: %w. Value of val is `%+v`", err, val)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in UpdateUFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// Will delete UFS database entries that match the UFSID or that match the VFS and Date provided in each UFS struct. If a UFSID > 0 is provided, the values for VFS and Date are ignored for that UFS struct.
func (vsam VSAModel) DeleteUFS(currentUser string, toDelete []unavailabilityForSchedule) error {
	for _, val := range toDelete {
		if val.UFSID < 1 && (val.VolunteerForSchedule < 1 || val.Date < 1) {
			return fmt.Errorf("error in DeleteUFS: method failed because one of the unavailabilityForSchedule structs did not have a value for UFSID or VolunteerForSchedule and Date: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in DeleteUFS: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	for _, val := range toDelete {
		var deleteUFSString string
		if val.UFSID > 0 {
			deleteUFSString = fmt.Sprintf(`delete from UnavailabilitiesForSchedule where User="%s" and UFSID=%d`, currentUser, val.UFSID)
		} else {
			deleteUFSString = fmt.Sprintf(`delete from UnavailabilitiesForSchedule where User="%s" and VolunteerForSchedule=%d and Date=%d`, currentUser, val.VolunteerForSchedule, val.Date)
		}
		_, err := tx.Exec(deleteUFSString)
		if err != nil {
			return fmt.Errorf("error in DeleteUFS: sql.Tx.Exec error: %w. Value of deleteUFSString is `%s`", err, deleteUFSString)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in DeleteUFS: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// correctUFS is a slices of maps with VFS structs as keys and slices of dates containing DateIDs as values. If a UFS row is linked to a VFS, but doesn't have a matching date, delete that VFS row.
func (vsam VSAModel) CleanOrphanedUFS(currentUser string, correctUFS map[volunteerForSchedule][]date) error {
	var UFSToDelete []string
	for key, value := range correctUFS {
		if key.VFSID == 0 {
			return fmt.Errorf("error in CleanOrphanedUFS: method failed because one of the provided volunteerForSchedule structs did not have a VFSID: %+v", map[volunteerForSchedule][]date{key: value})
		}
		var dates []int
		for _, dateStruct := range value {
			if dateStruct.DateID < 1 {
				return fmt.Errorf("error in CleanOrphanedUFS: method failed because one of the provided date structs did not have a DateID: %+v", map[volunteerForSchedule][]date{key: value})
			}
			dates = append(dates, dateStruct.DateID)
		}
		ufsCheck, err := vsam.RequestUFS(currentUser, []unavailabilityForSchedule{{VolunteerForSchedule: key.VFSID}})
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedUFS: %w", err)
		}
		for _, ufs := range ufsCheck {
			if !slices.Contains(dates, ufs.Date) {
				UFSToDelete = append(UFSToDelete, strconv.Itoa(ufs.UFSID))
				//log.Printf("UFSToDelete=`%#v`; ufs.Date=`%d", UFSToDelete, ufs.Date)
			}
		}
		tx, err := vsam.DB.Begin()
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedUFS: sql.DB.Begin error: %w", err)
		}
		defer tx.Rollback()
		deleteUFSQuery := fmt.Sprintf(`delete from UnavailabilitiesForSchedule where User = "%s" and UFSID in (%s)`, currentUser, CsvSlice(UFSToDelete, true))
		//fmt.Println(deleteUFSQuery)
		_, err = tx.Exec(deleteUFSQuery)
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedUFS: sql.Tx.Exec error: %w. Value of deleteUFSQuery is `%s`", err, deleteUFSQuery)
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedUFS: sql.Tx.Commit error: %w", err)
		}
	}
	return nil
}

func (vsam VSAModel) CreateSVOD(currentUser string, toCreate []scheduledVolunteerOnDate) error { // TODO
	check, err := vsam.RequestSVOD(currentUser, toCreate)
	if err != nil {
		return fmt.Errorf("error in CreateSVOD: %w", err)
	}
	if len(check) > 0 {
		return fmt.Errorf("error in CreateSVOD: method failed because at least one of the scheduledVolunteerOnDate entries to be created already exists in the database. Existing scheduledVolunteerOnDate entry(s): %+v", check)
	}
	checkDuplicates := []scheduledVolunteerOnDate{}
	for _, val := range toCreate { // User and SVODID do not need to be provided in the scheduledVolunteerOnDate structs
		if val.VolunteerForSchedule == (scheduledVolunteerOnDate{}.VolunteerForSchedule) {
			return fmt.Errorf("error in CreateSVOD: method failed because at least one of the scheduledVolunteerOnDate structs in toCreate did not have a value for VolunteerForSchedule: %+v", val)
		}
		if val.Date == (scheduledVolunteerOnDate{}.Date) {
			return fmt.Errorf("error in CreateSVOD: method failed because at least one of the scheduledVolunteerOnDate structs in toCreate did not have a value for Date: %+v", val)
		}
		if !slices.Contains(checkDuplicates, scheduledVolunteerOnDate{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date}) {
			checkDuplicates = append(checkDuplicates, scheduledVolunteerOnDate{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date})
		} else {
			return fmt.Errorf("error in CreateSVOD: method failed because at least one of the scheduledVolunteerOnDate structs in toCreate was a duplicate of another scheduledVolunteerOnDate struct in toCreate: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in CreateSVOD: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	fillSVODTableString := `insert into ScheduledVolunteersOnDates (User, VolunteerForSchedule, Date) values (?, ?, ?)`
	fillVFSTableStmt, err := tx.Prepare(fillSVODTableString)
	if err != nil {
		return fmt.Errorf("error in CreateSVOD: sql.Tx.Prepare error: %w. Value of fillSVODTableString is `%s`", err, fillSVODTableString)
	}
	defer fillVFSTableStmt.Close()
	for i := 0; i < len(toCreate); i++ {
		_, err = fillVFSTableStmt.Exec(currentUser, toCreate[i].VolunteerForSchedule, toCreate[i].Date)
		if err != nil {
			return fmt.Errorf("error in CreateSVOD: sql.Stmt.Exec error: %w. Value of toCreate[i] is `%+v`", err, toCreate[i])
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in CreateSVOD: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) RequestSVODSingle(currentUser string, scheduledVolunteerOnDateStruct scheduledVolunteerOnDate) (scheduledVolunteerOnDate, error) { // TODO
	scheduledVolunteersOnDates, err := vsam.RequestSVOD(currentUser, []scheduledVolunteerOnDate{scheduledVolunteerOnDateStruct})
	if err != nil {
		return scheduledVolunteerOnDate{}, fmt.Errorf("error in RequestSVODSingle: %w", err)
	}
	if len(scheduledVolunteersOnDates) != 1 {
		return scheduledVolunteerOnDate{}, fmt.Errorf("error in RequestSVODSingle: method failed to locate exactly one SVOD matching %+v. Found %d matches", scheduledVolunteerOnDateStruct, len(scheduledVolunteersOnDates))
	}
	return scheduledVolunteersOnDates[0], nil
}

func (vsam VSAModel) RequestSVOD(currentUser string, scheduledVolunteersOnDates []scheduledVolunteerOnDate) ([]scheduledVolunteerOnDate, error) { // TODO
	SVODQuery := fmt.Sprintf(`select * from scheduledVolunteersOnDates where User = "%s"`, currentUser)
	if len(scheduledVolunteersOnDates) > 0 {
		if check, failed := testEmpty(scheduledVolunteersOnDates, scheduledVolunteerOnDate{}); check {
			return []scheduledVolunteerOnDate{}, fmt.Errorf("error in RequestSVOD: method failed because one of the values in scheduledVolunteersOnDates had an empty/default values volunteerForSchedule struct: %+v", failed)
		}
		SVODQuery = fmt.Sprintf(`%s and (`, SVODQuery)
	}
	for i := 0; i < len(scheduledVolunteersOnDates); i++ {
		count := countGTZero([]int{scheduledVolunteersOnDates[i].SVODID, len(scheduledVolunteersOnDates[i].User), scheduledVolunteersOnDates[i].VolunteerForSchedule, scheduledVolunteersOnDates[i].Date})
		SVODQuery = fmt.Sprintf(`%s(`, SVODQuery)
		if scheduledVolunteersOnDates[i].SVODID > 0 {
			SVODQuery = fmt.Sprintf(`%sSVODID = %d`, SVODQuery, scheduledVolunteersOnDates[i].SVODID)
			count--
			if count > 0 {
				SVODQuery = fmt.Sprintf(`%s and `, SVODQuery)
			}
		}
		if len(scheduledVolunteersOnDates[i].User) > 0 {
			SVODQuery = fmt.Sprintf(`%sUser = "%s"`, SVODQuery, scheduledVolunteersOnDates[i].User)
			count--
			if count > 0 {
				SVODQuery = fmt.Sprintf(`%s and `, SVODQuery)
			}
		}
		if scheduledVolunteersOnDates[i].VolunteerForSchedule > 0 {
			SVODQuery = fmt.Sprintf(`%sVolunteerForSchedule = %d`, SVODQuery, scheduledVolunteersOnDates[i].VolunteerForSchedule)
			count--
			if count > 0 {
				SVODQuery = fmt.Sprintf(`%s and `, SVODQuery)
			}
		}
		if scheduledVolunteersOnDates[i].Date > 0 {
			SVODQuery = fmt.Sprintf(`%sDate = %d`, SVODQuery, scheduledVolunteersOnDates[i].Date)
		}
		SVODQuery = fmt.Sprintf(`%s)`, SVODQuery)
		if i+1 < len(scheduledVolunteersOnDates) {
			SVODQuery = fmt.Sprintf(`%s or `, SVODQuery)
		}
		//fmt.Println(count)
		//fmt.Println(SVODQuery)
	}
	if len(scheduledVolunteersOnDates) > 0 {
		SVODQuery = fmt.Sprintf(`%s)`, SVODQuery)
	}
	//fmt.Println(SVODQuery)
	var result []scheduledVolunteerOnDate
	rows, err := vsam.DB.Query(SVODQuery)
	if err != nil {
		return []scheduledVolunteerOnDate{}, fmt.Errorf("error in RequestSVOD: sql.DB.Query error: %w. Value of SVODQuery is `%s`", err, SVODQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var SVODStruct scheduledVolunteerOnDate
		err = rows.Scan(&SVODStruct.SVODID, &SVODStruct.User, &SVODStruct.VolunteerForSchedule, &SVODStruct.Date)
		if err != nil {
			return []scheduledVolunteerOnDate{}, fmt.Errorf("error in RequestSVOD: sql.Rows.Scan error: %w. Value of SVODStruct is `%+v`", err, SVODStruct)
		}
		result = append(result, SVODStruct)
	}
	err = rows.Err()
	if err != nil {
		return []scheduledVolunteerOnDate{}, fmt.Errorf("error in RequestSVOD: sql.Rows.Err error: %w", err)
	}
	return result, nil
}

func (vsam VSAModel) UpdateSVOD(currentUser string, toUpdate []scheduledVolunteerOnDate) error { // TODO
	if check, failed := testEmpty(toUpdate, scheduledVolunteerOnDate{}); check {
		return fmt.Errorf("error in UpdateSVOD: method failed because one of the values in toUpdate had an empty/default values scheduledVolunteerOnDate struct: %+v", failed)
	}
	head := `update scheduledVolunteersOnDates set`
	tail := fmt.Sprintf(`where User="%s" and SVODID=?`, currentUser)
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in UpdateSVOD: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	checkDuplicates := []scheduledVolunteerOnDate{}
	for _, val := range toUpdate {
		if val.SVODID == 0 {
			return fmt.Errorf("error in UpdateSVOD: method failed because one of the values in toUpdate had an empty/default value for SVODID: %+v", val)
		}
		currentSVOD, err := vsam.RequestSVODSingle(currentUser, scheduledVolunteerOnDate{SVODID: val.SVODID})
		if err != nil {
			return fmt.Errorf("error in UpdateSVOD: %w", err)
		}
		if !slices.Contains(checkDuplicates, scheduledVolunteerOnDate{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date}) {
			checkDuplicates = append(checkDuplicates, scheduledVolunteerOnDate{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date})
		} else {
			return fmt.Errorf("error in UpdateSVOD: method failed because at least two of the scheduledVolunteerOnDate structs in toUpdate would create duplicate scheduledVolunteerOnDate structs in the database: %+v", scheduledVolunteerOnDate{VolunteerForSchedule: val.VolunteerForSchedule, Date: val.Date})
		}
		currentSVOD.SVODID = 0
		updateSVODString := head
		count := countGTZero([]int{val.SVODID, len(val.User), val.VolunteerForSchedule, val.Date})
		count-- // This is needed because a SVODID has been provided (verified at the start of this loop).
		if count == 0 {
			return fmt.Errorf("error in UpdateSVOD: method failed because only one value was provided in an scheduledVolunteerOnDate struct. At least two values (a SVODID and a value to update) must be provided: %+v", val)
		}
		// count is at least 1
		//fmt.Println(count)
		//fmt.Println(updateSVODString)
		if val.VolunteerForSchedule > 0 {
			updateSVODString = fmt.Sprintf(`%s VolunteerForSchedule=%d`, updateSVODString, val.VolunteerForSchedule)
			count--
			currentSVOD.VolunteerForSchedule = val.VolunteerForSchedule
			if count > 0 {
				updateSVODString = fmt.Sprintf(`%s,`, updateSVODString)
			}
			//fmt.Println(count)
			//fmt.Println(updateSVODString)
		}
		if val.Date > 0 {
			updateSVODString = fmt.Sprintf(`%s Date=%d`, updateSVODString, val.Date)
			//fmt.Println(count)
			//fmt.Println(updateSVODString)
			currentSVOD.Date = val.Date
		}
		updateSVODString = fmt.Sprintf(`%s %s`, updateSVODString, tail)
		//fmt.Println(count)
		//fmt.Println(updateSVODString)
		if check, err := vsam.RequestSVOD(currentUser, []scheduledVolunteerOnDate{currentSVOD}); err != nil {
			return fmt.Errorf("error in UpdateSVOD: %w", err)
		} else if len(check) > 0 {
			return fmt.Errorf("error in UpdateSVOD: method failed because it would create a duplicate SVOD: %+v", val)
		}
		updateSchedulesStmt, err := tx.Prepare(updateSVODString)
		if err != nil {
			return fmt.Errorf("error in UpdateSVOD: sql.Stmt.Prepare error: %w. Value of updateSVODString is `%s`", err, updateSVODString)
		}
		defer updateSchedulesStmt.Close()
		_, err = updateSchedulesStmt.Exec(val.SVODID)
		if err != nil {
			return fmt.Errorf("error in UpdateSVOD: sql.Stmt.Exec error: %w. Value of val is `%+v`", err, val)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in UpdateSVOD: sql.Tx.Commit error: %w", err)
	}
	return nil
}

// Will delete SVOD database entries that match the SVODID or that match the VFS and Date provided in each SVOD struct. If a SVODID > 0 is provided, the values for VFS and Date are ignored for that SVOD struct.
func (vsam VSAModel) DeleteSVOD(currentUser string, toDelete []scheduledVolunteerOnDate) error { // TODO
	for _, val := range toDelete {
		if val.SVODID < 1 && (val.VolunteerForSchedule < 1 || val.Date < 1) {
			return fmt.Errorf("error in DeleteSVOD: method failed because one of the scheduledVolunteerOnDate structs did not have a value for SVODID or VolunteerForSchedule and Date: %+v", val)
		}
	}
	tx, err := vsam.DB.Begin()
	if err != nil {
		return fmt.Errorf("error in DeleteSVOD: sql.DB.Begin error: %w", err)
	}
	defer tx.Rollback()
	for _, val := range toDelete {
		var deleteSVODString string
		if val.SVODID > 0 {
			deleteSVODString = fmt.Sprintf(`delete from scheduledVolunteersOnDates where User="%s" and SVODID=%d`, currentUser, val.SVODID)
		} else {
			deleteSVODString = fmt.Sprintf(`delete from scheduledVolunteersOnDates where User="%s" and VolunteerForSchedule=%d and Date=%d`, currentUser, val.VolunteerForSchedule, val.Date)
		}
		_, err := tx.Exec(deleteSVODString)
		if err != nil {
			return fmt.Errorf("error in DeleteSVOD: sql.Tx.Exec error: %w. Value of deleteSVODString is `%s`", err, deleteSVODString)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error in DeleteSVOD: sql.Tx.Commit error: %w", err)
	}
	return nil
}

func (vsam VSAModel) CleanOrphanedSVOD(currentUser string, correctSVOD map[volunteerForSchedule][]date) error {
	var SVODToDelete []string
	for key, value := range correctSVOD {
		if key.VFSID == 0 {
			return fmt.Errorf("error in CleanOrphanedSVOD: method failed because one of the provided volunteerForSchedule structs did not have a VFSID: %+v", map[volunteerForSchedule][]date{key: value})
		}
		var dates []int
		for _, dateStruct := range value {
			if dateStruct.DateID < 1 {
				return fmt.Errorf("error in CleanOrphanedSVOD: method failed because one of the provided date structs did not have a DateID: %+v", map[volunteerForSchedule][]date{key: value})
			}
			dates = append(dates, dateStruct.DateID)
		}
		SVODCheck, err := vsam.RequestSVOD(currentUser, []scheduledVolunteerOnDate{{VolunteerForSchedule: key.VFSID}})
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedSVOD: %w", err)
		}
		for _, SVOD := range SVODCheck {
			if !slices.Contains(dates, SVOD.Date) {
				SVODToDelete = append(SVODToDelete, strconv.Itoa(SVOD.SVODID))
				//fmt.Println(SVODToDelete)
			}
		}
		tx, err := vsam.DB.Begin()
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedSVOD: sql.DB.Begin error: %w", err)
		}
		defer tx.Rollback()
		deleteSVODQuery := fmt.Sprintf(`delete from scheduledVolunteersOnDates where User = "%s" and SVODID in (%s)`, currentUser, CsvSlice(SVODToDelete, true))
		//fmt.Println(deleteSVODQuery)
		_, err = tx.Exec(deleteSVODQuery)
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedSVOD: sql.Tx.Exec error: %w. Value of deleteSVODQuery is `%s`", err, deleteSVODQuery)
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("error in CleanOrphanedSVOD: sql.Tx.Commit error: %w", err)
		}
	}
	return nil
}

/*
Weekdays, Months, and Dates are readonly.
What data will be requested by the app?
	List of schedule names for a user;
	A schedule joined with its UFS (joined with its VFS (joined with its Volunteers)) and its WFS for a user;
	Completed schedules represented in the database as a schedule joined with its SVOD (joined with its VFS (joined with its Volunteers)) for a user;
What data will be sent by the app?
	A schedule struct including all the data needed to create/update rows on Schedules, Volunteers, WFS, VFS, and UFS
	A completed schedule struct to create new rows in SVOD

This does not contemplate CRUDing users yet.
*/

func FillInSampleDB(currentUser string, DbModel VSAModel) {
	schedules := []schedule{
		{
			ScheduleName:       "test1",
			ShiftsOff:          3,
			VolunteersPerShift: 3,
			StartDate:          Must(DbModel.RequestDate(date{Month: 1, Day: 1, Year: 2024})).DateID,
			EndDate:            Must(DbModel.RequestDate(date{Month: 3, Day: 1, Year: 2024})).DateID,
			User:               currentUser,
		},
		{
			ScheduleName:       "test2",
			ShiftsOff:          3,
			VolunteersPerShift: 3,
			StartDate:          Must(DbModel.RequestDate(date{Month: 3, Day: 1, Year: 2024})).DateID,
			EndDate:            Must(DbModel.RequestDate(date{Month: 6, Day: 1, Year: 2024})).DateID,
			User:               currentUser,
		},
		{
			ScheduleName:       "test3",
			ShiftsOff:          3,
			VolunteersPerShift: 3,
			StartDate:          Must(DbModel.RequestDate(date{Month: 6, Day: 1, Year: 2024})).DateID,
			EndDate:            Must(DbModel.RequestDate(date{Month: 9, Day: 1, Year: 2024})).DateID,
			User:               currentUser,
		},
	}
	DbModel.CreateSchedules(currentUser, schedules)
	weekdaysForSchedule := []weekdayForSchedule{
		{
			User:     currentUser,
			Weekday:  Must(DbModel.RequestWeekday(weekday{WeekdayName: "Sunday"})).WeekdayName,
			Schedule: Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
		},
		{
			User:     currentUser,
			Weekday:  Must(DbModel.RequestWeekday(weekday{WeekdayName: "Wednesday"})).WeekdayName,
			Schedule: Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
		},
		{
			User:     currentUser,
			Weekday:  Must(DbModel.RequestWeekday(weekday{WeekdayName: "Friday"})).WeekdayName,
			Schedule: Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
		},
	}
	DbModel.CreateWFS(currentUser, weekdaysForSchedule)
	volunteers := []volunteer{
		{
			VolunteerName: "Tim",
			User:          currentUser,
		},
		{
			VolunteerName: "Bill",
			User:          currentUser,
		},
		{
			VolunteerName: "Jack",
			User:          currentUser,
		},
		{
			VolunteerName: "George",
			User:          currentUser,
		},
		{
			VolunteerName: "Bob",
			User:          currentUser,
		},
		{
			VolunteerName: "Lance",
			User:          currentUser,
		},
		{
			VolunteerName: "Larry",
			User:          currentUser,
		},
	}
	DbModel.CreateVolunteers(currentUser, volunteers)
	volunteersForSchedule := []volunteerForSchedule{
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Tim"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Jack"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "George"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bob"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Lance"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Larry"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Tim"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Jack"})).VolunteerID,
		},
		{
			User:      currentUser,
			Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
			Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "George"})).VolunteerID,
		},
	}
	DbModel.CreateVFS(currentUser, volunteersForSchedule)
	unavailabilitiesForSchedule := []unavailabilityForSchedule{
		{
			User: currentUser,
			VolunteerForSchedule: Must(DbModel.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
				Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Tim"})).VolunteerID,
			})).VFSID,
			Date: Must(DbModel.RequestDate(date{Month: 1, Day: 14, Year: 2024})).DateID,
		},
		{
			User: currentUser,
			VolunteerForSchedule: Must(DbModel.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test1"})).ScheduleID,
				Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bill"})).VolunteerID,
			})).VFSID,
			Date: Must(DbModel.RequestDate(date{Month: 1, Day: 21, Year: 2024})).DateID,
		},
		{
			User: currentUser,
			VolunteerForSchedule: Must(DbModel.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
				Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Bob"})).VolunteerID,
			})).VFSID,
			Date: Must(DbModel.RequestDate(date{Month: 5, Day: 12, Year: 2024})).DateID,
		},
		{
			User: currentUser,
			VolunteerForSchedule: Must(DbModel.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test2"})).ScheduleID,
				Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Lance"})).VolunteerID,
			})).VFSID,
			Date: Must(DbModel.RequestDate(date{Month: 5, Day: 19, Year: 2024})).DateID,
		},
		{
			User: currentUser,
			VolunteerForSchedule: Must(DbModel.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
				Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "Jack"})).VolunteerID,
			})).VFSID,
			Date: Must(DbModel.RequestDate(date{Month: 8, Day: 11, Year: 2024})).DateID,
		},
		{
			User: currentUser,
			VolunteerForSchedule: Must(DbModel.RequestVFSSingle(currentUser, volunteerForSchedule{
				Schedule:  Must(DbModel.RequestSchedule(currentUser, schedule{ScheduleName: "test3"})).ScheduleID,
				Volunteer: Must(DbModel.RequestVolunteer(currentUser, volunteer{VolunteerName: "George"})).VolunteerID,
			})).VFSID,
			Date: Must(DbModel.RequestDate(date{Month: 8, Day: 18, Year: 2024})).DateID,
		},
	}
	DbModel.CreateUFS(currentUser, unavailabilitiesForSchedule)
}

func main() {
	dbExists := false
	if _, err := os.Stat(DbName); err == nil {
		dbExists = true
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", DbName))
	if err != nil {
		log.Fatal(err)
	}
	env := &SampleEnv{
		Sample:       VSAModel{DB: db},
		LoggedInUser: "Seth",
	}
	defer env.Sample.DB.Close()
	if !dbExists {
		if err = env.Sample.CreateDatabase(); err != nil {
			log.Fatalf("Crashed in main() with error: %v", err)
		}
	}
	FillInSampleDB(env.LoggedInUser, env.Sample)
	fmt.Println("Done. Press enter to exit executable.")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Print(month{}, user{})
}

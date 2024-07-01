package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/codingsince1985/geo-golang"
	"github.com/codingsince1985/geo-golang/openstreetmap"
	simpleMap "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/xuri/excelize/v2"
	"log"
	"math/rand"
	"time"
)

var db *sql.DB

////go:embed GUIDataBase.db
//var dbData []byte

var AllJobs []JobInfo
var MainWindow JobWindow
var LocationCache map[string]*geo.Location
var JobCounts map[string]int

//const memory = ":memory:"

type JobWindow struct {
	DataDisplay            *widget.List
	CompanyDisplay         *widget.Entry
	PostingDateDisplay     *widget.Entry
	JobIDDisplay           *widget.Entry
	CountryDisplay         *widget.Entry
	LocationDisplay        *widget.Entry
	PublicationDateDisplay *widget.Entry
	SalaryMaxDisplay       *widget.Entry
	SalaryMinDisplay       *widget.Entry
	SalaryTypeDisplay      *widget.RadioGroup
	JobTitleDisplay        *widget.Entry
	CurrentSelection       int
}

type JobInfo struct {
	CompanyName     string
	PostingDate     string
	JobID           string
	Country         string
	Location        string
	PublicationDate string
	SalaryMax       string
	SalaryMin       string
	SalaryType      string
	JobTitle        string
}

func main() {
	// Open SQLite database connection
	var err error
	db, err = sql.Open("sqlite3", "./GUIDataBase.db")
	//file::memory:?cache=shared
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//// Load embedded SQLite database
	//_, err = db.Exec(string(dbData))
	//if err != nil {
	//	log.Fatal(err)
	//}

	// Create jobs table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Project3Data (
		company_name TEXT,
		posting_date TEXT,
		job_id TEXT,
		country TEXT,
		location TEXT,
		publication_date TEXT,
		salary_max TEXT,
		salary_min TEXT,
		salary_type TEXT,
		job_title TEXT
	)`)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("SQL database is ready to use!")

	// Read data from Excel file
	excelData := GetData("Project3Data.xlsx")

	LocationCache = make(map[string]*geo.Location)
	JobCounts = make(map[string]int)
	mapCenter := findLocation("Columbus, OH")
	bringUpMap(excelData, mapCenter)

	// Process and insert data into SQLite database
	for _, excelLine := range excelData {
		job := JobInfo{
			CompanyName:     excelLine[0],
			PostingDate:     excelLine[1],
			JobID:           excelLine[2],
			Country:         excelLine[3],
			Location:        excelLine[4],
			PublicationDate: excelLine[5],
			SalaryMax:       excelLine[6],
			SalaryMin:       excelLine[7],
			SalaryType:      excelLine[8],
			JobTitle:        excelLine[9],
		}
		log.Printf("Processing job: %+v\n", job)

		err := InsertJob(job)
		if err != nil {
			log.Println("Error inserting job:", err)
		}
		AllJobs = append(AllJobs, job)
	}
	fmt.Println("Data has been successfully imported into sqlite3 database!")

	// Initialize the GUI
	MainWindow = JobWindow{}
	app := app.New()
	fyneWindow := app.NewWindow("Your Next Job?")
	makeJobWindow(&MainWindow, fyneWindow)
	fyneWindow.ShowAndRun()
}

func GetData(fileName string) [][]string {
	excelFile, err := excelize.OpenFile(fileName)
	if err != nil {
		log.Fatal("File cannot open.", err)
	}
	defer excelFile.Close()

	allRows, err := excelFile.GetRows("Comp490 Jobs")
	if err != nil {
		log.Fatal(err)
	}
	return allRows
}

func InsertJob(job JobInfo) error {
	insertStatement, err := db.Prepare(`INSERT INTO Project3Data (
		CompanyName, PostingAge, JobId, Country, Location, PublicationDate, SalaryMax, SalaryMin, SalaryType, JobTitle
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertStatement.Close()

	_, err = insertStatement.Exec(
		job.CompanyName, job.PostingDate, job.JobID, job.Country, job.Location,
		job.PublicationDate, job.SalaryMax, job.SalaryMin, job.SalaryType, job.JobTitle,
	)
	if err != nil {
		return err
	}

	return nil
}

func makeJobWindow(jobDisplay *JobWindow, window fyne.Window) {
	// Create the list to display job data
	jobDisplay.DataDisplay = widget.NewList(GetNumJobs, CreateListItem, UpdateListItem)

	// Create the right pane for job details and buttons
	rightPane := container.NewVBox()
	nextLabel := widget.NewLabel("Company:")
	jobDisplay.CompanyDisplay = widget.NewEntry()
	rightPane.Add(nextLabel)
	rightPane.Add(jobDisplay.CompanyDisplay)

	// Add buttons for actions
	buttonZone := container.NewGridWithColumns(3)
	saveButton := widget.NewButton("Save", saveJob)
	deleteButton := widget.NewButton("Delete", deleteJob)
	updateButton := widget.NewButton("Update", updateJob)
	buttonZone.Add(saveButton)
	buttonZone.Add(deleteButton)
	buttonZone.Add(updateButton)

	// Combine the right pane and button zone
	content := container.NewBorder(nil, nil, nil, buttonZone, rightPane)

	// combine the list and right pane
	contentPane := container.NewHSplit(jobDisplay.DataDisplay, content)
	window.SetContent(contentPane)
	window.Resize(fyne.NewSize(1000, 900))
}

func saveJob() {
	newJob := JobInfo{
		CompanyName:     MainWindow.CompanyDisplay.Text,
		PostingDate:     time.Now().Format("2006-01-02"), // Assuming format "YYYY-MM-DD"
		JobID:           RandStringRunes(20),
		Country:         MainWindow.CountryDisplay.Text,
		Location:        MainWindow.LocationDisplay.Text,
		PublicationDate: fmt.Sprintf("%d", time.Now().Unix()),
		SalaryMax:       MainWindow.SalaryMaxDisplay.Text,
		SalaryMin:       MainWindow.SalaryMinDisplay.Text,
		SalaryType:      MainWindow.SalaryTypeDisplay.Selected,
		JobTitle:        MainWindow.JobTitleDisplay.Text,
	}
	err := InsertJob(newJob)
	if err != nil {
		log.Println("Error in inserting job:", err)
		return
	}
	AllJobs = append(AllJobs, newJob)
	MainWindow.DataDisplay.Refresh()
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func deleteJob() {
	if MainWindow.CurrentSelection >= 0 && MainWindow.CurrentSelection < len(AllJobs) {
		err := DeleteJob(AllJobs[MainWindow.CurrentSelection].JobID)
		if err != nil {
			log.Println("Error in deleting job:", err)
			return
		}
		AllJobs = append(AllJobs[:MainWindow.CurrentSelection], AllJobs[MainWindow.CurrentSelection+1:]...)
		MainWindow.CurrentSelection = 0
		MainWindow.DataDisplay.Select(MainWindow.CurrentSelection)
		MainWindow.DataDisplay.Refresh()
	}
}

func updateJob() {
	if MainWindow.CurrentSelection >= 0 && MainWindow.CurrentSelection < len(AllJobs) {
		updateJob := JobInfo{
			CompanyName:     MainWindow.CountryDisplay.Text,
			PostingDate:     time.Now().Format("2006-01-02"), // Assuming format "YYYY-MM-DD"
			JobID:           AllJobs[MainWindow.CurrentSelection].JobID,
			Country:         MainWindow.CountryDisplay.Text,
			Location:        MainWindow.LocationDisplay.Text,
			PublicationDate: fmt.Sprintf("%d", time.Now().Unix()),
			SalaryMax:       MainWindow.SalaryMaxDisplay.Text,
			SalaryMin:       MainWindow.SalaryMinDisplay.Text,
			SalaryType:      MainWindow.SalaryTypeDisplay.Selected,
			JobTitle:        MainWindow.JobTitleDisplay.Text,
		}
		err := UpdateJob(updateJob)
		if err != nil {
			log.Println("Error in updating job:", err)
			return
		}
		AllJobs[MainWindow.CurrentSelection] = updateJob
		MainWindow.DataDisplay.Refresh()
	}
}

func DeleteJob(jobID string) error {
	_, err := db.Exec("DELETE FROM Project3Data WHERE JobId = ?", jobID)
	return err
}

func UpdateJob(job JobInfo) error {
	_, err := db.Exec(`UPDATE Project3Data SET CompanyName = ?, PostingAge = ?, Project3Data.Country = ?, Location = ?, 
                PublicationDate = ?, SalaryMax = ?, SalaryMin = ?, SalaryType = ?, JobTitle = ? WHERE JobId = ?`,
		job.CompanyName, job.PostingDate, job.Country, job.Location, job.PublicationDate, job.SalaryMax, job.SalaryMin,
		job.SalaryType, job.JobTitle, job.JobID)
	return err
}

func GetNumJobs() int {
	return len(AllJobs)
}

func CreateListItem() fyne.CanvasObject {
	return widget.NewLabel("Job")
}

func UpdateListItem(i widget.ListItemID, item fyne.CanvasObject) {
	label := item.(*widget.Label)
	label.SetText(AllJobs[i].JobTitle)
}

func bringUpMap(data [][]string, loc *geo.Location) {
	processData(data, loc)
	context := simpleMap.NewContext()
	context.SetSize(1000, 1000)
	context.SetZoom(7)
	for city, numJobs := range JobCounts {
		cityLoc := LocationCache[city]
		pinColor := gotColor(numJobs)
		context.AddObject(
			simpleMap.NewMarker(
				s2.LatLngFromDegrees(cityLoc.Lat, cityLoc.Lng),
				pinColor,
				16,
			),
		)
	}

	context.SetCenter(s2.LatLngFromDegrees(loc.Lat, loc.Lng))
	image, err := context.Render()
	if err != nil {
		log.Fatal(err)
	}
	if err = gg.SavePNG("ITJobs.png", image); err != nil {
		panic(err)
	}

}

func gotColor(numberOfjobs int) string {
	if numberOfjobs > 75 {
		return "#00FF00" // Green
	}
	if numberOfjobs > 50 {
		return "#00FFFF" // Cyan
	}
	if numberOfjobs > 25 {
		return "#0000FF" // Blue
	}
	if numberOfjobs > 10 {
		return "#FF0000" // Red
	}
	if numberOfjobs > 5 {
		return "#FF00FF" // Magenta
	}
	if numberOfjobs > 1 {
		return "#101010" // Black
	}
	return "#FFFFF10" // Yellow
}

func processData(data [][]string, defaultLocation *geo.Location) {
	for rowNumber, job := range data {
		if rowNumber < 1 {
			continue
		}
		cityName := job[4]
		_, ok := LocationCache[cityName]
		if ok {
			JobCounts[cityName]++
		} else {
			loc := findLocation(cityName)
			if loc == nil {
				loc = defaultLocation
			}
			LocationCache[cityName] = loc
			JobCounts[cityName] = 1
		}
	}

}

func findLocation(city string) *geo.Location {
	geoLookup := openstreetmap.Geocoder()
	locationData, err := geoLookup.Geocode(city)
	if err != nil {
		log.Println("Error looking up location:", city, err)
	}
	return locationData
}



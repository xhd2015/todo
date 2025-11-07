package tool

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/xhd2015/go-dom-tui/charm/renderer"
	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/todo/component/chart"
)

const help = `
plot - Plot data in a chart

Options:
  -f,--data <file>  The data file to plot (JSON format)
  --random          Generate random data for 365 days (cannot be used with -f)
  -h,--help         Show this help message

Data Format:
  The JSON file should contain an array of data points:
  [
    {"x": "Label1", "y": 10.5},
    {"x": "Label2", "y": 12.3},
    {"x": "Label3", "y": 11.0}
  ]
`

func Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("requires sub command: plot")
	}
	cmd := args[0]
	args = args[1:]
	if cmd == "--help" || cmd == "help" {
		fmt.Print(strings.TrimPrefix(help, "\n"))
		return nil
	}
	switch cmd {
	case "plot":
		return handlePlot(args)
	default:
		return fmt.Errorf("unrecognized: %s", cmd)
	}
}

func handlePlot(args []string) error {
	var dataFile string
	var useRandom bool
	remainingArgs, err := flags.String("-f,--data", &dataFile).
		Bool("--random", &useRandom).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	// Guard clause: check for extra arguments
	if len(remainingArgs) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(remainingArgs, " "))
	}

	// Guard clause: check that -f and --random are not used together
	if dataFile != "" && useRandom {
		return fmt.Errorf("-f/--data and --random cannot be used together")
	}

	// Guard clause: check that at least one option is provided
	if dataFile == "" && !useRandom {
		return fmt.Errorf("either -f/--data or --random must be specified")
	}

	var dataPoints []chart.DataPoint
	var title string

	if useRandom {
		// Generate random data for 365 days
		dataPoints = generateRandomData()
		title = "Random Data (365 days)"
	} else {
		// Load data from file
		// Guard clause: check if file exists
		if _, err := os.Stat(dataFile); os.IsNotExist(err) {
			return fmt.Errorf("data file not found: %s", dataFile)
		}

		// Read the JSON file
		fileData, err := os.ReadFile(dataFile)
		if err != nil {
			return fmt.Errorf("failed to read data file: %w", err)
		}

		// Parse JSON into data points
		if err := json.Unmarshal(fileData, &dataPoints); err != nil {
			return fmt.Errorf("failed to parse JSON data: %w", err)
		}

		// Guard clause: check if data is empty
		if len(dataPoints) == 0 {
			fmt.Println("Warning: No data points found in file")
		}

		title = fmt.Sprintf("Data from %s", dataFile)
	}

	// Create and render the chart
	chartNode := chart.LineChart(chart.LineChartProps{
		Data:   dataPoints,
		Width:  80,
		Height: 20,
		Title:  title,
	})

	// Render to string
	output := renderer.RenderToString(chartNode)
	fmt.Print(output)

	return nil
}

// generateRandomData generates 365 days of random data points
func generateRandomData() []chart.DataPoint {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	dataPoints := make([]chart.DataPoint, 365)
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Generate random walk data with some trend
	baseValue := 50.0
	currentValue := baseValue

	for i := 0; i < 365; i++ {
		date := startDate.AddDate(0, 0, i)

		// Random walk with trend
		change := (rand.Float64() - 0.5) * 10.0 // Random change between -5 and +5
		currentValue += change

		// Add seasonal variation
		dayOfYear := float64(i)
		seasonal := 20.0 * (1.0 - ((dayOfYear-182.5)/182.5)*((dayOfYear-182.5)/182.5))
		currentValue += seasonal * 0.1

		// Keep value in reasonable range
		if currentValue < 10 {
			currentValue = 10
		}
		if currentValue > 100 {
			currentValue = 100
		}

		dataPoints[i] = chart.DataPoint{
			X: date.Format("2006-01-02"),
			Y: currentValue,
		}
	}

	return dataPoints
}

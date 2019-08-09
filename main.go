package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type result int

const (
	resultNoData = iota
	resultFail
	resultSuccess
)

type dateValue struct {
	DV        map[time.Time]float64
	FirstDate time.Time
	LastDate  time.Time
}

func main() {
	// prepare history data
	sAndP500History := parseSAndP500File("./SANDP500.csv")
	usInflationHistory := parseInflationRate("./USInflationRate.csv")

	// get parameters
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter initial capital in USD: ")
	t, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("read input err: %+v", err)
	}
	initCapital := float64(0)
	_, err = fmt.Sscanf(t, "%f\n", &initCapital)
	if err != nil {
		log.Fatalf("wrong capital: %+v", err)
	}
	fmt.Print("Enter withdraw percent only digit part: ")
	t, err = reader.ReadString('\n')
	if err != nil {
		log.Fatalf("read input err: %+v", err)
	}
	withdrawPercent := float64(0)
	_, err = fmt.Sscanf(t, "%f\n", &withdrawPercent)
	if err != nil {
		log.Fatalf("wrong withdraw percent: %+v", err)
	}
	withdrawPercent /= 100
	fmt.Print("Enter num of year to spend: ")
	t, err = reader.ReadString('\n')
	if err != nil {
		log.Fatalf("read input err: %+v", err)
	}
	numOfYear := 0
	_, err = fmt.Sscanf(t, "%d\n", &numOfYear)
	if err != nil {
		log.Fatalf("wrong year: %+v", err)
	}

	success, failed, na := 0, 0, 0
	var trace strings.Builder
	current := sAndP500History.FirstDate
	if usInflationHistory.FirstDate.After(current) {
		current = usInflationHistory.FirstDate
	}
	end := time.Now()
	for current.Before(end) {
		r := runPlan(current, withdrawPercent, initCapital, usInflationHistory, sAndP500History, numOfYear, &trace)
		if r == resultSuccess {
			success++
		} else if r == resultFail {
			failed++
		} else {
			na++
		}
		current = current.AddDate(0, 0, 1)
	}
	log.Printf("success = %d, failed = %d, na = %d\n", success, failed, na)
	log.Printf("success rate = %.3f\n", float64(success)/float64(success+failed))

	fileName := "./fourpercent-" + time.Now().Format(time.RFC3339Nano)
	ioutil.WriteFile(fileName, []byte(trace.String()), os.ModePerm)

	log.Println("done")
}

func findSAndP500Price(sAndP500Day time.Time, sAndP500History dateValue) float64 {
	for {
		price, ok := sAndP500History.DV[sAndP500Day]
		if ok {
			return price
		}

		sAndP500Day = sAndP500Day.AddDate(0, 0, 1)
		if sAndP500Day.After(sAndP500History.LastDate) {
			return 0
		}
	}
}

func runPlan(startTime time.Time, withdrawPercent float64, capital float64, cpiHistory, sAndP500History dateValue, numOfYears int, trace io.Writer) result {
	io.WriteString(trace, "initial conditions ===========================\n")
	io.WriteString(trace, fmt.Sprintf("capital: %f\n", capital))
	io.WriteString(trace, fmt.Sprintf("start date: %+v\n", startTime.Format("2006-01-02")))

	withdraw := capital * withdrawPercent
	io.WriteString(trace, fmt.Sprintf("withdraw: %f\n", withdraw))
	price := findSAndP500Price(startTime, sAndP500History)
	if price == 0 {
		io.WriteString(trace, "no price for the date\n\n\n\n")
		return resultNoData
	}
	io.WriteString(trace, fmt.Sprintf("s&p 500 price of the day: %f\n", price))
	shares := int((capital - withdraw) / price)
	if shares <= 0 {
		io.WriteString(trace, "run out of money\n\n\n\n")
		return resultFail
	}
	io.WriteString(trace, fmt.Sprintf("init hold shares: %d\n", shares))
	numOfYears--

	currentTime := startTime
	for i := 0; i < numOfYears; i++ {
		io.WriteString(trace, "\n\n")

		currentTime = currentTime.AddDate(1, 0, 0)
		io.WriteString(trace, fmt.Sprintf("currentTime: %+v\n", currentTime))

		// use cpi of last month
		cpiTime := time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC)
		cpiTime = cpiTime.AddDate(0, -1, 0)
		cpi, ok := cpiHistory.DV[cpiTime]
		if !ok {
			io.WriteString(trace, "no more cpi data\n\n\n\n")
			return resultNoData
		}
		io.WriteString(trace, fmt.Sprintf("cpi of %+v: %+v\n", cpiTime, cpi))

		withdraw = withdraw * (1 + (cpi / 100))
		io.WriteString(trace, fmt.Sprintf("withdraw: %+v\n", withdraw))
		price := findSAndP500Price(currentTime, sAndP500History)
		if price == 0 {
			io.WriteString(trace, "no more s&p 500 data\n\n\n\n")
			return resultNoData
		}
		io.WriteString(trace, fmt.Sprintf("price: %+v\n", price))
		soldShare := int(math.Ceil(withdraw / price))
		io.WriteString(trace, fmt.Sprintf("sold share: %+v\n", soldShare))
		shares -= soldShare
		io.WriteString(trace, fmt.Sprintf("remain share: %+v\n", shares))

		if shares <= 0 {
			io.WriteString(trace, "run out of money\n\n\n\n")
			return resultFail
		}
	}

	io.WriteString(trace, "pass the year\n\n\n\n")
	return resultSuccess
}

func printFirstN(dv dateValue, n int) {
	count := 0
	for k, v := range dv.DV {
		log.Println(k, v)
		count++
		if count == n {
			return
		}
	}
}

func parseInflationRate(path string) dateValue {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("open inflation csv %s failed %+v", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	history := dateValue{
		DV: make(map[time.Time]float64),
	}
	numOfRow := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read inflation csv %+v", err)
		}

		numOfRow++
		if numOfRow == 1 {
			// skip header
			continue
		}

		// row[i] is the inflation rate of ith month
		for i := 1; i <= 12; i++ {
			avg, err := strconv.ParseFloat(row[i], 64)
			if err != nil {
				continue
			}
			year, _ := strconv.Atoi(row[0])
			t := time.Date(year, time.Month(i), 1, 0, 0, 0, 0, time.UTC)
			history.DV[t] = avg

			if numOfRow == 2 && history.FirstDate.Equal(time.Time{}) {
				history.FirstDate = t
			}
			history.LastDate = t
		}
	}

	return history
}

func parseSAndP500File(path string) dateValue {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("open s&p500 csv %s failed %+v", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	history := dateValue{
		DV: make(map[time.Time]float64),
	}
	numOfRow := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read s&p500 csv %+v", err)
		}

		numOfRow++
		if numOfRow == 1 {
			// skip header
			continue
		}

		// row[0]:1950-01-03
		// row[4]:16.66
		t, _ := time.Parse("2006-01-02", row[0])
		history.DV[t], _ = strconv.ParseFloat(row[4], 64)

		if numOfRow == 2 && history.FirstDate.Equal(time.Time{}) {
			history.FirstDate = t
		}
		history.LastDate = t
	}

	return history
}

package main

import (
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

type datePrice struct {
	Day2Price map[time.Time]float64
	FirstDate time.Time
	LastDate  time.Time
}

func main() {
	maxWithdrawRate := flag.Float64("max", 0, "max withdraw rate. e.g., 0.04")
	minWithdrawRate := flag.Float64("min", 0, "min withdraw rate e.g., 0.03")
	yearToSpend := flag.Int("y", 0, "year to spend. e.g., 30")
	dividendTaxRate := flag.Float64("t", 0, "tax rate of dividend. e.g., 0.2")
	flag.Parse()

	// prepare history data
	day2PersonalConsumption := readPersonalConsumption("./personal-consumption-expenditures-per-capita.csv")
	day2SAndP500Price := readSAndP500File("./spx.csv")
	year2StockDividendYield := readSAndP500DividendYield("./spx-dividend-yield.csv")
	year2InflationRate := readInflationRate("./us-inflation-rate.csv")

	var successCount, failCount float64
	var minWithdrawCount, maxWithdrawCount float64
	for day, price := range day2SAndP500Price.Day2Price {
		currMaxWithdrawRate := *maxWithdrawRate
		currMinWithdrawRate := *minWithdrawRate

		consumption, ok := findPersonalConsumption(day, day2PersonalConsumption)
		if !ok {
			continue
		}
		initSaving := consumption / currMaxWithdrawRate
		// withdraw for consumption of  first year
		saving := initSaving * (1 - currMaxWithdrawRate)
		targetSaving := saving

		const (
			resultNoData = iota
			resultSuccess
			resultFail
		)
		result := resultSuccess
		prevYearStockPrice := price
		var currMinWithdrawCount, currMaxWithdrawCount float64
		for i := 0; i < *yearToSpend; i++ {
			aYearLater := day.AddDate(1, 0, 0)
			sellDay, currStockPrice := findSAndP500DayAndPrice(aYearLater, day2SAndP500Price)
			if sellDay == nil {
				result = resultNoData
				break
			}

			inflation, ok := year2InflationRate[day.Year()]
			if !ok {
				result = resultNoData
				break
			}
			targetSaving = targetSaving * (1 + inflation)
			currMaxWithdrawRate = currMaxWithdrawRate * (1 + inflation)
			currMinWithdrawRate = currMinWithdrawRate * (1 + inflation)

			stockPriceRatio := currStockPrice / prevYearStockPrice
			dividendYield, ok := year2StockDividendYield[day.Year()]
			if !ok {
				result = resultNoData
				break
			}
			dividendYield *= (1 - *dividendTaxRate)
			saving = saving*stockPriceRatio + saving*dividendYield
			withdrawRate := currMaxWithdrawRate
			count := &currMaxWithdrawCount
			if saving < targetSaving {
				count = &currMinWithdrawCount
				withdrawRate = currMinWithdrawRate
			}
			*count++

			withdraw := withdrawRate * initSaving
			saving -= withdraw
			if saving <= 0 {
				result = resultFail
				break
			}

			prevYearStockPrice = currStockPrice
			day = aYearLater
		}

		if result == resultSuccess {
			successCount++
		} else if result == resultFail {
			failCount++
		}

		if result != resultNoData {
			maxWithdrawCount += currMaxWithdrawCount
			minWithdrawCount += currMinWithdrawCount
		}
	}

	log.Println("successCount", successCount, " failCount", failCount, " pass rate", successCount/(successCount+failCount))
	log.Println("minWithdrawCount", minWithdrawCount, ", maxWithdrawCount", maxWithdrawCount, " min/total", minWithdrawCount/(minWithdrawCount+maxWithdrawCount))
}

func findPersonalConsumption(date time.Time, monthPersonalConsumption map[time.Time]float64) (float64, bool) {
	var target time.Time
	if date.Month() < 4 {
		target = time.Date(date.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	} else if date.Month() < 7 {
		target = time.Date(date.Year(), 4, 1, 0, 0, 0, 0, time.UTC)
	} else if date.Month() < 10 {
		target = time.Date(date.Year(), 7, 1, 0, 0, 0, 0, time.UTC)
	} else {
		target = time.Date(date.Year(), 10, 1, 0, 0, 0, 0, time.UTC)
	}
	v, ok := monthPersonalConsumption[target]
	return v, ok
}

func findSAndP500DayAndPrice(sAndP500Day time.Time, sAndP500History datePrice) (*time.Time, float64) {
	for {
		price, ok := sAndP500History.Day2Price[sAndP500Day]
		if ok {
			return &sAndP500Day, price
		}

		sAndP500Day = sAndP500Day.AddDate(0, 0, 1)
		if sAndP500Day.After(sAndP500History.LastDate) {
			return nil, 0
		}
	}
}

func printFirstN(dv datePrice, n int) {
	count := 0
	for k, v := range dv.Day2Price {
		log.Println(k, v)
		count++
		if count == n {
			return
		}
	}
}

func readInflationRate(path string) map[int]float64 {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("open csv %s failed %+v", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	result := map[int]float64{}
	// skip header
	reader.Read()
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read csv %s %v", path, err)
		}

		year, err := strconv.Atoi(row[0])
		if err != nil {
			log.Fatalf("parse csv %v %v %v", path, err, row)
		}

		if row[13] == "" {
			break
		}
		value, err := strconv.ParseFloat(row[13], 64)
		if err != nil {
			log.Fatalf("parse csv %v %v %v", path, err, row)
		}

		result[year] = value / 100
	}

	return result
}

func readSAndP500File(path string) datePrice {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("open csv %s failed %v", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	history := datePrice{
		Day2Price: make(map[time.Time]float64),
	}
	// skip header
	reader.Read()
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read s&p500 csv %+v", err)
		}

		// row[0]:1950-01-03
		// row[4]:16.66
		t, err := time.Parse("2006-01-02", row[0])
		if err != nil {
			log.Fatalf("parse csv %s %v %v", path, err, row)
		}
		history.Day2Price[t], err = strconv.ParseFloat(row[4], 64)
		if err != nil {
			log.Fatalf("parse csv %s %v %v", path, err, row)
		}

		if history.FirstDate.IsZero() {
			history.FirstDate = t
		}
		history.LastDate = t
	}

	return history
}

func readPersonalConsumption(path string) map[time.Time]float64 {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("open csv %s failed %v", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	result := map[time.Time]float64{}
	// skip header
	reader.Read()
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read csv %s  %v %v", path, err, row)
		}

		// row[0]:1950-01-03
		// row[1]:1091
		t, err := time.Parse("2006-01-02", row[0])
		if err != nil {
			log.Fatalf("parse csv %s %v %v", path, err, row)
		}
		spend, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			log.Fatalf("parse csv %s %v %v", path, err, row)
		}

		result[t] = spend
	}

	return result
}

func readSAndP500DividendYield(path string) map[int]float64 {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("open csv %s failed %v", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	result := map[int]float64{}
	// skip header
	reader.Read()
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read csv %s %v", path, err)
		}

		// row[0] Dec 31, 2020
		// row[1]	1.58%
		year, err := strconv.Atoi(row[0][len(row[0])-4:])
		if err != nil {
			log.Fatalf("parse csv %v %v", err, row)
		}
		value, err := strconv.ParseFloat(row[1][:len(row[1])-1], 64)
		if err != nil {
			log.Fatalf("parse csv %v %v", err, row)
		}
		result[year] = value / 100
	}

	return result
}

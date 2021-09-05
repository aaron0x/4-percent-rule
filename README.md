# Examine 4% rule by historical data

## Parameters
### 1. max withdraw rate (-max)
* e.g., -max 0.04

### 2. min withdraw rate (-min)
* e.g., -min 0.03

### 3. years to live (-y)
* e.g., -y 30

### 4. tax rate of dividend (-t)
* e.g., -t 0.3

## Flow
### 1. Set initial saving to
### personal consumption expenditures per capita / max withdraw rate.
### 2. For the frist time, withdraw
saving * max withdraw rate.
### 3. Compute next year's saving by stock price and dividend.
### 4. Compute target saving by inflation.
### 5. If saving is more than target saving, withdraw saving * max withdraw rate, otherwise, withdraw saving * min withdraw rate.
### 6. Repeat 3.-5. until years to live pass or there is no saving left.

## Output
### 1. successful days and failed days, also the successful rate.
### 2. max withdraw count and min withdraw count, also the ratio.

# Example Usage
## go run main.go -max 0.03 -min 0.02 -y 30 -t 0.4
# DATA source:
## personal-consumption-expenditures-per-capita
https://fred.stlouisfed.org/series/A794RC0Q052SBEA
## US Inflation rate
https://www.usinflationcalculator.com/inflation/historical-inflation-rates/
## spx
https://stooq.com/q/d/?s=%5Espx&c=0
## S&P 500 Dividend Yield by Year
https://www.multpl.com/s-p-500-dividend-yield/table/by-year
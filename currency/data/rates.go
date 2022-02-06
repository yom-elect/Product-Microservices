package data

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/go-hclog"
)

type ExchangeRates struct {
	log hclog.Logger
	rates map[string]float64
}

func NewRates(l hclog.Logger) (*ExchangeRates, error) {
	er := &ExchangeRates{log: l, rates: map[string]float64{}}

	err := er.GetRates()

	return er, err
}

func (e *ExchangeRates) GetRate(base string, dest string) (float64, error) {
	br, ok := e.rates[base]
	if !ok {
		return 0,fmt.Errorf("rate not found for currency %s", base)
	}

	dr, ok := e.rates[dest]
	if !ok {
		return 0,fmt.Errorf("rate not found for currency %s", dest)
	}

	return dr / br , nil
}

func (e *ExchangeRates) GetRates() error {
	resp, err :=  http.DefaultClient.Get("https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist-90d.xml")
	if err != nil {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected a status of 200, but got %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	md := &Cubes{}
	xml.NewDecoder(resp.Body).Decode(&md)

	for _,c := range md.CubeData {
		r, err := strconv.ParseFloat(c.Rate, 64)
		if err != nil {
			return err
		}

		e.rates[c.Currency] = r
	}

	e.rates["EUR"] = 1

	return nil
}

type Cubes struct {
	CubeData []Cube `xml:"Cube>Cube>Cube"`
}

type Cube struct {
	Currency string `xml:"currency,attr"`
	Rate string `xml:"rate,attr"`
}
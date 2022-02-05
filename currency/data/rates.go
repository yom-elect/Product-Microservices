package data

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/go-hclog"
)

type ExchangeRate struct {
	log hclog.Logger
	rates map[string]float64
}

func NewRates(l hclog.Logger) (*ExchangeRate, error) {
	er := &ExchangeRate{log: l, rates: map[string]float64{}}

	err := er.getRates()

	return er, err
}

func (e *ExchangeRate) getRates() error {
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

	return nil
}

type Cubes struct {
	CubeData []Cube `xml:"Cube>Cube>Cube"`
}

type Cube struct {
	Currency string `xml:"currency,attr"`
	Rate string `xml:"rate,attr"`
}
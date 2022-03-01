package server

import (
	"context"
	"io"
	"time"

	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"product.com/product-microservice/currency/data"
	protos "product.com/product-microservice/currency/protos/currency"
)

// Currency is a gRPC server it implements the methods defined by the CurrencyServer interface
type Currency struct {
	rates *data.ExchangeRates
	log hclog.Logger
	subscriptions map[protos.Currency_SubscribeRatesServer][]*protos.RateRequest
}

// NewCurrency creates a new Currency server
func NewCurrency(r *data.ExchangeRates ,l hclog.Logger) *Currency {
	c := &Currency{r,l, make(map[protos.Currency_SubscribeRatesServer][]*protos.RateRequest)}
	go c.handleUpdates()

	return c
}

func (c *Currency) handleUpdates(){
	ru := c.rates.MonitorRates(5* time.Second)
		for range ru {
			c.log.Info("Got Updated rates")

			// loop over subscribed clients
			for k, v := range c.subscriptions {

				//loop over subscribed rates
				for _, rr := range v {
					r, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
					if err != nil {
						c.log.Error("Unable to get update rate", "base" ,rr.GetBase().String(), "destination", rr.GetDestination().String())
					}

					err = k.Send(
						&protos.StreamingRateResponse{
							Message: &protos.StreamingRateResponse_RateResponse{
								RateResponse: &protos.RateResponse{Base: rr.Base, Destination: rr.Destination, Rate: r},
							},
						},)
						
					if err != nil {
						c.log.Error("Unable to get update rate", "base" ,rr.GetBase().String(), "destination", rr.GetDestination().String())
					}
				}
			}


		}
}

// GetRate implements the CurrencyServer GetRate method and returns the currency exchange rate
// for the two given currencies.
func (c *Currency) GetRate(ctx context.Context, rr *protos.RateRequest) (*protos.RateResponse, error) {
	c.log.Info("Handle request for GetRate", "base", rr.GetBase(), "dest", rr.GetDestination())

	if rr.Destination == rr.Base {
		grpcError := status.Newf(
			codes.InvalidArgument,
			"Base currency %s cannot be same as the destination currency",
			rr.Base.String(),
			rr.Destination.String(),
		)

		err , wde := grpcError.WithDetails(rr)
		if wde != nil {
			return nil, wde
		}

		return nil, err.Err()
	}

	rate, err := c.rates.GetRate(rr.GetBase().String(), rr.GetDestination().String())
	if err != nil {
		return nil, err
	}

	return &protos.RateResponse{Base:rr.Base, Destination:rr.Destination, Rate: rate}, nil
}

func (c *Currency) SubscribeRates(src protos.Currency_SubscribeRatesServer) error {

	// handle client messages
	for {
		rr, err := src.Recv()
		if err == io.EOF {
			c.log.Info("Client has closed connection")
			break
		}

		if err != nil {
			c.log.Error("Unable to read from client", "error", err)
			return err
		}

		c.log.Info("Handle Client request", "request", rr)
		
		rrs, ok := c.subscriptions[src]
		if !ok {
			rrs = []*protos.RateRequest{}
		}

		// check that subscription does not exists
		var validationError *status.Status
		for _, v := range rrs {
			if v.Base == rr.Base && v.Destination == rr.Destination {
				// subscription exists return errors
				validationError= status.Newf(
					codes.AlreadyExists,
					"Unable to subscribe for currency as subscriptions already exits")
				
				// add the original request as Metadata
				validationError, err = validationError.WithDetails(rr)
				if err != nil {
					c.log.Error("Unable to add metadata to error", "error", err)
					break
				}

				break
			}
		}

		// if a validation error return error and continue processing
		if validationError != nil {
			src.Send(
				&protos.StreamingRateResponse{
					Message: &protos.StreamingRateResponse_Error{
						Error: validationError.Proto(),
					},
				},
			)
			continue
		}

		// all ok
		rrs = append(rrs, rr)
		c.subscriptions[src] = rrs
	}

	return nil
}
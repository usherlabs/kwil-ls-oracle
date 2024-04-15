package ingest_resolution

import (
	"context"
	"github.com/kwilteam/kwil-db/common"
	"math/big"

	"github.com/kwilteam/kwil-db/core/types/serialize"
	"github.com/kwilteam/kwil-db/extensions/resolutions"
)

var LogStoreIngestResolution = &IngestResolution{
	RefundThreshold:       big.NewRat(1, 3),
	ConfirmationThreshold: big.NewRat(2, 3),
	// aprox 1 hour, assuming 6s block time
	ExpirationPeriod: 600,
	ResolutionName:   "log_store_ingest",
}

// use golang's init function, which runs before main, to register the extension
// see more here: https://www.digitalocean.com/community/tutorials/understanding-init-in-go
func init() {
	// calling RegisterResolution will make the resolution available on startup
	err := resolutions.RegisterResolution(LogStoreIngestResolution.ResolutionName, LogStoreIngestResolution.GetResolutionConfig())
	if err != nil {
		panic(err)
	}
}

type IngestResolution struct {
	// RefundThreshold is the required vote percentage threshold for
	// all voters on a resolution to be refunded the gas costs
	// associated with voting. This allows for resolutions that have
	// not received enough votes to pass to refund gas to the voters
	// that have voted on the resolution. For a 1/3rds threshold,
	// >=1/3rds of the voters must vote for the resolution for
	// refunds to occur. If this threshold is not met, voters will
	// not be refunded when the resolution expires. The number must
	// be a fraction between 0 and 1. If this field is nil, it will
	// default to only refunding voters when the resolution is confirmed.
	RefundThreshold *big.Rat
	// ConfirmationThreshold is the percentage of votes that must be confirm votes for the resolution to be confirmed.
	ConfirmationThreshold *big.Rat
	// ExpirationPeriod is the number of blocks after which the resolution will expire if it has not been confirmed.
	ExpirationPeriod int64
	ResolutionName   string
}

func (r *IngestResolution) GetResolutionConfig() resolutions.ResolutionConfig {
	return resolutions.ResolutionConfig{
		RefundThreshold:       r.RefundThreshold,
		ConfirmationThreshold: r.ConfirmationThreshold,
		ExpirationPeriod:      r.ExpirationPeriod,
		ResolveFunc: func(ctx context.Context, app *common.App, resolution *resolutions.Resolution) error {
			// Unmarshal the resolution payload into an AccountCreditResolution
			var newData IngestDataResolution
			err := newData.UnmarshalBinary(resolution.Body)
			if err != nil {
				return err
			}
			// Ingest the data
			// This is where you would ingest the data using actions inside the app, if the action has the name of the resolution
			contracts, err := getDataSetsWithAction(ctx, app, r.ResolutionName)

			if err != nil {
				return err
			}

			for _, contract := range contracts {
				_, err := app.Engine.Procedure(ctx, app.DB, &common.ExecutionData{
					Dataset:   contract,
					Procedure: r.ResolutionName,
					Args:      []any{newData.Data},
					Signer:    resolution.Proposer,
					Caller:    string(resolution.Proposer),
				})
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func getDataSetsWithAction(ctx context.Context, app *common.App, action string) ([]string, error) {
	allContracts, err := app.Engine.ListDatasets(ctx, []byte{})
	if err != nil {
		return nil, err
	}

	var contracts []string
	for _, contract := range allContracts {
		schema, err := app.Engine.GetSchema(ctx, contract.DBID)
		if err != nil {
			return nil, err
		}

		allContractProcedures := schema.Procedures

		for _, contractProcedure := range allContractProcedures {
			if contractProcedure.Name == action {
				contracts = append(contracts, contract.DBID)
				break
			}
		}
	}
	return contracts, nil
}

type IngestDataResolution struct {
	// Data is the data to be ingested.
	Data []interface{}
}

// MarshalBinary marshals the resolution into a binary format.
func (r *IngestDataResolution) MarshalBinary() ([]byte, error) {
	return serialize.Encode(r)
}

// UnmarshalBinary unmarshals the resolution from a binary format.
func (r *IngestDataResolution) UnmarshalBinary(data []byte) error {
	return serialize.DecodeInto(data, r)
}

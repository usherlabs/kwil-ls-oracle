package ingest_resolution

import (
	"fmt"
	"github.com/kwilteam/kwil-db/core/types"
	"strings"
)

type ContractSelector struct {
	Owner string
	Name  string
}

func FilterSelectedContracts(selectors []ContractSelector, contracts []types.DatasetIdentifier) []types.DatasetIdentifier {
	var selectedContracts []types.DatasetIdentifier
	for _, contract := range contracts {
		if IsSelected(selectors, contract) {
			selectedContracts = append(selectedContracts, contract)
		}
	}
	return selectedContracts
}

func IsSelected(selectors []ContractSelector, contract types.DatasetIdentifier) bool {
	for _, selector := range selectors {
		isOwnerMatch := selector.Owner == "*" || selector.Owner == string(contract.Owner)
		isNameMatch := selector.Name == "*" || selector.Name == contract.Name
		if isOwnerMatch && isNameMatch {
			return true
		}
	}
	return false
}

// LookupSchemaToSelectors converts the lookup schemas to contract selectors
// lookup schemas are in the format of `owner/name`
func LookupSchemaToSelectors(lookupSchemas []string) []ContractSelector {
	var selectors []ContractSelector
	for _, schema := range lookupSchemas {
		// first slash divides owner and name
		// let's find the slash index
		slashIndex := strings.Index(schema, "/")
		if slashIndex == -1 {
			panic(fmt.Sprintf("invalid schema: %s", schema))
		}

		owner := schema[:slashIndex]
		name := schema[slashIndex+1:]

		selectors = append(selectors, ContractSelector{
			Owner: owner,
			Name:  name,
		})
	}
	return selectors

}

package tokenlist

import (
	"encoding/json"

	"github.com/blang/semver"
	"gitlab.com/mayachain/mayanode/common/tokenlist/ethtokens"
)

var (
	ethTokenListV93 EVMTokenList
	ethTokenListV95 EVMTokenList
)

func init() {
	if err := json.Unmarshal(ethtokens.ETHTokenListRawV93, &ethTokenListV93); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(ethtokens.ETHTokenListRawV95, &ethTokenListV95); err != nil {
		panic(err)
	}
}

func GetETHTokenList(version semver.Version) EVMTokenList {
	switch {
	case version.GTE(semver.MustParse("1.95.0")):
		return ethTokenListV95
	case version.GTE(semver.MustParse("1.93.0")):
		return ethTokenListV93
	default:
		return ethTokenListV93
	}
}

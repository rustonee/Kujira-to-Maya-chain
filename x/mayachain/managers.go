package mayachain

import (
	"errors"
	"fmt"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v2/modules/apps/transfer/keeper"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"

	"gitlab.com/mayachain/mayanode/common"
	"gitlab.com/mayachain/mayanode/common/cosmos"
	"gitlab.com/mayachain/mayanode/constants"
	"gitlab.com/mayachain/mayanode/x/mayachain/keeper"
	kv1 "gitlab.com/mayachain/mayanode/x/mayachain/keeper/v1"
	"gitlab.com/mayachain/mayanode/x/mayachain/types"
)

const (
	genesisBlockHeight = 1
)

// ErrNotEnoughToPayFee will happen when the emitted asset is not enough to pay for fee
var ErrNotEnoughToPayFee = errors.New("not enough asset to pay for fees")

// Manager is an interface to define all the required methods
type Manager interface {
	GetConstants() constants.ConstantValues
	GetVersion() semver.Version
	Keeper() keeper.Keeper
	GasMgr() GasManager
	EventMgr() EventManager
	TxOutStore() TxOutStore
	NetworkMgr() NetworkManager
	ValidatorMgr() ValidatorManager
	ObMgr() ObserverManager
	PoolMgr() PoolManager
	SwapQ() SwapQueue
	OrderBookMgr() OrderBook
	Slasher() Slasher
	YggManager() YggManager
}

// GasManager define all the methods required to manage gas
type GasManager interface {
	BeginBlock(mgr Manager)
	EndBlock(ctx cosmos.Context, keeper keeper.Keeper, eventManager EventManager)
	AddGasAsset(gas common.Gas, increaseTxCount bool)
	ProcessGas(ctx cosmos.Context, keeper keeper.Keeper)
	GetGas() common.Gas
	GetFee(ctx cosmos.Context, chain common.Chain, asset common.Asset) cosmos.Uint
	GetMaxGas(ctx cosmos.Context, chain common.Chain) (common.Coin, error)
	GetGasRate(ctx cosmos.Context, chain common.Chain) cosmos.Uint
	SubGas(gas common.Gas)
}

// EventManager define methods need to be support to manage events
type EventManager interface {
	EmitEvent(ctx cosmos.Context, evt EmitEventItem) error
	EmitGasEvent(ctx cosmos.Context, gasEvent *EventGas) error
	EmitSwapEvent(ctx cosmos.Context, swap *EventSwap) error
	EmitFeeEvent(ctx cosmos.Context, feeEvent *EventFee) error
	EmitBondEvent(ctx cosmos.Context, mgr Manager, asset common.Asset, amount cosmos.Uint, bondType types.BondType, txIn common.Tx) error
}

// TxOutStore define the method required for TxOutStore
type TxOutStore interface {
	EndBlock(ctx cosmos.Context, mgr Manager) error
	GetBlockOut(ctx cosmos.Context) (*TxOut, error)
	ClearOutboundItems(ctx cosmos.Context)
	GetOutboundItems(ctx cosmos.Context) ([]TxOutItem, error)
	TryAddTxOutItem(ctx cosmos.Context, mgr Manager, toi TxOutItem, minOut cosmos.Uint) (bool, error)
	UnSafeAddTxOutItem(ctx cosmos.Context, mgr Manager, toi TxOutItem) error
	GetOutboundItemByToAddress(cosmos.Context, common.Address) []TxOutItem
	CalcTxOutHeight(cosmos.Context, semver.Version, TxOutItem) (int64, error)
}

// ObserverManager define the method to manage observes
type ObserverManager interface {
	BeginBlock()
	EndBlock(ctx cosmos.Context, keeper keeper.Keeper)
	AppendObserver(chain common.Chain, addrs []cosmos.AccAddress)
	List() []cosmos.AccAddress
}

// ValidatorManager define the method to manage validators
type ValidatorManager interface {
	BeginBlock(ctx cosmos.Context, constAccessor constants.ConstantValues, existingValidators []string) error
	EndBlock(ctx cosmos.Context, mgr Manager) []abci.ValidatorUpdate
	RequestYggReturn(ctx cosmos.Context, node NodeAccount, mgr Manager) error
	processRagnarok(ctx cosmos.Context, mgr Manager) error
	NodeAccountPreflightCheck(ctx cosmos.Context, na NodeAccount, constAccessor constants.ConstantValues) (NodeStatus, error)
}

// NetworkManager interface define the contract of network Manager
type NetworkManager interface {
	TriggerKeygen(ctx cosmos.Context, nas NodeAccounts) error
	RotateVault(ctx cosmos.Context, vault Vault) error
	EndBlock(ctx cosmos.Context, mgr Manager) error
	UpdateNetwork(ctx cosmos.Context, constAccessor constants.ConstantValues, gasManager GasManager, eventMgr EventManager) error
	RecallChainFunds(ctx cosmos.Context, chain common.Chain, mgr Manager, excludeNode common.PubKeys) error
}

// PoolManager interface define the contract of PoolManager
type PoolManager interface {
	EndBlock(ctx cosmos.Context, mgr Manager) error
}

// SwapQueue interface define the contract of Swap Queue
type SwapQueue interface {
	EndBlock(ctx cosmos.Context, mgr Manager) error
}

// OrderBook interface define the contract of Order Book
type OrderBook interface {
	EndBlock(ctx cosmos.Context, mgr Manager) error
}

// Slasher define all the method to perform slash
type Slasher interface {
	BeginBlock(ctx cosmos.Context, req abci.RequestBeginBlock, constAccessor constants.ConstantValues)
	HandleDoubleSign(ctx cosmos.Context, addr crypto.Address, infractionHeight int64, constAccessor constants.ConstantValues) error
	LackObserving(ctx cosmos.Context, constAccessor constants.ConstantValues) error
	LackSigning(ctx cosmos.Context, mgr Manager) error
	SlashVaultToLP(ctx cosmos.Context, vaultPK common.PubKey, coins common.Coins, mgr Manager, subsidize bool) error
	SlashNodeAccountLP(ctx cosmos.Context, na NodeAccount, slash cosmos.Uint) (cosmos.Uint, []PoolAmt, error)
	IncSlashPoints(ctx cosmos.Context, point int64, addresses ...cosmos.AccAddress)
	DecSlashPoints(ctx cosmos.Context, point int64, addresses ...cosmos.AccAddress)
}

// YggManager define method to fund yggdrasil
type YggManager interface {
	Fund(ctx cosmos.Context, mgr Manager) error
}

// Mgrs is an implementation of Manager interface
type Mgrs struct {
	currentVersion semver.Version
	constAccessor  constants.ConstantValues
	gasMgr         GasManager
	eventMgr       EventManager
	txOutStore     TxOutStore
	networkMgr     NetworkManager
	validatorMgr   ValidatorManager
	obMgr          ObserverManager
	poolMgr        PoolManager
	swapQ          SwapQueue
	orderBook      OrderBook
	slasher        Slasher
	yggManager     YggManager

	K                 keeper.Keeper
	cdc               codec.BinaryCodec
	coinKeeper        bankkeeper.Keeper
	accountKeeper     authkeeper.AccountKeeper
	ibcTransferkeeper ibctransferkeeper.Keeper
	storeKey          cosmos.StoreKey
}

// NewManagers  create a new Manager
func NewManagers(keeper keeper.Keeper, cdc codec.BinaryCodec, coinKeeper bankkeeper.Keeper, accountKeeper authkeeper.AccountKeeper, ibcTransferkeeper ibctransferkeeper.Keeper, storeKey cosmos.StoreKey) *Mgrs {
	return &Mgrs{
		K:                 keeper,
		cdc:               cdc,
		coinKeeper:        coinKeeper,
		accountKeeper:     accountKeeper,
		ibcTransferkeeper: ibcTransferkeeper,
		storeKey:          storeKey,
	}
}

func (mgr *Mgrs) GetVersion() semver.Version {
	return mgr.currentVersion
}

func (mgr *Mgrs) GetConstants() constants.ConstantValues {
	return mgr.constAccessor
}

// BeginBlock detect whether there are new version available, if it is available then create a new version of Mgr
func (mgr *Mgrs) BeginBlock(ctx cosmos.Context) error {
	v := mgr.K.GetLowestActiveVersion(ctx)
	if v.Equals(mgr.GetVersion()) {
		return nil
	}
	// version is different , thus all the manager need to re-create
	mgr.currentVersion = v
	mgr.constAccessor = constants.GetConstantValues(v)
	var err error

	mgr.K, err = GetKeeper(v, mgr.cdc, mgr.coinKeeper, mgr.accountKeeper, mgr.ibcTransferkeeper, mgr.storeKey)
	if err != nil {
		return fmt.Errorf("fail to create keeper: %w", err)
	}

	if v.GTE(semver.MustParse("1.96.0")) { // TODO remove version checks after fork
		storedVer, hasStoredVer := mgr.Keeper().GetVersionWithCtx(ctx)
		if !hasStoredVer || v.GT(storedVer) {
			// store the version for contextual lookups if it has been upgraded
			mgr.Keeper().SetVersionWithCtx(ctx, v)
		}
	}

	mgr.gasMgr, err = GetGasManager(v, mgr.K)
	if err != nil {
		return fmt.Errorf("fail to create gas manager: %w", err)
	}
	mgr.eventMgr, err = GetEventManager(v)
	if err != nil {
		return fmt.Errorf("fail to get event manager: %w", err)
	}
	mgr.txOutStore, err = GetTxOutStore(v, mgr.K, mgr.eventMgr, mgr.gasMgr)
	if err != nil {
		return fmt.Errorf("fail to get tx out store: %w", err)
	}

	mgr.networkMgr, err = GetNetworkManager(v, mgr.K, mgr.txOutStore, mgr.eventMgr)
	if err != nil {
		return fmt.Errorf("fail to get vault manager: %w", err)
	}

	mgr.poolMgr, err = GetPoolManager(v, mgr.K)
	if err != nil {
		return fmt.Errorf("fail to get pool manager: %w", err)
	}

	mgr.validatorMgr, err = GetValidatorManager(v, mgr.K, mgr.networkMgr, mgr.txOutStore, mgr.eventMgr)
	if err != nil {
		return fmt.Errorf("fail to get validator manager: %w", err)
	}

	mgr.obMgr, err = GetObserverManager(v)
	if err != nil {
		return fmt.Errorf("fail to get observer manager: %w", err)
	}

	mgr.swapQ, err = GetSwapQueue(v, mgr.K)
	if err != nil {
		return fmt.Errorf("fail to create swap queue: %w", err)
	}

	mgr.orderBook, err = GetOrderBook(v, mgr.K)
	if err != nil {
		return fmt.Errorf("fail to create order book: %w", err)
	}

	mgr.slasher, err = GetSlasher(v, mgr.K, mgr.eventMgr)
	if err != nil {
		return fmt.Errorf("fail to create swap queue: %w", err)
	}

	mgr.yggManager, err = GetYggManager(v, mgr.K)
	if err != nil {
		return fmt.Errorf("fail to create swap queue: %w", err)
	}
	return nil
}

// Keeper return Keeper
func (mgr *Mgrs) Keeper() keeper.Keeper { return mgr.K }

// GasMgr return GasManager
func (mgr *Mgrs) GasMgr() GasManager { return mgr.gasMgr }

// EventMgr return EventMgr
func (mgr *Mgrs) EventMgr() EventManager { return mgr.eventMgr }

// TxOutStore return an TxOutStore
func (mgr *Mgrs) TxOutStore() TxOutStore { return mgr.txOutStore }

// VaultMgr return a valid NetworkManager
func (mgr *Mgrs) NetworkMgr() NetworkManager { return mgr.networkMgr }

// PoolMgr return a valid PoolManager
func (mgr *Mgrs) PoolMgr() PoolManager { return mgr.poolMgr }

// ValidatorMgr return an implementation of ValidatorManager
func (mgr *Mgrs) ValidatorMgr() ValidatorManager { return mgr.validatorMgr }

// ObMgr return an implementation of ObserverManager
func (mgr *Mgrs) ObMgr() ObserverManager { return mgr.obMgr }

// SwapQ return an implementation of SwapQueue
func (mgr *Mgrs) SwapQ() SwapQueue { return mgr.swapQ }

// OrderBookMgr
func (mgr *Mgrs) OrderBookMgr() OrderBook { return mgr.orderBook }

// Slasher return an implementation of Slasher
func (mgr *Mgrs) Slasher() Slasher { return mgr.slasher }

// YggManager return an implementation of YggManager
func (mgr *Mgrs) YggManager() YggManager { return mgr.yggManager }

// GetKeeper return Keeper
func GetKeeper(version semver.Version, cdc codec.BinaryCodec, coinKeeper bankkeeper.Keeper, accountKeeper authkeeper.AccountKeeper, ibcTransferkeeper ibctransferkeeper.Keeper, storeKey cosmos.StoreKey) (keeper.Keeper, error) {
	if version.GTE(semver.MustParse("0.1.0")) {
		return kv1.NewKVStore(cdc, coinKeeper, accountKeeper, ibcTransferkeeper, storeKey, version), nil
	}
	return nil, errInvalidVersion
}

// GetGasManager return GasManager
func GetGasManager(version semver.Version, keeper keeper.Keeper) (GasManager, error) {
	constAcessor := constants.GetConstantValues(version)
	switch {
	case version.GTE(semver.MustParse("1.104.0")):
		return newGasMgrV104(constAcessor, keeper), nil
	case version.GTE(semver.MustParse("1.103.0")):
		return newGasMgrV103(constAcessor, keeper), nil
	case version.GTE(semver.MustParse("1.100.0")):
		return newGasMgrV100(constAcessor, keeper), nil
	case version.GTE(semver.MustParse("1.98.0")):
		return newGasMgrV98(constAcessor, keeper), nil
	case version.GTE(semver.MustParse("1.94.0")):
		return newGasMgrV94(constAcessor, keeper), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetEventManager will return an implementation of EventManager
func GetEventManager(version semver.Version) (EventManager, error) {
	if version.GTE(semver.MustParse("0.1.0")) {
		return newEventMgrV1(), nil
	}
	return nil, errInvalidVersion
}

// GetTxOutStore will return an implementation of the txout store that
func GetTxOutStore(version semver.Version, keeper keeper.Keeper, eventMgr EventManager, gasManager GasManager) (TxOutStore, error) {
	constAccessor := constants.GetConstantValues(version)
	switch {
	case version.GTE(semver.MustParse("1.104.0")):
		return newTxOutStorageV104(keeper, constAccessor, eventMgr, gasManager), nil
	case version.GTE(semver.MustParse("1.99.0")):
		return newTxOutStorageV97(keeper, constAccessor, eventMgr, gasManager), nil
	case version.GTE(semver.MustParse("1.95.0")):
		return newTxOutStorageV95(keeper, constAccessor, eventMgr, gasManager), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetNetworkManager  retrieve a NetworkManager that is compatible with the given version
func GetNetworkManager(version semver.Version, keeper keeper.Keeper, txOutStore TxOutStore, eventMgr EventManager) (NetworkManager, error) {
	switch {
	case version.GTE(semver.MustParse("1.102.0")):
		return newNetworkMgrV102(keeper, txOutStore, eventMgr), nil
	case version.GTE(semver.MustParse("1.96.0")):
		return newNetworkMgrV96(keeper, txOutStore, eventMgr), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetValidatorManager create a new instance of Validator Manager
func GetValidatorManager(version semver.Version, keeper keeper.Keeper, networkMgr NetworkManager, txOutStore TxOutStore, eventMgr EventManager) (ValidatorManager, error) {
	switch {
	case version.GTE(semver.MustParse("1.102.0")):
		return newValidatorMgrV102(keeper, networkMgr, txOutStore, eventMgr), nil
	case version.GTE(semver.MustParse("1.95.0")):
		return newValidatorMgrV95(keeper, networkMgr, txOutStore, eventMgr), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetObserverManager return an instance that implements ObserverManager interface
// when there is no version can match the given semver , it will return nil
func GetObserverManager(version semver.Version) (ObserverManager, error) {
	if version.GTE(semver.MustParse("0.1.0")) {
		return newObserverMgrV1(), nil
	}
	return nil, errInvalidVersion
}

// GetPoolManager return an implementation of PoolManager
func GetPoolManager(version semver.Version, keeper keeper.Keeper) (PoolManager, error) {
	switch {
	case version.GTE(semver.MustParse("1.95.0")):
		return newPoolMgrV95(keeper), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetSwapQueue retrieve a SwapQueue that is compatible with the given version
func GetSwapQueue(version semver.Version, keeper keeper.Keeper) (SwapQueue, error) {
	switch {
	case version.GTE(semver.MustParse("1.95.0")):
		return newSwapQv95(keeper), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetOrderBook retrieve a OrderBook that is compatible with the given version
func GetOrderBook(version semver.Version, keeper keeper.Keeper) (OrderBook, error) {
	switch {
	case version.GTE(semver.MustParse("0.1.0")):
		return newOrderBookV1(keeper), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetSlasher return an implementation of Slasher
func GetSlasher(version semver.Version, keeper keeper.Keeper, eventMgr EventManager) (Slasher, error) {
	switch {
	case version.GTE(semver.MustParse("1.105.0")):
		return newSlasherV105(keeper, eventMgr), nil
	case version.GTE(semver.MustParse("1.104.0")):
		return newSlasherV104(keeper, eventMgr), nil
	case version.GTE(semver.MustParse("1.102.0")):
		return newSlasherV102(keeper, eventMgr), nil
	case version.GTE(semver.MustParse("1.92.0")):
		return newSlasherV92(keeper, eventMgr), nil
	default:
		return nil, errInvalidVersion
	}
}

// GetYggManager return an implementation of YggManager
func GetYggManager(version semver.Version, keeper keeper.Keeper) (YggManager, error) {
	if version.GTE(semver.MustParse("0.79.0")) {
		return newYggMgrV79(keeper), nil
	}
	return nil, errInvalidVersion
}

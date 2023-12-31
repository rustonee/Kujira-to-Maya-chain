package types

import (
	. "gopkg.in/check.v1"

	"gitlab.com/mayachain/mayanode/common"
	"gitlab.com/mayachain/mayanode/common/cosmos"
)

type EventSuite struct{}

var _ = Suite(&EventSuite{})

func (s EventSuite) TestSwapEvent(c *C) {
	evt := NewEventSwap(
		common.BNBAsset,
		cosmos.NewUint(5),
		cosmos.NewUint(5),
		cosmos.NewUint(5),
		cosmos.ZeroUint(),
		GetRandomTx(),
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100)),
		cosmos.NewUint(5),
	)
	c.Check(evt.Type(), Equals, "swap")
	events, err := evt.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestAddLiqudityEvent(c *C) {
	evt := NewEventAddLiquidity(
		common.BNBAsset,
		cosmos.NewUint(5),
		GetRandomBaseAddress(),
		cosmos.NewUint(5),
		cosmos.NewUint(5),
		GetRandomTxHash(),
		GetRandomTxHash(),
		GetRandomBNBAddress(),
	)
	c.Check(evt.Type(), Equals, "add_liquidity")
	events, err := evt.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestWithdrawEvent(c *C) {
	evt := NewEventWithdraw(
		common.BNBAsset,
		cosmos.NewUint(6),
		5000,
		cosmos.NewDec(0),
		GetRandomTx(),
		cosmos.NewUint(100),
		cosmos.NewUint(100),
		cosmos.ZeroUint(),
	)
	c.Check(evt.Type(), Equals, "withdraw")
	events, err := evt.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestPool(c *C) {
	evt := NewEventPool(common.BNBAsset, PoolStatus_Available)
	c.Check(evt.Type(), Equals, "pool")
	c.Check(evt.Pool.String(), Equals, common.BNBAsset.String())
	c.Check(evt.Status.String(), Equals, PoolStatus_Available.String())
	events, err := evt.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestReward(c *C) {
	evt := NewEventRewards(cosmos.NewUint(300), []PoolAmt{
		{common.BNBAsset, 30},
		{common.BTCAsset, 40},
	})
	c.Check(evt.Type(), Equals, "rewards")
	c.Check(evt.BondReward.String(), Equals, "300")
	c.Assert(evt.PoolRewards, HasLen, 2)
	c.Check(evt.PoolRewards[0].Asset.Equals(common.BNBAsset), Equals, true)
	c.Check(evt.PoolRewards[0].Amount, Equals, int64(30))
	c.Check(evt.PoolRewards[1].Asset.Equals(common.BTCAsset), Equals, true)
	c.Check(evt.PoolRewards[1].Amount, Equals, int64(40))
	events, err := evt.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestSlashLiquidity(c *C) {
	na := GetRandomValidatorNode(NodeStatus_Active)
	evt := NewEventSlashLiquidity(na.NodeAddress, common.BNBAsset, GetRandomBaseAddress(), cosmos.NewUint(100))
	c.Check(evt.Type(), Equals, "slash_liquidity")
	c.Check(evt.Asset, Equals, common.BNBAsset)
	c.Assert(evt.LpUnits.Uint64(), Equals, cosmos.NewUint(100).Uint64())
	events, err := evt.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestSlash(c *C) {
	evt := NewEventSlash(common.BNBAsset, []PoolAmt{
		{common.BNBAsset, -20},
		{common.BaseAsset(), 30},
	})
	c.Check(evt.Type(), Equals, "slash")
	c.Check(evt.Pool, Equals, common.BNBAsset)
	c.Assert(evt.SlashAmount, HasLen, 2)
	c.Check(evt.SlashAmount[0].Asset, Equals, common.BNBAsset)
	c.Check(evt.SlashAmount[0].Amount, Equals, int64(-20))
	c.Check(evt.SlashAmount[1].Asset, Equals, common.BaseAsset())
	c.Check(evt.SlashAmount[1].Amount, Equals, int64(30))
	events, err := evt.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestEventGas(c *C) {
	eg := NewEventGas()
	c.Assert(eg, NotNil)
	eg.UpsertGasPool(GasPool{
		Asset:    common.BNBAsset,
		AssetAmt: cosmos.NewUint(1000),
		CacaoAmt: cosmos.ZeroUint(),
	})
	c.Assert(eg.Pools, HasLen, 1)
	c.Assert(eg.Pools[0].Asset, Equals, common.BNBAsset)
	c.Assert(eg.Pools[0].CacaoAmt.Equal(cosmos.ZeroUint()), Equals, true)
	c.Assert(eg.Pools[0].AssetAmt.Equal(cosmos.NewUint(1000)), Equals, true)

	eg.UpsertGasPool(GasPool{
		Asset:    common.BNBAsset,
		AssetAmt: cosmos.NewUint(1234),
		CacaoAmt: cosmos.NewUint(1024),
	})
	c.Assert(eg.Pools, HasLen, 1)
	c.Assert(eg.Pools[0].Asset, Equals, common.BNBAsset)
	c.Assert(eg.Pools[0].CacaoAmt.Equal(cosmos.NewUint(1024)), Equals, true)
	c.Assert(eg.Pools[0].AssetAmt.Equal(cosmos.NewUint(2234)), Equals, true)

	eg.UpsertGasPool(GasPool{
		Asset:    common.BTCAsset,
		AssetAmt: cosmos.NewUint(1024),
		CacaoAmt: cosmos.ZeroUint(),
	})
	c.Assert(eg.Pools, HasLen, 2)
	c.Assert(eg.Pools[1].Asset, Equals, common.BTCAsset)
	c.Assert(eg.Pools[1].AssetAmt.Equal(cosmos.NewUint(1024)), Equals, true)
	c.Assert(eg.Pools[1].CacaoAmt.Equal(cosmos.ZeroUint()), Equals, true)

	eg.UpsertGasPool(GasPool{
		Asset:    common.BTCAsset,
		AssetAmt: cosmos.ZeroUint(),
		CacaoAmt: cosmos.ZeroUint(),
	})

	c.Assert(eg.Pools, HasLen, 2)
	c.Assert(eg.Pools[1].Asset, Equals, common.BTCAsset)
	c.Assert(eg.Pools[1].AssetAmt.Equal(cosmos.NewUint(1024)), Equals, true)
	c.Assert(eg.Pools[1].CacaoAmt.Equal(cosmos.ZeroUint()), Equals, true)

	eg.UpsertGasPool(GasPool{
		Asset:    common.BTCAsset,
		AssetAmt: cosmos.ZeroUint(),
		CacaoAmt: cosmos.NewUint(3333),
	})

	c.Assert(eg.Pools, HasLen, 2)
	c.Assert(eg.Pools[1].Asset, Equals, common.BTCAsset)
	c.Assert(eg.Pools[1].AssetAmt.Equal(cosmos.NewUint(1024)), Equals, true)
	c.Assert(eg.Pools[1].CacaoAmt.Equal(cosmos.NewUint(3333)), Equals, true)
	events, err := eg.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (s EventSuite) TestEventFee(c *C) {
	event := NewEventFee(GetRandomTxHash(), common.Fee{
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		},
		PoolDeduct: cosmos.NewUint(1023),
	}, cosmos.NewUint(5))
	c.Assert(event.Type(), Equals, FeeEventType)
	evts, err := event.Events()
	c.Assert(err, IsNil)
	c.Assert(evts, HasLen, 1)
}

func (s EventSuite) TestEventDonate(c *C) {
	e := NewEventDonate(common.BNBAsset, GetRandomTx())
	c.Check(e.Type(), Equals, "donate")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventRefund(c *C) {
	e := NewEventRefund(1, "refund", GetRandomTx(), common.NewFee(common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(100)),
	}, cosmos.ZeroUint()))
	c.Check(e.Type(), Equals, "refund")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventBond(c *C) {
	e := NewEventBond(cosmos.NewUint(100), BondType_bond_paid, GetRandomTx())
	c.Check(e.Type(), Equals, "bond")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventBondV105(c *C) {
	e := NewEventBondV105(common.BNBAsset, cosmos.NewUint(100), BondType_bond_paid, GetRandomTx())
	c.Check(e.Type(), Equals, "bond")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventReserve(c *C) {
	e := NewEventReserve(ReserveContributor{
		Address: GetRandomBNBAddress(),
		Amount:  cosmos.NewUint(100),
	}, GetRandomTx())
	c.Check(e.Type(), Equals, "reserve")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventErrata(c *C) {
	e := NewEventErrata(GetRandomTxHash(), PoolMods{
		NewPoolMod(common.BNBAsset, cosmos.NewUint(100), true, cosmos.NewUint(200), true),
	})
	c.Check(e.Type(), Equals, "errata")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventOutbound(c *C) {
	e := NewEventOutbound(GetRandomTxHash(), GetRandomTx())
	c.Check(e.Type(), Equals, "outbound")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventSlashPoint(c *C) {
	e := NewEventSlashPoint(GetRandomBech32Addr(), 100, "what ever")
	c.Check(e.Type(), Equals, "slash_points")
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

func (EventSuite) TestEventPoolStageCost(c *C) {
	e := NewEventPoolBalanceChanged(NewPoolMod(common.BTCAsset, cosmos.NewUint(100), false, cosmos.ZeroUint(), false), "test")
	c.Check(e.Type(), Equals, PoolBalanceChangeEventType)
	events, err := e.Events()
	c.Check(err, IsNil)
	c.Check(events, NotNil)
}

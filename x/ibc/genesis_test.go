package ibc_test

import (
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	connectiontypes "github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	ibctmtypes "github.com/cosmos/cosmos-sdk/x/ibc/07-tendermint/types"
	localhosttypes "github.com/cosmos/cosmos-sdk/x/ibc/09-localhost/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
	ibctesting "github.com/cosmos/cosmos-sdk/x/ibc/testing"
	"github.com/cosmos/cosmos-sdk/x/ibc/types"
)

func (suite *IBCTestSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expPass  bool
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.NewGenesisState(
					[]clienttypes.GenesisClientState{
						clienttypes.NewGenesisClientState(
							clientID, ibctmtypes.NewClientState(chainID, ibctmtypes.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs()),
						),
						clienttypes.NewGenesisClientState(
							clientexported.ClientTypeLocalHost, localhosttypes.NewClientState("chaindID", 10),
						),
					},
					[]clienttypes.ClientConsensusStates{
						clienttypes.NewClientConsensusStates(
							clientID,
							[]clientexported.ConsensusState{
								ibctmtypes.NewConsensusState(
									suite.header.Time, commitmenttypes.NewMerkleRoot(suite.header.AppHash), suite.header.GetHeight(), suite.header.NextValidatorsHash,
								),
							},
						),
					},
					true,
				),
				ConnectionGenesis: connectiontypes.NewGenesisState(
					[]connectiontypes.IdentifiedConnection{
						connectiontypes.NewIdentifiedConnection(connectionID, connectiontypes.NewConnectionEnd(connectiontypes.INIT, clientID, connectiontypes.NewCounterparty(clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))), []string{ibctesting.ConnectionVersion})),
					},
					[]connectiontypes.ConnectionPaths{
						connectiontypes.NewConnectionPaths(clientID, []string{host.ConnectionPath(connectionID)}),
					},
				),
				ChannelGenesis: channeltypes.NewGenesisState(
					[]channeltypes.IdentifiedChannel{
						channeltypes.NewIdentifiedChannel(
							port1, channel1, channeltypes.NewChannel(
								channeltypes.INIT, channelOrder,
								channeltypes.NewCounterparty(port2, channel2), []string{connectionID}, channelVersion,
							),
						),
					},
					[]channeltypes.PacketAckCommitment{
						channeltypes.NewPacketAckCommitment(port2, channel2, 1, []byte("ack")),
					},
					[]channeltypes.PacketAckCommitment{
						channeltypes.NewPacketAckCommitment(port1, channel1, 1, []byte("commit_hash")),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port1, channel1, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
				),
			},
			expPass: true,
		},
		{
			name: "invalid client genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.NewGenesisState(
					[]clienttypes.GenesisClientState{
						clienttypes.NewGenesisClientState(
							clientID, ibctmtypes.NewClientState(chainID, ibctmtypes.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs()),
						),
						clienttypes.NewGenesisClientState(
							clientexported.ClientTypeLocalHost, localhosttypes.NewClientState("(chaindID)", 0),
						),
					},
					nil,
					false,
				),
				ConnectionGenesis: connectiontypes.DefaultGenesisState(),
			},
			expPass: false,
		},
		{
			name: "invalid connection genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.DefaultGenesisState(),
				ConnectionGenesis: connectiontypes.NewGenesisState(
					[]connectiontypes.IdentifiedConnection{
						connectiontypes.NewIdentifiedConnection(connectionID, connectiontypes.NewConnectionEnd(connectiontypes.INIT, "(CLIENTIDONE)", connectiontypes.NewCounterparty(clientID, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))), []string{"1.0.0"})),
					},
					[]connectiontypes.ConnectionPaths{
						connectiontypes.NewConnectionPaths(clientID, []string{host.ConnectionPath(connectionID)}),
					},
				),
			},
			expPass: false,
		},
		{
			name: "invalid channel genesis",
			genState: &types.GenesisState{
				ClientGenesis:     clienttypes.DefaultGenesisState(),
				ConnectionGenesis: connectiontypes.DefaultGenesisState(),
				ChannelGenesis: channeltypes.GenesisState{
					Acknowledgements: []channeltypes.PacketAckCommitment{
						channeltypes.NewPacketAckCommitment("(portID)", channel1, 1, []byte("ack")),
					},
				},
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

func (suite *IBCTestSuite) TestInitGenesis() {
	testCases := []struct {
		name     string
		genState *types.GenesisState
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.NewGenesisState(
					[]clienttypes.GenesisClientState{
						clienttypes.NewGenesisClientState(
							clientID, ibctmtypes.NewClientState(chainID, ibctmtypes.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs()),
						),
						clienttypes.NewGenesisClientState(
							clientexported.ClientTypeLocalHost, localhosttypes.NewClientState("chaindID", 10),
						),
					},
					[]clienttypes.ClientConsensusStates{
						clienttypes.NewClientConsensusStates(
							clientID,
							[]clientexported.ConsensusState{
								ibctmtypes.NewConsensusState(
									suite.header.Time, commitmenttypes.NewMerkleRoot(suite.header.AppHash), suite.header.GetHeight(), suite.header.ValidatorSet.Hash(),
								),
							},
						),
					},
					true,
				),
				ConnectionGenesis: connectiontypes.NewGenesisState(
					[]connectiontypes.IdentifiedConnection{
						connectiontypes.NewIdentifiedConnection(connectionID, connectiontypes.NewConnectionEnd(connectiontypes.INIT, clientID, connectiontypes.NewCounterparty(clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))), []string{ibctesting.ConnectionVersion})),
					},
					[]connectiontypes.ConnectionPaths{
						connectiontypes.NewConnectionPaths(clientID, []string{host.ConnectionPath(connectionID)}),
					},
				),
				ChannelGenesis: channeltypes.NewGenesisState(
					[]channeltypes.IdentifiedChannel{
						channeltypes.NewIdentifiedChannel(
							port1, channel1, channeltypes.NewChannel(
								channeltypes.INIT, channelOrder,
								channeltypes.NewCounterparty(port2, channel2), []string{connectionID}, channelVersion,
							),
						),
					},
					[]channeltypes.PacketAckCommitment{
						channeltypes.NewPacketAckCommitment(port2, channel2, 1, []byte("ack")),
					},
					[]channeltypes.PacketAckCommitment{
						channeltypes.NewPacketAckCommitment(port1, channel1, 1, []byte("commit_hash")),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port1, channel1, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
				),
			},
		},
	}

	for _, tc := range testCases {
		app := simapp.Setup(false)

		suite.NotPanics(func() {
			ibc.InitGenesis(app.BaseApp.NewContext(false, abci.Header{Height: 1}), *app.IBCKeeper, true, tc.genState)
		})
	}
}

// TODO: HandlerTestSuite should replace IBCTestSuite
func (suite *HandlerTestSuite) TestExportGenesis() {
	testCases := []struct {
		msg      string
		malleate func()
	}{
		{
			"success",
			func() {
				// creates clients
				suite.coordinator.Setup(suite.chainA, suite.chainB)
				// create extra clients
				suite.coordinator.CreateClient(suite.chainA, suite.chainB, clientexported.Tendermint)
				suite.coordinator.CreateClient(suite.chainA, suite.chainB, clientexported.Tendermint)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()

			var gs *types.GenesisState
			suite.NotPanics(func() {
				gs = ibc.ExportGenesis(suite.chainA.GetContext(), *suite.chainA.App.IBCKeeper)
			})

			// init genesis based on export
			suite.NotPanics(func() {
				ibc.InitGenesis(suite.chainA.GetContext(), *suite.chainA.App.IBCKeeper, true, gs)
			})

			suite.NotPanics(func() {
				cdc := codec.NewProtoCodec(suite.chainA.App.InterfaceRegistry())
				genState := cdc.MustMarshalJSON(gs)
				cdc.MustUnmarshalJSON(genState, gs)
			})

			// init genesis based on marshal and unmarshal
			suite.NotPanics(func() {
				ibc.InitGenesis(suite.chainA.GetContext(), *suite.chainA.App.IBCKeeper, true, gs)
			})
		})
	}
}

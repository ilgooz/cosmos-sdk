package utils

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	clientutils "github.com/cosmos/cosmos-sdk/x/ibc/02-client/client/utils"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

// QueryConnection returns a connection end.
// If prove is true, it performs an ABCI store query in order to retrieve the merkle proof. Otherwise,
// it uses the gRPC query client.
func QueryConnection(
	clientCtx client.Context, connectionID string, prove bool,
) (*types.QueryConnectionResponse, error) {
	if prove {
		return queryConnectionABCI(clientCtx, connectionID)
	}

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryConnectionRequest{
		ConnectionId: connectionID,
	}

	return queryClient.Connection(context.Background(), req)
}

func queryConnectionABCI(clientCtx client.Context, connectionID string) (*types.QueryConnectionResponse, error) {
	req := abci.RequestQuery{
		Path:  "store/ibc/key",
		Data:  host.KeyConnection(connectionID),
		Prove: true,
	}

	res, err := clientCtx.QueryABCI(req)
	if err != nil {
		return nil, err
	}

	var connection types.ConnectionEnd
	if err := clientCtx.LegacyAmino.UnmarshalBinaryBare(res.Value, &connection); err != nil {
		return nil, err
	}

	proofBz, err := clientCtx.LegacyAmino.MarshalBinaryBare(res.Proof)
	if err != nil {
		return nil, err
	}

	return types.NewQueryConnectionResponse(connectionID, connection, proofBz, res.Height), nil
}

// QueryClientConnections queries the connection paths registered for a particular client.
// If prove is true, it performs an ABCI store query in order to retrieve the merkle proof. Otherwise,
// it uses the gRPC query client.
func QueryClientConnections(
	clientCtx client.Context, clientID string, prove bool,
) (*types.QueryClientConnectionsResponse, error) {
	if prove {
		return queryClientConnectionsABCI(clientCtx, clientID)
	}

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryClientConnectionsRequest{
		ClientId: clientID,
	}

	return queryClient.ClientConnections(context.Background(), req)
}

func queryClientConnectionsABCI(clientCtx client.Context, clientID string) (*types.QueryClientConnectionsResponse, error) {
	req := abci.RequestQuery{
		Path:  "store/ibc/key",
		Data:  host.KeyClientConnections(clientID),
		Prove: true,
	}

	res, err := clientCtx.QueryABCI(req)
	if err != nil {
		return nil, err
	}

	var paths []string
	if err := clientCtx.LegacyAmino.UnmarshalBinaryBare(res.Value, &paths); err != nil {
		return nil, err
	}

	proofBz, err := clientCtx.LegacyAmino.MarshalBinaryBare(res.Proof)
	if err != nil {
		return nil, err
	}

	return types.NewQueryClientConnectionsResponse(clientID, paths, proofBz, res.Height), nil
}

// QueryConnectionClientState returns the ClientState of a connection end. If
// prove is true, it performs an ABCI store query in order to retrieve the
// merkle proof. Otherwise, it uses the gRPC query client.
func QueryConnectionClientState(
	clientCtx client.Context, connectionID string, prove bool,
) (*types.QueryConnectionClientStateResponse, error) {

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryConnectionClientStateRequest{
		ConnectionId: connectionID,
	}

	res, err := queryClient.ConnectionClientState(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if prove {
		clientState, proof, proofHeight, err := clientutils.QueryClientStateABCI(clientCtx, res.IdentifiedClientState.Id)
		if err != nil {
			return nil, err
		}

		// use client state returned from ABCI query in case query height differs
		identifiedClientState := clienttypes.NewIdentifiedClientState(res.IdentifiedClientState.Id, clientState)
		res = types.NewQueryConnectionClientStateResponse(identifiedClientState, proof, int64(proofHeight))
	}

	return res, nil
}

// QueryConnectionConsensusState returns the ConsensusState of a connection end. If
// prove is true, it performs an ABCI store query in order to retrieve the
// merkle proof. Otherwise, it uses the gRPC query client.
func QueryConnectionConsensusState(
	clientCtx client.Context, connectionID string, height uint64, prove bool,
) (*types.QueryConnectionConsensusStateResponse, error) {

	queryClient := types.NewQueryClient(clientCtx)
	req := &types.QueryConnectionConsensusStateRequest{
		ConnectionId: connectionID,
		Height:       height,
	}

	res, err := queryClient.ConnectionConsensusState(context.Background(), req)
	if err != nil {
		return nil, err
	}

	consensusState, err := clienttypes.UnpackConsensusState(res.ConsensusState)
	if err != nil {
		return nil, err
	}

	if prove {
		consensusState, proof, proofHeight, err := clientutils.QueryConsensusStateABCI(clientCtx, res.ClientId, consensusState.GetHeight())
		if err != nil {
			return nil, err
		}

		// use consensus state returned from ABCI query in case query height differs
		anyConsensusState, err := clienttypes.PackConsensusState(consensusState)
		if err != nil {
			return nil, err
		}

		res = types.NewQueryConnectionConsensusStateResponse(res.ClientId, anyConsensusState, consensusState.GetHeight(), proof, int64(proofHeight))
	}

	return res, nil
}

// ParsePrefix unmarshals an cmd input argument from a JSON string to a commitment
// Prefix. If the input is not a JSON, it looks for a path to the JSON file.
func ParsePrefix(cdc *codec.LegacyAmino, arg string) (commitmenttypes.MerklePrefix, error) {
	var prefix commitmenttypes.MerklePrefix
	if err := cdc.UnmarshalJSON([]byte(arg), &prefix); err != nil {
		// check for file path if JSON input is not provided
		contents, err := ioutil.ReadFile(arg)
		if err != nil {
			return commitmenttypes.MerklePrefix{}, errors.New("neither JSON input nor path to .json file were provided")
		}
		if err := cdc.UnmarshalJSON(contents, &prefix); err != nil {
			return commitmenttypes.MerklePrefix{}, errors.Wrap(err, "error unmarshalling commitment prefix")
		}
	}
	return prefix, nil
}

// ParseProof unmarshals a cmd input argument from a JSON string to a commitment
// Proof. If the input is not a JSON, it looks for a path to the JSON file. It
// then marshals the commitment proof into a proto encoded byte array.
func ParseProof(cdc *codec.LegacyAmino, arg string) ([]byte, error) {
	var merkleProof commitmenttypes.MerkleProof
	if err := cdc.UnmarshalJSON([]byte(arg), &merkleProof); err != nil {
		// check for file path if JSON input is not provided
		contents, err := ioutil.ReadFile(arg)
		if err != nil {
			return nil, errors.New("neither JSON input nor path to .json file were provided")
		}
		if err := cdc.UnmarshalJSON(contents, &merkleProof); err != nil {
			return nil, fmt.Errorf("error unmarshalling commitment proof: %w", err)
		}
	}

	return cdc.MarshalBinaryBare(&merkleProof)
}

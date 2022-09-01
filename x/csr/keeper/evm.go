package keeper

import (
	"github.com/Canto-Network/Canto/v2/x/csr/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// function to deploy an arbitrary smart-contract, takes as argument, the compiled
// contract object, as well as an arbitrary length array of arguments to the deployments
// deploys the contract from the module account
func (k Keeper) DeployContract(
	ctx sdk.Context,
	contract evmtypes.CompiledContract,
	args ...interface{},
) (common.Address, error) {
	// pack constructor arguments according to compiled contract's abi
	// method name is nil in this case, we are calling the constructor
	ctorArgs, err := contract.ABI.Pack("", args)
	if err != nil {
		return common.Address{}, sdkerrors.Wrapf(types.ErrContractDeployments,
			"::DeployContract: error packing data: %s", err.Error())
	}
	// pack method data into byte string, enough for bin and constructor arguments
	data := make([]byte, len(contract.Bin)+len(ctorArgs))
	// copy bin into data, and append to data the constructor arguments
	copy(data[:len(contract.Bin)], contract.Bin)
	// copy constructor args last
	copy(data[len(contract.Bin):], ctorArgs)
	// retrieve sequence number first to derive address if not by CREATE2
	nonce, err := k.accountKeeper.GetSequence(ctx, types.ModuleAddress.Bytes())
	if err != nil {
		return common.Address{},
			sdkerrors.Wrapf(types.ErrContractDeployments,
				"::DeployContract: error retrieving nonce: %s", err.Error())
	}

	// deploy contract using erc20 callEVMWithData, applies contract deployments to
	// current stateDb
	_, err = k.erc20Keeper.CallEVMWithData(ctx, types.ModuleAddress, nil, data, true)
	if err != nil {
		return common.Address{},
			sdkerrors.Wrapf(types.ErrAddressDerivation,
				"::DeployContract: error retrieving nonce: %s", err.Error())
	}

	// determine how to derive contract address, is to be derived from nonce
	return crypto.CreateAddress(types.ModuleAddress, nonce), nil
}

// function to interact with a contract once it is deployed, requires function signature,
// as well as arguments, pass pointer of argument type to CallMethod, and returned value from call is returned
func (k Keeper) CallMethod(
	ctx sdk.Context,
	method string,
	contract evmtypes.CompiledContract,
	ret *interface{},
	args ...interface{},
) (*evmtypes.MsgEthereumTxResponse, error) {
	// pack method args
	methodArgs, err := contract.ABI.Pack(method, args)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrContractDeployments, "::DeployContract: method call incorrect: %s", err.Error())
	}
	// call method
	resp, err := k.erc20Keeper.CallEVMWithData(ctx, types.ModuleAddress, nil, methodArgs, true)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrContractDeployments, "::CallMethod: error applying message: %s", err.Error())
	}
	// now unpack data into retType
	if err = contract.ABI.UnpackIntoInterface(ret, method, resp.Ret); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrAddressDerivation, "::CallMethod: error retrieving returned value: %s", err.Error())
	}

	return resp, nil
}

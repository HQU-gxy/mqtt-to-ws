package controller

import (
	sdk "github.com/33cn/chain33-sdk-go"
	"github.com/33cn/chain33-sdk-go/client"
	"github.com/33cn/chain33-sdk-go/crypto"
	"github.com/33cn/chain33-sdk-go/dapp/storage"
	"github.com/33cn/chain33-sdk-go/types"
)

// SaveToBlockchain
// From https://github.com/33cn/chain33-sdk-go/blob/master/dapp/storage/storage_test.go
// API https://github.com/33cn/chain33-sdk-go/tree/master/dapp/storage
func SaveToBlockchain(content []byte, privkey, url string) error {
	tx, err := storage.CreateContentStorageTx("", storage.OpCreate, "", content, "")
	if err != nil {
		return err
	}
	hexbytes, _ := types.FromHex(privkey)
	_, err = sdk.Sign(tx, hexbytes, crypto.SECP256K1, nil)
	// txhash := types.ToHexPrefix(sdk.Hash(tx))
	if err != nil {
		return err
	}
	jsonclient, err := client.NewJSONClient("", url)
	if err != nil {
		return err
	}
	// Is this necessary?
	signTx := types.ToHexPrefix(types.Encode(tx))
	_, err = jsonclient.SendTransaction(signTx)
	if err != nil {
		return err
	}
	return nil
}
